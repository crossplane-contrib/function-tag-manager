// Package v1beta1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=tag-manager.fn.crossplane.io
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
type ManagedTags struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// AddTags are fields that will be added to every composed resource
	// +optional
	AddTags []AddTag `json:"addTags,omitempty"`

	// IgnoreTags is a map of tag keys to ignore if set on the
	// resource outside of Crossplane
	// +optional
	IgnoreTags IgnoreTags `json:"ignoreTags,omitempty"`
}

type Tags map[string]string

// TagManagerType configures where we get tags from
type TagManagerType string

const (
	FromCompositeFieldPath TagManagerType = "FromCompositeFieldPath"
	FromValue              TagManagerType = "FromValue"
)

// TagManagerPolicy sets what happens when the tag exists in the resource
type TagManagerPolicy string

const (
	// ExistingTagPolicyReplace replaces the desired value of a tag if the observed tag differs
	ExistingTagPolicyReplace TagManagerPolicy = "Replace"
	// ExistingTagPolicyReplace retains the desired value of a tag if the observed tag differs
	ExistingTagPolicyRetain TagManagerPolicy = "Retain"
)

type AddTag struct {

	// Type determines where tags are sourced from. FromValue are inline
	// to the composition. FromCompositeFieldPath fetches tags from a field in
	// the composite resource
	// +kubebuilder:validation:Enum=FromCompositeFieldPath;FromValue
	// +optional
	Type TagManagerType `json:"type,omitempty"`

	// FromFieldPath if type is FromCompositeFieldPath, get additional tags
	// from the field in the Composite (like spec.parameters.tags)
	// +optional
	FromFieldPath *string `json:"fromFieldPath,omitempty"`

	// Tags are tags to add to the resource in the form of a map
	// + optional
	Tags Tags `json:"tags,omitempty"`

	// Policy determines what tag value to use in case there already is a matching tag key
	// in the desired resource. Replace will overwrite the value, while Retain will keep
	// the existing value in the desired resource.
	// +kubebuilder:validation:Enum=Replace;Retain
	// +optional
	Policy TagManagerPolicy `json:"policy,omitempty"`
}

type IgnoreTag struct {
	// Type determines where tag keysare sourced from. FromValue are inline
	// to the composition. FromCompositeFieldPath fetches keys from a field in
	// the composite resource
	// +kubebuilder:validation:Enum=FromCompositeFieldPath;FromValue
	Type TagManagerType `json:"type"`

	// FromFieldPath if type is FromCompositeFieldPath, get keys to ignore
	// from the field in the Composite (like spec.parameters.ignoreTags)
	// +optional
	FromFieldPath *string `json:"fromFieldPath,omitempty"`

	// Keys are tag keys to ignore for the FromValue type
	// +optional
	Keys []string `json:"keys,omitempty"`

	// +kubebuilder:validation:Enum=Replace;Retain
	// +optional
	Policy TagManagerPolicy `json:"policy,omitempty"`
}
type IgnoreTags []IgnoreTag

func (a *AddTag) GetType() TagManagerType {
	if a == nil || a.Type == "" {
		return FromValue
	}
	return a.Type
}

func (a *AddTag) GetPolicy() TagManagerPolicy {
	if a == nil || a.Type == "" {
		return ExistingTagPolicyReplace
	}
	return a.Policy
}

func (i *IgnoreTag) GetType() TagManagerType {
	if i == nil || i.Type == "" {
		return FromValue
	}
	return i.Type
}

func (a *IgnoreTag) GetPolicy() TagManagerPolicy {
	if a == nil || a.Type == "" {
		return ExistingTagPolicyReplace
	}
	return a.Policy
}

func GetKeys(i []IgnoreTag) []string {
	var keys []string
	for _, tag := range i {
		switch t := tag.GetType(); t {
		case FromValue:
			if tag.Keys != nil {
				keys = append(keys, tag.Keys...)
			}
		}

	}
	return keys
}
