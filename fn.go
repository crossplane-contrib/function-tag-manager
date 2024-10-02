package main

import (
	"context"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/pkg/logging"

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

// RunFunction runs the Function
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running Function", "tag-manager", req.GetMeta().GetTag())

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
		"xr-d", oxr.Resource.GetAPIVersion(),
		"xr-kind", oxr.Resource.GetKind(),
		"xr-name", oxr.Resource.GetName(),
	)

	// Process all the AddTags into 2 groups based on Policy: Replace or Retain
	// we also need to resolve any tags coming from a Composite fieldpath
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
		desired.Resource.GetObjectKind()
		if IgnoreResource(desired) {
			f.log.Debug("skipping resource due to label", string(name), desired.Resource.GroupVersionKind().String())
			continue
		}
		if !SupportedManagedResource(desired, ManagedResourceFilter) {
			f.log.Debug("skipping resource that doesn't support tags", string(name), desired.Resource.GroupVersionKind().String())
			continue
		}

		err := MergeTags(desired, additionalTags)
		if err != nil {
			f.log.Debug("error adding tags", string(name), err.Error())
		}

		// Ignore tags only if there is an existing Composed resource with tags in the status
		if observed, ok := observedComposed[name]; ok {
			ignoreTags := f.ResolveIgnoreTags(in.IgnoreTags, oxr, &observed)
			if ignoreTags != nil {
				err := MergeTags(desired, *ignoreTags)
				if err != nil {
					f.log.Debug("error adding tags to ignore", string(name), err.Error())
				}
			}
		}
	}

	if err := response.SetDesiredComposedResources(rsp, desiredComposed); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composed resources in %T", rsp))
		return rsp, nil
	}

	response.Normalf(rsp, "Successfully Processed tags")

	return rsp, nil
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
