package main

import (
	"context"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
)

func TestRunFunction(t *testing.T) {
	type args struct {
		ctx context.Context
		req *fnv1.RunFunctionRequest
	}

	type want struct {
		rsp *fnv1.RunFunctionResponse
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"BadInput": {
			reason: "The Function should return an error when given incorrect input",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "tag-manger.fn.crossplane.io/v1beta1",
						"kind": "ManagedTags",
						"addTagsX": [
						  {
							"type": "FromValue",
							"policy": "Replace",
							"tags": {
							  "from": "value",
							  "add": "tags"
							}
						  },
						  {
							"type": "FromCompositeFieldPath",
							"fromFieldPath": "spec.parameters.additionalTags",
							"policy": "Replace"
						  },
						  {
							"type": "FromCompositeFieldPath",
							"fromFieldPath": "spec.parameters.optionalTags",
							"policy": "Retain"
						  }
						],
						"ignoreTagsY": [
						  {
							"type": "FromValue",
							"policy": "Replace",
							"keys": [
							  "external-tag-1",
							  "external-tag-2"
							]
						  }
						]
					  }`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "tag-manager", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "cannot get Function input from *v1.RunFunctionRequest: cannot get function input *v1beta1.ManagedTags from *v1.RunFunctionRequest...",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"ResponseIsReturned": {
			reason: "The Function should return an empty result with no desired resources",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "tag-manger.fn.crossplane.io/v1beta1",
						"kind": "ManagedTags",
						"addTags": [
						  {
							"type": "FromValue",
							"policy": "Replace",
							"tags": {
							  "from": "value",
							  "add": "tags"
							}
						  },
						  {
							"type": "FromCompositeFieldPath",
							"fromFieldPath": "spec.parameters.additionalTags",
							"policy": "Replace"
						  },
						  {
							"type": "FromCompositeFieldPath",
							"fromFieldPath": "spec.parameters.optionalTags",
							"policy": "Retain"
						  }
						],
						"ignoreTags": [
						  {
							"type": "FromValue",
							"policy": "Replace",
							"keys": [
							  "external-tag-1",
							  "external-tag-2"
							]
						  }
						]
					  }`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Desired: &fnv1.State{},
					Meta:    &fnv1.ResponseMeta{Tag: "tag-manager", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_NORMAL,
							Message:  "Successfully Processed tags",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &Function{log: logging.NewNopLogger()}
			rsp, err := f.RunFunction(tc.args.ctx, tc.args.req)

			if diff := cmp.Diff(tc.want.rsp, rsp,
				protocmp.IgnoreFields(&fnv1.Result{}, "message"), // ignore error messages on parsing input
				protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestIgnoreResource(t *testing.T) {
	type args struct {
		res *resource.DesiredComposed
	}

	cases := map[string]struct {
		reason string
		args   args
		want   bool
	}{
		"NilResource": {
			reason: "Nil Resource returns true",
			args:   args{},
			want:   true,
		},
		"ResourceWithoutLabelOrAnnotation": {
			reason: "A resource without Annotation or Label set to true returns false",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "TagManager",
							"metadata": map[string]any{
								"name": "test-resource",
							},
						},
					}},
				},
			},
			want: false,
		},
		"ResourceWithAnnotationTrue": {
			reason: "A resource with Annotation set to true returns true",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{
						Unstructured: unstructured.Unstructured{
							Object: map[string]any{
								"apiVersion": "example.crossplane.io/v1",
								"kind":       "TagManager",
								"metadata": map[string]any{
									"name": "test-resource",
									"annotations": map[string]any{
										IgnoreResourceAnnotation: "True",
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		"ResourceWithAnnotationTrueMixedCase": {
			reason: "Annotation value should support mixed case",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "TagManager",
							"metadata": map[string]any{
								"name": "test-resource",
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "trUe",
								},
							},
						},
					}},
				},
			},
			want: true,
		},
		"ResourceWithAnnotationFalse": {
			reason: "A resource with the annotation set to not true returns false",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "TagManager",
							"metadata": map[string]any{
								"name": "test-resource",
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "False",
								},
							},
						},
					}},
				},
			},
			want: false,
		},
		"ResourceWithLabelOnlyTrue": {
			reason: "A resource with only the label (no annotation) set to true should still be ignored for backward compatibility",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "TagManager",
							"metadata": map[string]any{
								"name": "test-resource",
								"labels": map[string]any{
									IgnoreResourceLabel: "True",
								},
							},
						},
					}},
				},
			},
			want: true,
		},
		"ResourceWithLabelOnlyFalse": {
			reason: "A resource with only the label set to false returns false",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "TagManager",
							"metadata": map[string]any{
								"name": "test-resource",
								"labels": map[string]any{
									IgnoreResourceLabel: "False",
								},
							},
						},
					}},
				},
			},
			want: false,
		},
		"AnnotationTakesPrecedenceOverLabel_BothTrue": {
			reason: "When both annotation and label are present and set to true, annotation takes precedence (should ignore)",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "TagManager",
							"metadata": map[string]any{
								"name": "test-resource",
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "True",
								},
								"labels": map[string]any{
									IgnoreResourceLabel: "True",
								},
							},
						},
					}},
				},
			},
			want: true,
		},
		"AnnotationTakesPrecedenceOverLabel_AnnotationFalseLabelTrue": {
			reason: "When annotation is false but label is true, annotation takes precedence (should NOT ignore)",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "TagManager",
							"metadata": map[string]any{
								"name": "test-resource",
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "False",
								},
								"labels": map[string]any{
									IgnoreResourceLabel: "True",
								},
							},
						},
					}},
				},
			},
			want: false,
		},
		"AnnotationTakesPrecedenceOverLabel_AnnotationTrueLabelFalse": {
			reason: "When annotation is true but label is false, annotation takes precedence (should ignore)",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "TagManager",
							"metadata": map[string]any{
								"name": "test-resource",
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "True",
								},
								"labels": map[string]any{
									IgnoreResourceLabel: "False",
								},
							},
						},
					}},
				},
			},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			res := IgnoreResource(tc.args.res)
			if res != tc.want {
				t.Errorf("%s\nignoreResource(...): -want: %t, +got: %t", tc.reason, tc.want, res)
			}
		})
	}
}
