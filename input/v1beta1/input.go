// Package v1beta1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=tags.fn.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This isn't a custom resource, in the sense that we never install its CRD.
// It is a KRM-like object, so we generate a CRD to describe its schema.

// TODO: Add your input type here! It doesn't need to be called 'Input', you can
// rename it to anything you like.

// Input can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type Tags struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// AddTags are fields that will be added to every composed resource
	AddTags map[string]string `json:"addTags,omitempty"`

	// IgnoreTags is a map of tag keys to ignore
	IgnoreTags []string `json:"ignoreTags,omitempty"`

	// // Overwrite is whether existing tags are overwritten
	// Overwrite bool `json:"overwrite,omitempty"`
	// we are using maps.Copy() which overwrites by default
}
