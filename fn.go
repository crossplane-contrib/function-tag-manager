package main

import (
	"context"
	"strings"

	"github.com/crossplane-contrib/function-tag-manager/filters"
	"github.com/crossplane-contrib/function-tag-manager/input/v1beta1"
	fncontext "github.com/crossplane/function-sdk-go/context"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1.FunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running Function", "tag-manager", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	in := &v1beta1.ManagedTags{}

	err := request.GetInput(req, in)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	oxr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composite resource from %T", req))
		return rsp, nil
	}

	env := &unstructured.Unstructured{}
	if v, ok := request.GetContextKey(req, fncontext.KeyEnvironment); ok {
		err := resource.AsObject(v.GetStructValue(), env)
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "cannot get Composition environment from %T context key %q", req, fncontext.KeyEnvironment))
			return rsp, nil
		}

		f.log.Debug("Loaded Composition environment from Function context", "context-key", fncontext.KeyEnvironment)
	}

	f.log.WithValues(
		"xr-d", oxr.Resource.GetAPIVersion(),
		"xr-kind", oxr.Resource.GetKind(),
		"xr-name", oxr.Resource.GetName(),
	)

	// Process all the AddTags into 2 groups based on Policy: Replace or Retain
	// we also need to resolve any tags coming from a Composite fieldpath
	additionalTags := f.ResolveAddTags(in.AddTags, oxr, env)

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

	resourceFilter := filters.NewResourceFilter()

	for name, desired := range desiredComposed {
		desired.Resource.GetObjectKind()

		if IgnoreResource(desired) {
			f.log.Debug("skipping resource due to ignore annotation or label", string(name), desired.Resource.GroupVersionKind().String())
			continue
		}

		if !SupportedManagedResource(desired, resourceFilter) {
			f.log.Debug("skipping resource that doesn't support tags", string(name), desired.Resource.GroupVersionKind().String())
			continue
		}

		err := MergeTags(desired, additionalTags)
		if err != nil {
			f.log.Debug("error adding tags", string(name), err.Error())
		}

		// Ignore tags only if there is an existing Composed resource with tags in the status
		if observed, ok := observedComposed[name]; ok {
			ignoreTags := f.ResolveIgnoreTags(in.IgnoreTags, oxr, &observed, env)
			if ignoreTags != nil {
				err := MergeTags(desired, *ignoreTags)
				if err != nil {
					f.log.Debug("error adding tags to ignore", string(name), err.Error())
				}
			}
		}

		removeTags := f.ResolveRemoveTags(in.RemoveTags, oxr, env)
		// Remove tags
		if len(removeTags) > 0 {
			err := RemoveTags(desired, removeTags)
			if err != nil {
				f.log.Debug("error removing tags", string(name), err.Error())
			}
		}
	}

	err = response.SetDesiredComposedResources(rsp, desiredComposed)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composed resources in %T", rsp))
		return rsp, nil
	}

	response.Normalf(rsp, "Successfully Processed tags")

	return rsp, nil
}

// IgnoreResource whether this resource has a label or annotation set to ignore.
// If the annotation is present, it takes precedence over the label.
func IgnoreResource(dc *resource.DesiredComposed) bool {
	if dc == nil {
		return true
	}

	annotations := dc.Resource.GetAnnotations()

	// Check annotation first - if present, it takes precedence
	aval, ok := annotations[IgnoreResourceAnnotation]
	if ok {
		return strings.EqualFold(aval, "true")
	}

	// Fall back to label for backward compatibility
	var labels map[string]any

	err := fieldpath.Pave(dc.Resource.Object).GetValueInto("metadata.labels", &labels)
	if err != nil {
		return false
	}

	val, ok := labels[IgnoreResourceLabel].(string)
	if ok && strings.EqualFold(val, "true") {
		return true
	}

	return false
}
