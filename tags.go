package main

import (
	"fmt"

	"dario.cat/mergo"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/stevendborrelli/function-tag-manager/input/v1beta1"
)

// IgnoreCrossplaneTags tags added by Crossplane automatically
// TODO: implement
var IgnoreCrossplaneTags = []string{"crossplane-kind", "crossplane-name", "crossplane-providerconfig"}

// IgnoreResourceAnnotation set this label to `True` or `true` to disable
// this function managing the resource's tags
const IgnoreResourceLabel = "tag-manager.fn.crossplane.io/ignore-resource"

// TagUpdater contains tags that are to be updated on a Desired Composed Resource
type TagUpdater struct {
	// Replace the tag values on the Desired Composed Resource will be overwritten if the keys match
	Replace v1beta1.Tags
	// Retain the tag values on the Desired Composed Resource if the keys match
	Retain v1beta1.Tags
}

// ResolveAddTags returns tags that will be Retained and Replaced
func (f *Function) ResolveAddTags(in []v1beta1.AddTag, oxr *resource.Composite) TagUpdater {
	tu := TagUpdater{}
	for _, at := range in {
		var tags v1beta1.Tags
		switch t := at.GetType(); t {
		case v1beta1.FromValue:
			_ = mergo.Map(&tags, at.Tags)
		case v1beta1.FromCompositeFieldPath: // resolve fields
			err := fieldpath.Pave(oxr.Resource.Object).GetValueInto(*at.FromFieldPath, &tags)
			if err != nil {
				f.log.Debug("Unable to read tags from Composite field: ", *at.FromFieldPath, err)
			}
		}
		if at.GetPolicy() == v1beta1.ExistingTagPolicyRetain {
			_ = mergo.Map(&tu.Retain, tags)
		} else {
			_ = mergo.Map(&tu.Replace, tags)
		}
	}
	return tu
}

// MergeTags merges tags to a Desired Composed Resource
func MergeTags(desired *resource.DesiredComposed, tu TagUpdater) error {
	var desiredTags v1beta1.Tags
	_ = fieldpath.Pave(desired.Resource.Object).GetValueInto("spec.forProvider.tags", &desiredTags)

	err := mergo.Map(&desiredTags, tu.Retain)
	if err != nil {
		return err
	}
	err = mergo.Map(&desiredTags, tu.Replace, mergo.WithOverride)
	if err != nil {
		return err
	}
	err = desired.Resource.SetValue("spec.forProvider.tags", desiredTags)

	return err
}

// ResolveImportTags returns tags that are populated from observed resources
func (f *Function) ResolveIgnoreTags(in []v1beta1.IgnoreTag, oxr *resource.Composite, observed *resource.ObservedComposed) *TagUpdater {
	tu := &TagUpdater{}
	if observed == nil {
		return tu
	}
	var observedTags v1beta1.Tags
	if err := fieldpath.Pave(observed.Resource.Object).GetValueInto("status.atProvider.tags", &observedTags); err != nil {
		//f.log.Debug("unable to fetch tags from observed resource", string(observed.Resource.GetName()), observed.Resource.GroupVersionKind().String())
		fmt.Println(err)
		return tu
	}
	for _, at := range in {
		var keys []string
		tags := make(map[string]string)
		switch t := at.GetType(); t {
		case v1beta1.FromValue:
			keys = at.Keys
		case v1beta1.FromCompositeFieldPath: // resolve fields
			err := fieldpath.Pave(oxr.Resource.Object).GetValueInto(*at.FromFieldPath, &keys)
			if err != nil {
				f.log.Debug("Unable to read tags from Composite field: ", *at.FromFieldPath, err)
			}
		}
		for _, k := range keys {
			if val, ok := observedTags[k]; ok {
				tags[k] = val
			}
		}

		if at.GetPolicy() == v1beta1.ExistingTagPolicyRetain {
			_ = mergo.Map(&tu.Retain, tags)
		} else {
			_ = mergo.Map(&tu.Replace, tags)
		}
	}
	return tu
}
