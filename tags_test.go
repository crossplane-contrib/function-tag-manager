package main

import (
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/resource/composite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stevendborrelli/function-tag-manager/input/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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
			err := MergeTags(tc.args.desired, tc.args.tu)

			if diff := cmp.Diff(tc.want.desired, tc.args.desired); diff != "" {
				t.Errorf("%s\nfAddTags(): -want err, +got err:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestResolveIgnoreTags(t *testing.T) {
	ignoreReplacePath := "spec.ignoreTagsReplace"
	ignoreRetainPath := "spec.ignoreTagsRetain"
	type args struct {
		in       []v1beta1.IgnoreTag
		oxr      *resource.Composite
		observed *resource.ObservedComposed
	}
	type want struct {
		tu *TagUpdater
	}
	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"EmptyInput": {
			reason: "With no input should return an empty TagUpdater",
			args: args{
				in:  []v1beta1.IgnoreTag{},
				oxr: &resource.Composite{},
				observed: &resource.ObservedComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "ManagedResource",
							"metadata": map[string]any{
								"name": "test-resource",
								"labels": map[string]any{
									IgnoreResourceLabel: "False",
								},
							},
							"status": map[string]any{
								"atProvider": map[string]any{
									"tags": map[string]any{
										"replaceField1": "fromObserved",
										"replaceField2": "fromObserved2",
										"retainField1":  "fromObserveRetain",
									},
								},
							},
						}}},
				},
			},
			want: want{&TagUpdater{}},
		},
		"CorrectlyReadInput": {
			reason: "Read IgnoreTag fields and correctly populate TagUpdater",
			args: args{
				in: []v1beta1.IgnoreTag{
					{
						Type: v1beta1.FromValue,
						Keys: []string{"replaceField1", "replaceField2"},
					},
					{
						Type:   v1beta1.FromValue,
						Keys:   []string{"retainField1"},
						Policy: v1beta1.ExistingTagPolicyRetain,
					},
					{
						Type:          v1beta1.FromCompositeFieldPath,
						FromFieldPath: &ignoreReplacePath,
						Policy:        v1beta1.ExistingTagPolicyReplace,
					},
					{
						Type:          v1beta1.FromCompositeFieldPath,
						FromFieldPath: &ignoreRetainPath,
						Policy:        v1beta1.ExistingTagPolicyRetain,
					},
				},
				oxr: &resource.Composite{
					Resource: &composite.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
						"apiVersion": "example.crossplane.io/v1",
						"kind":       "XR",
						"metadata": map[string]any{
							"name": "test-resource",
						},
						"spec": map[string]any{
							"ignoreTagsReplace": []string{
								"XRKey",
							},
							"ignoreTagsRetain": []string{
								"XROptionalKey",
								"XROptionalKey2",
							},
						},
					}}},
				},
				observed: &resource.ObservedComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "ManagedResource",
							"metadata": map[string]any{
								"name": "test-resource",
								"labels": map[string]any{
									IgnoreResourceLabel: "False",
								},
							},
							"status": map[string]any{
								"atProvider": map[string]any{
									"tags": map[string]any{
										"replaceField1":  "fromObserved",
										"replaceField2":  "fromObserved2",
										"retainField1":   "fromObserveRetain",
										"XRKey":          "definedInXR",
										"XROptionalKey":  "definedInXR2",
										"XROptionalKey2": "definedInXROptional2",
										"unusedField":    "unused",
									},
								},
							},
						}}},
				},
			},
			want: want{
				&TagUpdater{
					Replace: v1beta1.Tags{
						"XRKey":         "definedInXR",
						"replaceField1": "fromObserved",
						"replaceField2": "fromObserved2",
					},
					Retain: v1beta1.Tags{
						"XROptionalKey":  "definedInXR2",
						"XROptionalKey2": "definedInXROptional2",
						"retainField1":   "fromObserveRetain",
					},
				},
			},
		},
	}

	f := &Function{log: logging.NewNopLogger()}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tu := f.ResolveIgnoreTags(tc.args.in, tc.args.oxr, tc.args.observed)

			if diff := cmp.Diff(tc.want.tu, tu); diff != "" {
				t.Errorf("%s\nfResolveAddTags(): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}
