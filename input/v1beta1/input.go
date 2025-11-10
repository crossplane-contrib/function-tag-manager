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

// ManagedTags can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type ManagedTags struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// AddTags are fields that will be added to every composed resource.
	// +optional
	AddTags []AddTag `json:"addTags,omitempty"`

	// IgnoreTags is a list of tag keys to ignore if set on the
	// resource outside of Crossplane
	// +optional
	IgnoreTags IgnoreTags `json:"ignoreTags,omitempty"`

	// IgnoreTags is a list of tag keys to remove from the resource.
	// +optional
	RemoveTags RemoveTags `json:"removeTags,omitempty"`
}

// Tags contains a map tags.
type Tags map[string]string

// TagManagerType configures the source of the input tags.
type TagManagerType string

const (
	// FromCompositeFieldPath instructs the function to get tag settings from the Composite fieldpath.
	FromCompositeFieldPath TagManagerType = "FromCompositeFieldPath"
	// FromValue are static values set in the function's input.
	FromValue TagManagerType = "FromValue"
)

// TagManagerPolicy sets what happens when the tag exists in the resource.
type TagManagerPolicy string

const (
	// ExistingTagPolicyReplace replaces the desired value of a tag if the observed tag differs.
	ExistingTagPolicyReplace TagManagerPolicy = "Replace"
	// ExistingTagPolicyRetain retains the desired value of a tag if the observed tag differs.
	ExistingTagPolicyRetain TagManagerPolicy = "Retain"
)

// AddTag defines tags that should be added to every resource.
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

// IgnoreTag is a tag that is "ignored" by setting the desired value to the observed value.
type IgnoreTag struct {
	// Type determines where tag keys are sourced from. FromValue are inline
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

// IgnoreTags is a list of IgnoreTag settings.
type IgnoreTags []IgnoreTag

// RemoveTag is a tag that removed from the desired state.
type RemoveTag struct {
	// Type determines where tag keys are sourced from. FromValue are inline
	// to the composition. FromCompositeFieldPath fetches keys from a field in
	// the composite resource
	// +kubebuilder:validation:Enum=FromCompositeFieldPath;FromValue
	Type TagManagerType `json:"type"`

	// FromFieldPath if type is FromCompositeFieldPath, get keys to remove
	// from the field in the Composite (like spec.parameters.removeTags)
	// +optional
	FromFieldPath *string `json:"fromFieldPath,omitempty"`

	// Keys are tag keys to ignore for the FromValue type
	// +optional
	Keys []string `json:"keys,omitempty"`
}

// RemoveTags is an array of RemoveTag settings.
type RemoveTags []RemoveTag

// GetType returns the type of the managed tag.
func (a *AddTag) GetType() TagManagerType {
	if a == nil || a.Type == "" {
		return FromValue
	}

	return a.Type
}

// GetPolicy returns the add tag policy.
func (a *AddTag) GetPolicy() TagManagerPolicy {
	if a == nil || a.Type == "" {
		return ExistingTagPolicyReplace
	}

	return a.Policy
}

// GetType returns the type of the managed tag.
func (i *IgnoreTag) GetType() TagManagerType {
	if i == nil || i.Type == "" {
		return FromValue
	}

	return i.Type
}

// GetPolicy returns the tag policy.
func (i *IgnoreTag) GetPolicy() TagManagerPolicy {
	if i == nil || i.Type == "" {
		return ExistingTagPolicyReplace
	}

	return i.Policy
}

// GetType returns the type of the managed tag.
func (a *RemoveTag) GetType() TagManagerType {
	if a == nil || a.Type == "" {
		return FromValue
	}

	return a.Type
}
