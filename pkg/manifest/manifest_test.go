// Copyright (c) 2020 Red Hat, Inc.

package manifest

import (
	"os"
	"testing"

	operatorsv1beta1 "github.com/open-cluster-management/multicloudhub-operator/pkg/apis/operators/v1beta1"
)

func TestGetImageOverrideType(t *testing.T) {
	mch := &operatorsv1beta1.MultiClusterHub{}
	t.Run("Manifest format", func(t *testing.T) {
		want := Manifest
		if got := GetImageOverrideType(mch); got != want {
			t.Errorf("GetImageOverrideType() = %v, want %v", got, want)
		}
	})

	mch.Spec.Overrides.ImageTagSuffix = "foo"
	t.Run("Suffix format", func(t *testing.T) {
		want := Suffix
		if got := GetImageOverrideType(mch); got != want {
			t.Errorf("GetImageOverrideType() = %v, want %v", got, want)
		}
	})
}

func Test_readManifestFile(t *testing.T) {
	t.Run("Get manifest", func(t *testing.T) {
		os.Setenv(ManifestsPathEnvVar, "../../image-manifests")
		version := "2.0.0"
		_, err := readManifestFile(version)
		if err != nil {
			t.Errorf("readManifestFile() error = %v, wantErr %v", err, nil)
		}
	})

	t.Run("File not found", func(t *testing.T) {
		os.Setenv(ManifestsPathEnvVar, "../../image-manifests")
		version := "0.0.0"
		_, err := readManifestFile(version)
		if err == nil {
			t.Errorf("readManifestFile() did not return error")
		}
	})

	t.Run("Env var missing", func(t *testing.T) {
		os.Unsetenv(ManifestsPathEnvVar)
		version := "2.0.0"
		_, err := readManifestFile(version)
		if err == nil {
			t.Errorf("readManifestFile() did not return error")
		}
	})
}

func Test_buildFullImageReference(t *testing.T) {
	mi := ManifestImage{
		ImageKey:     "test_app",
		ImageName:    "test-app",
		ImageVersion: "2.0.0",
		ImageRemote:  "quay.io/open-cluster-management",
		ImageDigest:  "sha256:abc123",
	}
	mch := &operatorsv1beta1.MultiClusterHub{}

	mch1 := mch.DeepCopy()

	mch2 := mch.DeepCopy()
	mch2.Spec.Overrides.ImageRepository = "foo.io/bar"

	mch3 := mch.DeepCopy()
	mch3.Spec.Overrides.ImageTagSuffix = "baz"

	type args struct {
		mch *operatorsv1beta1.MultiClusterHub
		mi  ManifestImage
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Default (sha format)",
			args: args{mch1, mi},
			want: "quay.io/open-cluster-management/test-app@sha256:abc123",
		},
		{
			name: "Custom registry",
			args: args{mch2, mi},
			want: "foo.io/bar/test-app@sha256:abc123",
		},
		{
			name: "Use image suffix format",
			args: args{mch3, mi},
			want: "quay.io/open-cluster-management/test-app:2.0.0-baz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildFullImageReference(tt.args.mch, tt.args.mi); got != tt.want {
				t.Errorf("buildFullImageReference() = %v, want %v", got, tt.want)
			}
		})
	}
}