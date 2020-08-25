// Copyright (c) 2020 Red Hat, Inc.

package multiclusterhub

import (
	"context"

	operatorsv1 "github.com/open-cluster-management/multicloudhub-operator/pkg/apis/operator/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	uninstallList = func(m *operatorsv1.MultiClusterHub) []*unstructured.Unstructured {
		return []*unstructured.Unstructured{
			newUnstructured("ocm-validating-webhook", m.Namespace, "admissionregistration.k8s.io/v1beta1", "ValidatingWebhookConfiguration"),
			newUnstructured("managedclusterclaims.cluster.open-cluster-management.io", "", "apiextensions.k8s.io/v1beta1", "CustomResourceDefinition"),
		}
	}
)

func (r *ReconcileMultiClusterHub) ensureRemovalsGone(m *operatorsv1.MultiClusterHub) (*reconcile.Result, error) {
	removals := uninstallList(m)
	for i := range removals {
		rr, err := r.uninstall(removals[i])
		if rr != nil {
			return rr, err
		}
	}
	return nil, nil
}

func newUnstructured(name, namespace, apiVersion, kind string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}

func (r *ReconcileMultiClusterHub) uninstall(u *unstructured.Unstructured) (*reconcile.Result, error) {
	obLog := log.WithValues("Namespace", u.GetNamespace(), "Name", u.GetName(), "Kind", u.GetKind())

	found := u.NewEmptyInstance()
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name:      u.GetName(),
		Namespace: u.GetNamespace(),
	}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		// Error that isn't due to the resource not existing
		obLog.Error(err, "Failed to get subscription")
		return &reconcile.Result{}, err
	}

	err = r.client.Delete(context.TODO(), found)
	if err != nil {
		obLog.Error(err, "Failed to delete object")
		return &reconcile.Result{}, err
	}
	obLog.Info("Deleted instance")
	return nil, nil
}
