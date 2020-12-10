// Copyright (c) 2020 Red Hat, Inc.

package multiclusterhub

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/storage"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	syaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

func getActionConfig(namespace string) (*action.Configuration, error) {
	_ = new(action.Configuration)

	actionConfig := new(action.Configuration)
	var kubeConfig *genericclioptions.ConfigFlags
	// Create the rest config instance with ServiceAccount values loaded in them
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// Create the ConfigFlags struct instance with initialized values from ServiceAccount
	kubeConfig = genericclioptions.NewConfigFlags(false)
	kubeConfig.APIServer = &config.Host
	kubeConfig.BearerToken = &config.BearerToken
	kubeConfig.CAFile = &config.CAFile
	kubeConfig.Namespace = &namespace
	if err := actionConfig.Init(kubeConfig, namespace, os.Getenv("HELM_DRIVER"), log.Info); err != nil {
		return nil, err
	}
	return actionConfig, nil
}

func updateHelmReleases() {
	actionConfig, err := getActionConfig("open-cluster-management")
	listAction := action.NewList(actionConfig)
	releases, err := listAction.Run()
	if err != nil {
		log.Info(err.Error())
	}
	for _, release := range releases {
		log.Info("Release: " + release.Name + " Status: " + release.Info.Status.String())
		secretName := makeKey(release.Name, release.Version)
		newManifest, changed, err := injectKeepAnnotations(release.Manifest)
		if err != nil {
			log.Error(err, "issue modifying manifest", "Release", release.Name)
		}
		if !changed {
			continue
		}
		release.Manifest = newManifest
		log.Info("Release needs updating", "Release", release.Name)
		err = actionConfig.Releases.Driver.Update(secretName, release)
		if err != nil {
			log.Error(err, "issue updating helmrelease storage", "Release", release.Name)
		}
	}
}

// from https://github.com/helm/helm/blob/master/pkg/storage/storage.go
func makeKey(rlsname string, version int) string {
	return fmt.Sprintf("%s.%s.v%d", storage.HelmStorageType, rlsname, version)
}

var sep = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

// injectKeepAnnotations will add helm keep annotations to CRDs in the release manifest and return
// the new manifest along with whether or not changes were made
func injectKeepAnnotations(manifest string) (string, bool, error) {
	// Making sure that any extra whitespace in YAML stream doesn't interfere in splitting documents correctly.
	tmpManifest := strings.TrimSpace(manifest)
	docs := sep.Split(tmpManifest, -1)
	changed := false

	for _, yamlString := range docs {
		obj := &unstructured.Unstructured{}

		// decode YAML into unstructured.Unstructured
		dec := syaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		_, _, err := dec.Decode([]byte(yamlString), nil, obj)
		if err != nil {
			// log.Printf("Error deserializing: %s", yamlString)
			continue
		}

		if obj.GetKind() != "CustomResourceDefinition" {
			continue
		}

		_, ok := obj.GetAnnotations()[kube.ResourcePolicyAnno]
		if ok {
			// log.Info((fmt.Sprintf("CRD %s already has keep policy", obj.GetName())))
			continue
		}

		log.Info((fmt.Sprintf("CRD %s is missing a keep policy", obj.GetName())))
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[kube.ResourcePolicyAnno] = kube.KeepPolicy
		obj.SetAnnotations(annotations)
		changed = true

		// remarshal to get updated string
		b, err := yaml.Marshal(obj.Object)
		if err != nil {
			return "", false, err
		}

		// Replace entire resource string segment with annotated resource
		manifest = strings.Replace(manifest, yamlString, string(b), -1)
	}
	return manifest, changed, nil
}
