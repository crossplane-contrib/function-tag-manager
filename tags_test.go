package main

import (
	"testing"

	"github.com/crossplane-contrib/function-tag-manager/input/v1beta1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/resource/composite"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
)

func TestResolveAddTags(t *testing.T) {
	fieldPath := "spec.additionalTags"
	optionalFieldPath := "spec.optionalTags"

	envFieldPathRetain := "tagsRetain"
	envFieldPathReplace := "tagsReplace"

	type args struct {
		in  []v1beta1.AddTag
		oxr *resource.Composite
		env *unstructured.Unstructured
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
				Retain:  v1beta1.Tags{"retain": "me", "retain2": "me2"},
			}},
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
				Retain:  v1beta1.Tags{"retain": "me", "retain2": "me2"},
			}},
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
				Retain:  v1beta1.Tags{"retain": "", "retain2": ""},
			}},
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
							"annotations": map[string]any{
								IgnoreResourceAnnotation: "False",
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
				},
			},
		},
		"ValuesFromEnvironment": {
			reason: "Test getting tags from Environment field Path",
			args: args{
				in: []v1beta1.AddTag{
					{
						FromFieldPath: &envFieldPathReplace,
						Type:          v1beta1.FromEnvironmentFieldPath,
					},
					{
						FromFieldPath: &envFieldPathRetain,
						Type:          v1beta1.FromEnvironmentFieldPath,
						Policy:        v1beta1.ExistingTagPolicyRetain,
					},
				},
				oxr: &resource.Composite{},
				env: &unstructured.Unstructured{Object: map[string]any{
					"tagsRetain": map[string]any{
						"tag1": "retain",
					},
					"tagsReplace": map[string]any{
						"tag2": "replace",
					},
				}},
			},

			want: want{
				TagUpdater{
					Replace: v1beta1.Tags{"tag2": "replace"},
					Retain:  v1beta1.Tags{"tag1": "retain"},
				},
			},
		},
	}
	f := &Function{log: logging.NewNopLogger()}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := f.ResolveAddTags(tc.args.in, tc.args.oxr, tc.args.env)

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
				tu: TagUpdater{
					Replace: v1beta1.Tags{"fromField": "fromXR", "fromField2": "fromXR2", "replace": "me"},
					Retain:  v1beta1.Tags{"optionalKey": "fromXR", "optionalKey2": "fromXR2"},
				},
			},
			want: want{
				desired: &resource.DesiredComposed{
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

	envReplacePath := "ignoreTagsReplace"
	envRetainPath := "ignoreTagsRetain"

	type args struct {
		in       []v1beta1.IgnoreTag
		oxr      *resource.Composite
		observed *resource.ObservedComposed
		env      *unstructured.Unstructured
	}

	type want struct {
		tu *TagUpdater
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ReturnNilOnMissingStatus": {
			reason: "With empty Observed Status return Nil",
			args:   args{},
			want:   want{nil},
		},

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
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "False",
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
						},
					}},
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
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "False",
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
						},
					}},
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
		"ValuesFromEnvironment": {
			reason: "Test ignoring tags from Environment field Path",
			args: args{
				in: []v1beta1.IgnoreTag{
					{
						Type:          v1beta1.FromEnvironmentFieldPath,
						FromFieldPath: &envReplacePath,
						Policy:        v1beta1.ExistingTagPolicyReplace,
					},
					{
						Type:          v1beta1.FromEnvironmentFieldPath,
						FromFieldPath: &envRetainPath,
						Policy:        v1beta1.ExistingTagPolicyRetain,
					},
				},
				oxr: &resource.Composite{},
				observed: &resource.ObservedComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "example.crossplane.io/v1",
							"kind":       "ManagedResource",
							"metadata": map[string]any{
								"name": "test-resource",
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "False",
								},
							},
							"status": map[string]any{
								"atProvider": map[string]any{
									"tags": map[string]any{
										"replaceEnvKey": "fromEnvReplace",
										"retainEnvKey":  "fromEnvRetain",
										"retainEnvKey2": "fromEnvRetain2",
										"unusedField":   "unused",
									},
								},
							},
						},
					}},
				},
				env: &unstructured.Unstructured{Object: map[string]any{
					"ignoreTagsReplace": []string{
						"replaceEnvKey",
					},
					"ignoreTagsRetain": []string{
						"retainEnvKey",
						"retainEnvKey2",
					},
				}},
			},
			want: want{
				&TagUpdater{
					Replace: v1beta1.Tags{
						"replaceEnvKey": "fromEnvReplace",
					},
					Retain: v1beta1.Tags{
						"retainEnvKey":  "fromEnvRetain",
						"retainEnvKey2": "fromEnvRetain2",
					},
				},
			},
		},
	}

	f := &Function{log: logging.NewNopLogger()}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tu := f.ResolveIgnoreTags(tc.args.in, tc.args.oxr, tc.args.observed, tc.args.env)

			if diff := cmp.Diff(tc.want.tu, tu); diff != "" {
				t.Errorf("%s\nfResolveAddTags(): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestResolveRemoveTags(t *testing.T) {
	fieldPath := "spec.removeTags"

	envFieldPath := "removeTags"

	type args struct {
		in  []v1beta1.RemoveTag
		oxr *resource.Composite
		env *unstructured.Unstructured
	}

	type want struct {
		keys []string
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"EmptyInput": {
			reason: "With no input should return an empty TagUpdater",
			args: args{
				in: []v1beta1.RemoveTag{},
			},
			want: want{
				keys: []string{},
			},
		},
		"SimpleFromValue": {
			reason: "Keys should be populated correctly from simple values",
			args: args{
				in: []v1beta1.RemoveTag{
					{
						Type: v1beta1.FromValue,
						Keys: []string{
							"key1",
							"key2",
						},
					},
					{
						Type: v1beta1.FromValue,
						Keys: []string{
							"key3",
							"key4",
						},
					},
				},
			},
			want: want{
				keys: []string{"key1", "key2", "key3", "key4"},
			},
		},
		"ValuesFromComposite": {
			reason: "Test getting keys from XR field Path",
			args: args{
				in: []v1beta1.RemoveTag{
					{
						Type:          v1beta1.FromCompositeFieldPath,
						FromFieldPath: &fieldPath,
					},
					{
						Type: v1beta1.FromValue,
						Keys: []string{
							"key1",
							"key2",
						},
					},
				},
				oxr: &resource.Composite{
					Resource: &composite.Unstructured{Unstructured: unstructured.Unstructured{Object: map[string]any{
						"apiVersion": "example.crossplane.io/v1",
						"kind":       "XR",
						"metadata": map[string]any{
							"name": "test-resource",
							"annotations": map[string]any{
								IgnoreResourceAnnotation: "False",
							},
						},
						"spec": map[string]any{
							"removeTags": []string{
								"fromXR1",
								"fromXR2",
							},
						},
					}}},
				},
			},
			want: want{
				keys: []string{"fromXR1", "fromXR2", "key1", "key2"},
			},
		},
		"ValuesFromEnvironment": {
			reason: "Test getting keys from Environment field Path",
			args: args{
				in: []v1beta1.RemoveTag{
					{
						Type:          v1beta1.FromEnvironmentFieldPath,
						FromFieldPath: &envFieldPath,
					},
				},
				oxr: &resource.Composite{},
				env: &unstructured.Unstructured{Object: map[string]any{
					"removeTags": []string{
						"fromEnv1",
						"fromEnv2",
					},
				}},
			},
			want: want{
				keys: []string{"fromEnv1", "fromEnv2"},
			},
		},
	}
	f := &Function{log: logging.NewNopLogger()}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := f.ResolveRemoveTags(tc.args.in, tc.args.oxr, tc.args.env)

			if diff := cmp.Diff(tc.want.keys, got, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("%s\nfResolveRemoveTags(): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestRemoveTags(t *testing.T) {
	type args struct {
		desired *resource.DesiredComposed
		keys    []string
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
			reason: "A resource with no tags should be a no-op",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{
						Unstructured: unstructured.Unstructured{
							Object: map[string]any{
								"apiVersion": "example.crossplane.io/v1",
								"kind":       "TagManager",
								"metadata": map[string]any{
									"name": "test-resource",
									"annotations": map[string]any{
										IgnoreResourceAnnotation: "False",
									},
								},
								"spec": map[string]any{
									"forProvider": map[string]any{
										"region": "eu-south",
									},
								},
							},
						},
					},
				},
				keys: []string{"key1", "key2"},
			},
			want: want{
				desired: &resource.DesiredComposed{
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
							"spec": map[string]any{
								"forProvider": map[string]any{
									"region": "eu-south",
								},
							},
						},
					}},
				},
				err: nil,
			},
		},
		"RemoveAllTags": {
			reason: "Remove all tags correctly",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{
						Unstructured: unstructured.Unstructured{
							Object: map[string]any{
								"apiVersion": "example.crossplane.io/v1",
								"kind":       "TagManager",
								"metadata": map[string]any{
									"name": "test-resource",
									"annotations": map[string]any{
										IgnoreResourceAnnotation: "False",
									},
								},
								"spec": map[string]any{
									"forProvider": map[string]any{
										"region": "eu-south",
										"tags": map[string]any{
											"key1": "value1",
											"key2": "value2",
										},
									},
								},
							},
						},
					},
				},
				keys: []string{"key1", "key2"},
			},
			want: want{
				desired: &resource.DesiredComposed{
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
							"spec": map[string]any{
								"forProvider": map[string]any{
									"region": "eu-south",
									"tags":   map[string]any{},
								},
							},
						},
					}},
				},
				err: nil,
			},
		},
		"RemoveSomeTags": {
			reason: "Remove all tags correctly",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{
						Unstructured: unstructured.Unstructured{
							Object: map[string]any{
								"apiVersion": "example.crossplane.io/v1",
								"kind":       "TagManager",
								"metadata": map[string]any{
									"name": "test-resource",
									"annotations": map[string]any{
										IgnoreResourceAnnotation: "False",
									},
								},
								"spec": map[string]any{
									"forProvider": map[string]any{
										"region": "eu-south",
										"tags": map[string]any{
											"key1": "value1",
											"key2": "value2",
											"key3": "keep",
										},
									},
								},
							},
						},
					},
				},
				keys: []string{"key1", "key2"},
			},
			want: want{
				desired: &resource.DesiredComposed{
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
							"spec": map[string]any{
								"forProvider": map[string]any{
									"region": "eu-south",
									"tags": map[string]any{
										"key3": "keep",
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
			err := RemoveTags(tc.args.desired, tc.args.keys)

			if diff := cmp.Diff(tc.want.desired, tc.args.desired); diff != "" {
				t.Errorf("%s\nfAddTags(): -want err, +got err:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}
