package main

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/resource/composite"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stevendborrelli/function-tag-manager/input/v1beta1"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
		"ResponseIsReturned": {
			reason: "The Function should return a fatal result if no input was specified",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "hello"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "tag-manager.fn.crossplane.io",
						"kind": "Input",
						"example": "Hello, world!"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "hello", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_NORMAL,
							Message:  "I was run with input \"Hello, world!\"",
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

			if diff := cmp.Diff(tc.want.rsp, rsp, protocmp.Transform()); diff != "" {
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
		"ResourceWithoutLabels": {
			reason: "A resource without Labels returns false",
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
		"ResourceWithLabelTrue": {
			reason: "A resource with Label set to true returns true",
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
						}},
					},
				},
			},
			want: true,
		},
		"ResourceWithLabelTrueMixedCase": {
			reason: "Label value should support mixed case",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
						"apiVersion": "example.crossplane.io/v1",
						"kind":       "TagManager",
						"metadata": map[string]any{
							"name": "test-resource",
							"labels": map[string]any{
								IgnoreResourceLabel: "trUe",
							},
						},
					},
					}},
				},
			},
			want: true,
		},
		"ResourceWithLabelFalse": {
			reason: "A resource with label set to not true returns false",
			args: args{
				res: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
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

func TestResolveAddTags(t *testing.T) {
	fieldPath := "spec.additionalTags"
	optionalFieldPath := "spec.optionalTags"
	type args struct {
		in  []v1beta1.AddTag
		oxr *resource.Composite
	}
	type want struct {
		tu TagUpdater
	}
	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"EmptyInput": {
			reason: "With no input should return an empty TagUpdater",
			args: args{
				in: []v1beta1.AddTag{},
			},
			want: want{TagUpdater{}},
		},
		"SimpleFromValue": {
			reason: "TagUpdater should be populated correctly from simple values",
			args: args{
				in: []v1beta1.AddTag{
					{Type: v1beta1.FromValue, Tags: v1beta1.Tags{"retain": "me", "retain2": "me2"}, Policy: "Retain"},
					{Type: v1beta1.FromValue, Tags: v1beta1.Tags{"replace": "me"}, Policy: "Replace"},
				},
			},
			want: want{TagUpdater{
				Replace: v1beta1.Tags{"replace": "me"},
				Retain:  v1beta1.Tags{"retain": "me", "retain2": "me2"}}},
		},
		"DefaultReplace": {
			reason: "By default tags are replaced",
			args: args{
				in: []v1beta1.AddTag{
					{Type: v1beta1.FromValue, Tags: v1beta1.Tags{"retain": "me", "retain2": "me2"}, Policy: "Retain"},
					{Type: v1beta1.FromValue, Tags: v1beta1.Tags{"replace": "me"}},
				},
			},
			want: want{TagUpdater{
				Replace: v1beta1.Tags{"replace": "me"},
				Retain:  v1beta1.Tags{"retain": "me", "retain2": "me2"}}},
		},
		"MissingValues": {
			reason: "With missing values should populate keys only",
			args: args{
				in: []v1beta1.AddTag{
					{Type: v1beta1.FromValue, Tags: v1beta1.Tags{"retain": "", "retain2": ""}, Policy: "Retain"},
					{Type: v1beta1.FromValue, Tags: v1beta1.Tags{"replace": "me"}},
				},
			},
			want: want{TagUpdater{
				Replace: v1beta1.Tags{"replace": "me"},
				Retain:  v1beta1.Tags{"retain": "", "retain2": ""}}},
		},
		"ValuesFromComposite": {
			reason: "Test getting tags from XR field Path",
			args: args{
				in: []v1beta1.AddTag{
					{
						FromFieldPath: &fieldPath,
						Type:          v1beta1.FromCompositeFieldPath,
					},
					{
						FromFieldPath: &optionalFieldPath,
						Type:          v1beta1.FromCompositeFieldPath,
						Policy:        "Retain",
					},
					{
						Type: v1beta1.FromValue,
						Tags: v1beta1.Tags{"replace": "me"},
					},
				},
				oxr: &resource.Composite{
					Resource: &composite.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
						"apiVersion": "example.crossplane.io/v1",
						"kind":       "XR",
						"metadata": map[string]any{
							"name": "test-resource",
							"labels": map[string]any{
								IgnoreResourceLabel: "False",
							},
						},
						"spec": map[string]any{
							"additionalTags": map[string]any{
								"fromField":  "fromXR",
								"fromField2": "fromXR2",
							},
							"optionalTags": map[string]any{
								"optionalKey":  "fromXR",
								"optionalKey2": "fromXR2",
							},
						},
					}}},
				},
			},
			want: want{
				TagUpdater{
					Replace: v1beta1.Tags{"fromField": "fromXR", "fromField2": "fromXR2", "replace": "me"},
					Retain:  v1beta1.Tags{"optionalKey": "fromXR", "optionalKey2": "fromXR2"},
				}},
		},
	}
	f := &Function{log: logging.NewNopLogger()}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := f.ResolveAddTags(tc.args.in, tc.args.oxr)

			if diff := cmp.Diff(tc.want.tu, got); diff != "" {
				t.Errorf("%s\nfResolveAddTags(): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestAddTags(t *testing.T) {
	type args struct {
		desired *resource.DesiredComposed
		tu      TagUpdater
	}
	type want struct {
		desired *resource.DesiredComposed
		err     error
	}
	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ResourceNoTags": {
			reason: "A basic Test of merging and dealing with a resource with no tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
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
				tu: TagUpdater{
					Replace: v1beta1.Tags{"fromField": "fromXR", "fromField2": "fromXR2", "replace": "me"},
					Retain:  v1beta1.Tags{"optionalKey": "fromXR", "optionalKey2": "fromXR2"},
				},
			},
			want: want{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
						"apiVersion": "example.crossplane.io/v1",
						"kind":       "TagManager",
						"metadata": map[string]any{
							"name": "test-resource",
							"labels": map[string]any{
								IgnoreResourceLabel: "False",
							},
						},
						"spec": map[string]any{
							"forProvider": map[string]any{
								"tags": map[string]any{
									"fromField":    string("fromXR"),
									"fromField2":   string("fromXR2"),
									"optionalKey":  string("fromXR"),
									"optionalKey2": string("fromXR2"),
									"replace":      string("me"),
								},
							},
						},
					},
					}},
				},
				err: nil,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := AddTags(tc.args.desired, tc.args.tu)

			if diff := cmp.Diff(tc.want.desired, tc.args.desired); diff != "" {
				t.Errorf("%s\nfAddTags(): -want err, +got err:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestFilterResourceByGroupKind(t *testing.T) {
	type args struct {
		desired *resource.DesiredComposed
		filter  ResourceFilter
	}
	cases := map[string]struct {
		reason string
		args   args
		want   bool
	}{
		"APIGroupExclude": {
			reason: "Filter Due to API group",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
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
				filter: resourceFilter,
			},
			want: true,
		},
		"APIGroupInclude": {
			reason: "Include due to API group",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
						"apiVersion": "aws.upbound.io/v1beta1",
						"kind":       "VPC",
						"metadata": map[string]any{
							"name": "test-resource",
						},
					},
					}},
				},
				filter: resourceFilter,
			},
			want: false,
		},
		"KindExclude": {
			reason: "Filter Kinds that don't support tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
						"apiVersion": "aws.upbound.io/v1beta1",
						"kind":       "ProviderConfig",
						"metadata": map[string]any{
							"name": "test-resource",
						},
					},
					}},
				},
				filter: resourceFilter,
			},
			want: true,
		},
	}
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			if got := FilterResourceByGroupKind(tt.args.desired, tt.args.filter); got != tt.want {
				t.Errorf("FilterResourceByGroupKind() = %v, want %v", got, tt.want)
			}
		})
	}
}
