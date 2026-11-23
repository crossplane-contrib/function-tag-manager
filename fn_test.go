package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/crossplane-contrib/function-tag-manager/filters"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
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
			f, err := NewFunction(logging.NewNopLogger(), response.DefaultTTL)
			if err != nil {
				t.Fatalf("NewFunction(...): %v", err)
			}

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

func TestRunFunctionCacheTTL(t *testing.T) {
	type args struct {
		ctx context.Context
		ttl time.Duration
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
		"CustomTTL": {
			reason: "The TTL in the input should override the Function's TTL",
			args: args{
				ttl: 30 * time.Second,
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "tag-manager.fn.crossplane.io/v1beta1",
						"kind": "ManagedTags",
						"ttl": "5m"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Desired: &fnv1.State{},
					Meta:    &fnv1.ResponseMeta{Tag: "tag-manager", Ttl: durationpb.New(5 * time.Minute)},
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
		"ZeroTTL": {
			reason: "The Function should accept a TTL of zero",
			args: args{
				ttl: response.DefaultTTL,
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "tag-manager.fn.crossplane.io/v1beta1",
						"kind": "ManagedTags",
						"ttl": "0s"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Desired: &fnv1.State{},
					Meta:    &fnv1.ResponseMeta{Tag: "tag-manager", Ttl: durationpb.New(0)},
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
		"FunctionTTL": {
			reason: "The Function should use its own TTL when the input doesn't set one",
			args: args{
				ttl: 30 * time.Second,
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "tag-manager.fn.crossplane.io/v1beta1",
						"kind": "ManagedTags"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Desired: &fnv1.State{},
					Meta:    &fnv1.ResponseMeta{Tag: "tag-manager", Ttl: durationpb.New(30 * time.Second)},
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
		"InvalidTTL": {
			reason: "The Function should return a fatal result when the TTL can't be parsed",
			args: args{
				ttl: response.DefaultTTL,
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "tag-manager.fn.crossplane.io/v1beta1",
						"kind": "ManagedTags",
						"ttl": "5"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "tag-manager", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  `cannot parse ttl: time: missing unit in duration "5"`,
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
		"NegativeTTL": {
			reason: "The Function should return a fatal result when the TTL is negative",
			args: args{
				ttl: response.DefaultTTL,
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "tag-manager.fn.crossplane.io/v1beta1",
						"kind": "ManagedTags",
						"ttl": "-1s"
					}`),
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "tag-manager", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_FATAL,
							Message:  "ttl duration cannot be negative",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f, err := NewFunction(logging.NewNopLogger(), tc.args.ttl)
			if err != nil {
				t.Fatalf("NewFunction(...): %v", err)
			}

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

func TestRunFunctionFields(t *testing.T) {
	// A filter for providers that don't call their tags field `tags`. The GCP
	// resource also reports the tags that exist on the external resource,
	// including those set outside Crossplane, in a different field.
	f := &Function{
		log: logging.NewNopLogger(),
		ttl: response.DefaultTTL,
		filter: filters.ResourceFilter{
			"storage.gcp.upbound.io/Bucket": {Spec: "labels", Observed: "effectiveLabels"},
			"ec2.aws.upbound.io/VPC":        {Spec: "tags", Observed: "tags"},
		},
	}

	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "tag-manger.fn.crossplane.io/v1beta1",
			"kind": "ManagedTags",
			"addTags": [{"type": "FromValue", "policy": "Replace", "tags": {"added": "tag"}}],
			"ignoreTags": [{"type": "FromValue", "policy": "Replace", "keys": ["external"]}],
			"removeTags": [{"type": "FromValue", "keys": ["remove"]}]
		}`),
		Observed: &fnv1.State{
			Resources: map[string]*fnv1.Resource{
				"bucket": {Resource: resource.MustStructJSON(`{
					"apiVersion": "storage.gcp.upbound.io/v1beta1",
					"kind": "Bucket",
					"status": {"atProvider": {
						"labels": {"external": "notReported"},
						"effectiveLabels": {"external": "fromObserved"}
					}}
				}`)},
				"vpc": {Resource: resource.MustStructJSON(`{
					"apiVersion": "ec2.aws.upbound.io/v1beta1",
					"kind": "VPC",
					"status": {"atProvider": {"tags": {"external": "fromObserved"}}}
				}`)},
			},
		},
		Desired: &fnv1.State{
			Resources: map[string]*fnv1.Resource{
				"bucket": {Resource: resource.MustStructJSON(`{
					"apiVersion": "storage.gcp.upbound.io/v1beta1",
					"kind": "Bucket",
					"spec": {"forProvider": {"labels": {"existing": "label", "remove": "me"}}}
				}`)},
				"vpc": {Resource: resource.MustStructJSON(`{
					"apiVersion": "ec2.aws.upbound.io/v1beta1",
					"kind": "VPC",
					"spec": {"forProvider": {"tags": {"existing": "tag", "remove": "me"}}}
				}`)},
				"unsupported": {Resource: resource.MustStructJSON(`{
					"apiVersion": "example.crossplane.io/v1",
					"kind": "Other",
					"spec": {"forProvider": {"region": "us-east-1"}}
				}`)},
			},
		},
	}

	rsp, err := f.RunFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("f.RunFunction(...): unexpected error: %v", err)
	}

	want := map[string]map[string]any{
		"bucket": {
			"apiVersion": "storage.gcp.upbound.io/v1beta1",
			"kind":       "Bucket",
			"spec": map[string]any{"forProvider": map[string]any{
				"labels": map[string]any{"existing": "label", "added": "tag", "external": "fromObserved"},
			}},
		},
		"vpc": {
			"apiVersion": "ec2.aws.upbound.io/v1beta1",
			"kind":       "VPC",
			"spec": map[string]any{"forProvider": map[string]any{
				"tags": map[string]any{"existing": "tag", "added": "tag", "external": "fromObserved"},
			}},
		},
		"unsupported": {
			"apiVersion": "example.crossplane.io/v1",
			"kind":       "Other",
			"spec":       map[string]any{"forProvider": map[string]any{"region": "us-east-1"}},
		},
	}

	got := make(map[string]map[string]any)
	for name, r := range rsp.GetDesired().GetResources() {
		got[name] = r.GetResource().AsMap()
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("f.RunFunction(...): -want desired, +got desired:\n%s", diff)
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

func TestRunFunctionGCP(t *testing.T) {
	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "tag-manager.fn.crossplane.io/v1beta1",
			"kind": "ManagedTags",
			"addTags": [{"type": "FromValue", "policy": "Replace", "tags": {"team": "platform"}}],
			"ignoreTags": [{"type": "FromValue", "policy": "Replace", "keys": ["external"]}],
			"removeTags": [{"type": "FromValue", "keys": ["remove"]}]
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{Resource: resource.MustStructJSON(`{
				"apiVersion": "example.crossplane.io/v1",
				"kind": "XR",
				"metadata": {"name": "xr"}
			}`)},
			Resources: map[string]*fnv1.Resource{
				"bucket": {Resource: resource.MustStructJSON(`{
					"apiVersion": "storage.gcp.upbound.io/v1beta1",
					"kind": "Bucket",
					"metadata": {"name": "bucket"},
					"status": {"atProvider": {"labels": {"external": "set-elsewhere"}, "tags": {"external": "not-a-label"}}}
				}`)},
			},
		},
		Desired: &fnv1.State{
			Resources: map[string]*fnv1.Resource{
				"bucket": {Resource: resource.MustStructJSON(`{
					"apiVersion": "storage.gcp.upbound.io/v1beta1",
					"kind": "Bucket",
					"metadata": {"name": "bucket"},
					"spec": {"forProvider": {"labels": {"existing": "label", "remove": "me"}}}
				}`)},
				"namespacedBucket": {Resource: resource.MustStructJSON(`{
					"apiVersion": "storage.gcp.m.upbound.io/v1beta1",
					"kind": "Bucket",
					"metadata": {"name": "namespaced-bucket", "namespace": "default"},
					"spec": {"forProvider": {}}
				}`)},
				"bucketACL": {Resource: resource.MustStructJSON(`{
					"apiVersion": "storage.gcp.upbound.io/v1beta1",
					"kind": "BucketACL",
					"metadata": {"name": "bucket-acl"},
					"spec": {"forProvider": {}}
				}`)},
				"alertPolicy": {Resource: resource.MustStructJSON(`{
					"apiVersion": "monitoring.gcp.upbound.io/v1beta1",
					"kind": "AlertPolicy",
					"metadata": {"name": "alert-policy"},
					"spec": {"forProvider": {"userLabels": {"existing": "label"}}}
				}`)},
			},
		},
	}

	want := map[string]map[string]any{
		"bucket": {
			"labels": map[string]any{"existing": "label", "team": "platform", "external": "set-elsewhere"},
		},
		"namespacedBucket": {
			"labels": map[string]any{"team": "platform"},
		},
		"bucketACL": {},
		"alertPolicy": {
			"userLabels": map[string]any{"existing": "label"},
		},
	}

	f, err := NewFunction(logging.NewNopLogger(), response.DefaultTTL)
	if err != nil {
		t.Fatalf("NewFunction(...): %v", err)
	}

	rsp, err := f.RunFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("f.RunFunction(...): unexpected error: %v", err)
	}

	got := map[string]map[string]any{}

	for name, r := range rsp.GetDesired().GetResources() {
		forProvider, _ := r.GetResource().AsMap()["spec"].(map[string]any)["forProvider"].(map[string]any)
		got[name] = forProvider
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("f.RunFunction(...): spec.forProvider: -want, +got:\n%s", diff)
	}
}

func TestRunFunctionLabelRules(t *testing.T) {
	manyLabels := func(n int) string {
		labels := make([]string, 0, n)
		for i := range n {
			labels = append(labels, fmt.Sprintf("%q: %q", fmt.Sprintf("key%d", i), "v"))
		}

		return strings.Join(labels, ",")
	}

	request := func(addTags, gvk, forProvider string) *fnv1.RunFunctionRequest {
		apiVersion, kind, _ := strings.Cut(gvk, "|")

		return &fnv1.RunFunctionRequest{
			Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
			Input: resource.MustStructJSON(`{
				"apiVersion": "tag-manager.fn.crossplane.io/v1beta1",
				"kind": "ManagedTags",
				"addTags": [{"type": "FromValue", "policy": "Replace", "tags": {` + addTags + `}}]
			}`),
			Observed: &fnv1.State{
				Composite: &fnv1.Resource{Resource: resource.MustStructJSON(`{
					"apiVersion": "example.crossplane.io/v1", "kind": "XR", "metadata": {"name": "xr"}
				}`)},
			},
			Desired: &fnv1.State{
				Resources: map[string]*fnv1.Resource{
					"bucket": {Resource: resource.MustStructJSON(`{
						"apiVersion": "` + apiVersion + `",
						"kind": "` + kind + `",
						"metadata": {"name": "bucket"},
						"spec": {"forProvider": ` + forProvider + `}
					}`)},
				},
			},
		}
	}

	withRemoveTags := func(req *fnv1.RunFunctionRequest, keys string) *fnv1.RunFunctionRequest {
		in := req.GetInput().AsMap()
		in["removeTags"] = []any{map[string]any{"type": "FromValue", "keys": resource.MustStructJSON(`{"k": [` + keys + `]}`).AsMap()["k"]}}

		input, err := structpb.NewStruct(in)
		if err != nil {
			t.Fatal(err)
		}

		req.Input = input

		return req
	}

	const (
		gcpBucket = "storage.gcp.upbound.io/v1beta1|Bucket"
		awsBucket = "s3.aws.upbound.io/v1beta1|Bucket"
	)

	cases := map[string]struct {
		reason    string
		req       *fnv1.RunFunctionRequest
		wantFatal []string
		wantTags  map[string]any
		wantCount int
	}{
		"InvalidGCPKey": {
			reason:    "A GCP label key with uppercase letters is rejected, naming the resource, the key and the rule",
			req:       request(`"Environment": "prod"`, gcpBucket, `{}`),
			wantFatal: []string{`"bucket"`, `"Environment"`, "label key", "lowercase"},
		},
		"InvalidGCPValue": {
			reason:    "A GCP label value with uppercase letters is rejected",
			req:       request(`"env": "Prod"`, gcpBucket, `{}`),
			wantFatal: []string{`"bucket"`, `"env"`, "label value", "lowercase"},
		},
		"TooManyGCPLabels": {
			reason:    "Adding labels past the GCP limit is rejected",
			req:       request(`"a": "1", "b": "2"`, gcpBucket, `{"labels": {`+manyLabels(63)+`}}`),
			wantFatal: []string{`"bucket"`, "limit of 64"},
		},
		"RemovedLabelsAreNotCounted": {
			reason:    "Labels that are removed don't count towards the GCP limit",
			req:       withRemoveTags(request(`"a": "1", "b": "2"`, gcpBucket, `{"labels": {`+manyLabels(63)+`}}`), `"key0", "key1"`),
			wantCount: 63,
		},
		"GCPLabelsAtLimit": {
			reason:    "A resource can have exactly 64 labels",
			req:       request(`"a": "1"`, gcpBucket, `{"labels": {`+manyLabels(63)+`}}`),
			wantCount: 64,
		},
		"ValidGCPLabels": {
			reason:   "Valid GCP labels are added",
			req:      request(`"team": "platform", "cost-center": "1234"`, gcpBucket, `{}`),
			wantTags: map[string]any{"team": "platform", "cost-center": "1234"},
		},
		"ExistingInvalidGCPLabelIsNotChecked": {
			reason:   "Labels that the Function didn't add aren't validated",
			req:      request(`"team": "platform"`, gcpBucket, `{"labels": {"Legacy": "Value"}}`),
			wantTags: map[string]any{"Legacy": "Value", "team": "platform"},
		},
		"AWSTagsAreNotRestricted": {
			reason:   "Uppercase tags are fine for AWS, which has no rules",
			req:      request(`"Environment": "Prod"`, awsBucket, `{}`),
			wantTags: map[string]any{"Environment": "Prod"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f, err := NewFunction(logging.NewNopLogger(), response.DefaultTTL)
			if err != nil {
				t.Fatalf("NewFunction(...): %v", err)
			}

			rsp, err := f.RunFunction(context.Background(), tc.req)
			if err != nil {
				t.Fatalf("%s\nf.RunFunction(...): unexpected error: %v", tc.reason, err)
			}

			var fatal string

			for _, r := range rsp.GetResults() {
				if r.GetSeverity() == fnv1.Severity_SEVERITY_FATAL {
					fatal = r.GetMessage()
				}
			}

			if len(tc.wantFatal) > 0 {
				if fatal == "" {
					t.Fatalf("%s\nf.RunFunction(...): want a fatal result, got none", tc.reason)
				}

				for _, s := range tc.wantFatal {
					if !strings.Contains(fatal, s) {
						t.Errorf("%s\nf.RunFunction(...): fatal message %q doesn't contain %q", tc.reason, fatal, s)
					}
				}

				return
			}

			if fatal != "" {
				t.Fatalf("%s\nf.RunFunction(...): unexpected fatal result: %s", tc.reason, fatal)
			}

			forProvider, _ := rsp.GetDesired().GetResources()["bucket"].GetResource().AsMap()["spec"].(map[string]any)["forProvider"].(map[string]any)
			labelsOrTags, _ := forProvider["labels"].(map[string]any)

			if labelsOrTags == nil {
				labelsOrTags, _ = forProvider["tags"].(map[string]any)
			}

			if tc.wantCount > 0 {
				if got := len(labelsOrTags); got != tc.wantCount {
					t.Errorf("%s\nf.RunFunction(...): got %d tags, want %d", tc.reason, got, tc.wantCount)
				}

				return
			}

			if diff := cmp.Diff(tc.wantTags, labelsOrTags); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): tags: -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestRunFunctionInvalidLabelsChangeNothing(t *testing.T) {
	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Tag: "tag-manager"},
		Input: resource.MustStructJSON(`{
			"apiVersion": "tag-manager.fn.crossplane.io/v1beta1",
			"kind": "ManagedTags",
			"addTags": [{"type": "FromValue", "policy": "Replace", "tags": {"Environment": "Prod"}}]
		}`),
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{Resource: resource.MustStructJSON(`{
				"apiVersion": "example.crossplane.io/v1", "kind": "XR", "metadata": {"name": "xr"}
			}`)},
		},
		Desired: &fnv1.State{
			Resources: map[string]*fnv1.Resource{
				"aws": {Resource: resource.MustStructJSON(`{
					"apiVersion": "s3.aws.upbound.io/v1beta1", "kind": "Bucket",
					"metadata": {"name": "aws"}, "spec": {"forProvider": {}}
				}`)},
				"gcp": {Resource: resource.MustStructJSON(`{
					"apiVersion": "storage.gcp.upbound.io/v1beta1", "kind": "Bucket",
					"metadata": {"name": "gcp"}, "spec": {"forProvider": {}}
				}`)},
			},
		},
	}

	f, err := NewFunction(logging.NewNopLogger(), response.DefaultTTL)
	if err != nil {
		t.Fatalf("NewFunction(...): %v", err)
	}

	rsp, err := f.RunFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("f.RunFunction(...): unexpected error: %v", err)
	}

	var fatal bool

	for _, r := range rsp.GetResults() {
		fatal = fatal || r.GetSeverity() == fnv1.Severity_SEVERITY_FATAL
	}

	if !fatal {
		t.Fatal("f.RunFunction(...): want a fatal result, got none")
	}

	for name, r := range rsp.GetDesired().GetResources() {
		forProvider, _ := r.GetResource().AsMap()["spec"].(map[string]any)["forProvider"].(map[string]any)
		if len(forProvider) != 0 {
			t.Errorf("f.RunFunction(...): resource %q was changed: %v", name, forProvider)
		}
	}
}
