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
	AddTags []AddTag `json:"addTags,omitempty"`

	// IgnoreTags is a map of tag keys to ignore if set on the
	// resource outside of Crossplane
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
	Type          TagManagerType   `json:"type,omitempty"`
	FromFieldPath *string          `json:"fromFieldPath,omitempty"`
	Tags          Tags             `json:"tags,omitempty"`
	Policy        TagManagerPolicy `json:"policy,omitempty"`
}

type IgnoreTag struct {
	Type          TagManagerType   `json:"type"`
	FromFieldPath *string          `json:"fromFieldPath,omitempty"`
	Key           *string          `json:"key,omitempty"`
	Policy        TagManagerPolicy `json:"policy,omitempty"`
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

func GetKeys(i []IgnoreTag) []string {
	var keys []string
	for _, tag := range i {
		switch t := tag.GetType(); t {
		case FromValue:
			if tag.Key != nil {
				keys = append(keys, *tag.Key)
			}
		}

	}
	return keys
}
