package main

import (
	"context"
	"maps"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"

	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"

	"github.com/stevendborrelli/function-tag-manager/input/v1beta1"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (*fnv1beta1.RunFunctionResponse, error) {
	f.log.Info("Running Function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	in := &v1beta1.Tags{}
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

	// The composed resources that actually exist.
	observed, err := request.GetObservedComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composed resources from %T", req))
		return rsp, nil
	}

	// The composed resources desired by any previous Functions in the pipeline.
	desired, err := request.GetDesiredComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get desired composed resources from %T", req))
		return rsp, nil
	}

	// for _, res := range observed {
	// 	t := new(map[string]string)
	// 	err := res.Resource.GetValueInto("spec.forProvider.tags", t)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		continue
	// 	}
	// 	for k, v := range t {
	// 		fmt.Println(k, v)
	// 	}
	// }

	for name, des := range desired {
		t := new(map[string]string)
		err := des.Resource.GetValueInto("spec.forProvider.tags", t)
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "unable to get tags of resource %s", name))
			return rsp, nil
		}
		maps.Copy(*t, in.AddTags)
		err = des.Resource.SetValue("spec.forProvider.tags", t)
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "unable to set tags of resource %s", name))
			return rsp, nil
		}
		desired[name] = des
	}

	if err := response.SetDesiredComposedResources(rsp, desired); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composed resources in %T", rsp))
		return rsp, nil
	}

	response.Normalf(rsp, "I was run with input %q", in.AddTags)

	return rsp, nil
}
