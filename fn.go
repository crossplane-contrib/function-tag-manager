package main

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"dario.cat/mergo"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/stevendborrelli/function-tag-manager/input/v1beta1"
)

// Function returns whatever response you ask it to.
// not working
type Function struct {
	fnv1.FunctionRunnerServiceServer

	log logging.Logger
}

// IgnoreCrossplaneTags tags added by Crossplane automatically
// TODO: implement
var IgnoreCrossplaneTags = []string{"crossplane-kind", "crossplane-name", "crossplane-providerconfig"}

type IncludeAPIGroups map[string]bool
type SkipKinds = map[string]bool

// ResourceFilter are built-in resource filters. We only
// want to match tags on resources that support them
type ResourceFilter struct {
	IncludeAPIGroups IncludeAPIGroups
	SkipKinds        SkipKinds
}

// Resources to skip by Kind

var resourceFilter = ResourceFilter{
	// IncludeAPIGroups
	IncludeAPIGroups: IncludeAPIGroups{
		"aws.upbound.io":   true,
		"azure.upbound.io": true,
		"gcp.upbound.io":   true,
	},
	// Resources to skip by Kind
	SkipKinds: SkipKinds{
		"Provider":                true,
		"ProviderConfig":          true,
		"DeploymentRuntimeConfig": true,
	},
}

// IgnoreResourceAnnotation set this label to `True` or `true` do disable
// this function managing the resource
const IgnoreResourceLabel = "tag-manager.crossplane.io/ignore-resource"

// TagUpdater contains tags that are to be updated on a Desired Composed Resource
type TagUpdater struct {
	// Replace the tag values on the Desired Composed Resource will be overwritten if the keys match
	Replace v1beta1.Tags
	// Retain the tag values on the Desired Composed Resource if the keys match
	Retain v1beta1.Tags
}

// RunFunction runs the Function
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running Function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	in := &v1beta1.ManagedTags{}
	if err := request.GetInput(req, in); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	oxr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composite resource from %T", req))
		return rsp, nil
	}

	f.log.WithValues(
		"xr-apiversion", oxr.Resource.GetAPIVersion(),
		"xr-kind", oxr.Resource.GetKind(),
		"xr-name", oxr.Resource.GetName(),
	)

	// Process all the AddTags into 2 groups based on Policy: Replace or Retain
	// we also need to resolve any tags coming from a fieldpath
	additionalTags := f.ResolveAddTags(in.AddTags, oxr)

	// The composed resources that actually exist.
	observedComposed, err := request.GetObservedComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composed resources from %T", req))
		return rsp, nil
	}

	// The composed resources desired by any previous Functions in the pipeline.
	desiredComposed, err := request.GetDesiredComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get desired composed resources from %T", req))
		return rsp, nil
	}

	for name, desired := range desiredComposed {
		if IgnoreResource(desired) {
			f.log.Debug("skipping resource due to label", string(name), desired.Resource.GroupVersionKind().String())
			continue
		}
		if FilterResourceByGroupKind(desired, resourceFilter) {
			f.log.Debug("skipping resource due to GroupKind filter", string(name), desired.Resource.GroupVersionKind().String())
			continue
		}
		err := AddTags(desired, additionalTags)
		if err != nil {
			f.log.Debug("error adding tags", string(name), err.Error())
		}

		// Ignore tags only if there is an existing composed resource
		observed, ok := observedComposed[name]
		if ok {
			IgnoreTags(desired, observed, in.IgnoreTags)
		}
	}
	// For each of the ignore tags we add the tag to the desired state

	// Add additional tags to the desired state

	// for _, resource := range observedComposed {

	// 	var tags map[string]string
	// 	if err := fieldpath.Pave(resource.Resource.Object).GetValueInto("status.atProvider.tags", &tags); err != nil {
	// 		fmt.Println("No tags found")
	// 	}
	// 	for _, dk := range IgnoreCrossplaneTags {
	// 		delete(tags, dk)
	// 	}
	// 	fmt.Println(tags)
	// }

	if err := response.SetDesiredComposedResources(rsp, desiredComposed); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composed resources in %T", rsp))
		return rsp, nil
	}

	response.Normalf(rsp, "I was run with input")

	return rsp, nil
}

// FilterResource returns true if a resource should be skipped by Group or Kind
// by default resources are skipped
func FilterResourceByGroupKind(desired *resource.DesiredComposed, filter ResourceFilter) bool {
	if _, ok := filter.SkipKinds[desired.Resource.GetKind()]; ok {
		return true
	}

	// Filter out desired objects that are not a Managed Resource by looking for a forProvider field
	var forProvider map[string]any
	if err := fieldpath.Pave(desired.Resource.Object).GetValueInto("spec.forProvider", &forProvider); err != nil {
		return true
	}

	apiGroup := strings.Split(desired.Resource.GetAPIVersion(), "/")[0]
	for k := range filter.IncludeAPIGroups {
		if strings.Contains(apiGroup, k) {
			return false
		}
	}
	// Filter out any remaining resources
	return true
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

// Adds tags to a Desired Composed Resource
func AddTags(desired *resource.DesiredComposed, tu TagUpdater) error {
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

func IgnoreTags(desired *resource.DesiredComposed, observed resource.ObservedComposed, tags []v1beta1.IgnoreTag) {

}

// IgnoreResource whether this resource has a label set to ignore
func IgnoreResource(dc *resource.DesiredComposed) bool {
	if dc == nil {
		return true
	}
	var labels map[string]any
	err := fieldpath.Pave(dc.Resource.Object).GetValueInto("metadata.labels", &labels)
	if err != nil {
		return false
	} else {
		val, ok := labels[IgnoreResourceLabel].(string)
		if ok && strings.ToLower(val) == "true" {
			return true
		}
	}
	return false
}
