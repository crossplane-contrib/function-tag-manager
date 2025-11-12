package main

import (
	"testing"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestCheckFieldPath(t *testing.T) {
	type args struct {
		schema *extv1.JSONSchemaProps
		path   []string
	}

	type want struct {
		result bool
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"NilSchema": {
			reason: "Should return false for nil schema",
			args: args{
				schema: nil,
				path:   []string{"spec", "forProvider", "tags"},
			},
			want: want{
				result: false,
			},
		},
		"EmptyPath": {
			reason: "Should return false for empty path",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: map[string]extv1.JSONSchemaProps{
						"spec": {},
					},
				},
				path: []string{},
			},
			want: want{
				result: false,
			},
		},
		"NilProperties": {
			reason: "Should return false when schema has nil properties",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: nil,
				},
				path: []string{"spec"},
			},
			want: want{
				result: false,
			},
		},
		"SingleLevelFieldExists": {
			reason: "Should return true when single level field exists",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: map[string]extv1.JSONSchemaProps{
						"spec": {},
					},
				},
				path: []string{"spec"},
			},
			want: want{
				result: true,
			},
		},
		"SingleLevelFieldDoesNotExist": {
			reason: "Should return false when single level field does not exist",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: map[string]extv1.JSONSchemaProps{
						"spec": {},
					},
				},
				path: []string{"status"},
			},
			want: want{
				result: false,
			},
		},
		"NestedPathExists": {
			reason: "Should return true when nested path exists",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: map[string]extv1.JSONSchemaProps{
						"spec": {
							Properties: map[string]extv1.JSONSchemaProps{
								"forProvider": {
									Properties: map[string]extv1.JSONSchemaProps{
										"tags": {},
									},
								},
							},
						},
					},
				},
				path: []string{"spec", "forProvider", "tags"},
			},
			want: want{
				result: true,
			},
		},
		"NestedPathPartiallyExists": {
			reason: "Should return false when nested path partially exists",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: map[string]extv1.JSONSchemaProps{
						"spec": {
							Properties: map[string]extv1.JSONSchemaProps{
								"forProvider": {},
							},
						},
					},
				},
				path: []string{"spec", "forProvider", "tags"},
			},
			want: want{
				result: false,
			},
		},
		"NestedPathDoesNotExistAtFirstLevel": {
			reason: "Should return false when first level of nested path does not exist",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: map[string]extv1.JSONSchemaProps{
						"status": {},
					},
				},
				path: []string{"spec", "forProvider", "tags"},
			},
			want: want{
				result: false,
			},
		},
		"NestedPathDoesNotExistAtMiddleLevel": {
			reason: "Should return false when middle level of nested path does not exist",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: map[string]extv1.JSONSchemaProps{
						"spec": {
							Properties: map[string]extv1.JSONSchemaProps{
								"initProvider": {},
							},
						},
					},
				},
				path: []string{"spec", "forProvider", "tags"},
			},
			want: want{
				result: false,
			},
		},
		"DeepNestedPath": {
			reason: "Should handle deep nested paths correctly",
			args: args{
				schema: &extv1.JSONSchemaProps{
					Properties: map[string]extv1.JSONSchemaProps{
						"spec": {
							Properties: map[string]extv1.JSONSchemaProps{
								"forProvider": {
									Properties: map[string]extv1.JSONSchemaProps{
										"config": {
											Properties: map[string]extv1.JSONSchemaProps{
												"settings": {
													Properties: map[string]extv1.JSONSchemaProps{
														"tags": {},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				path: []string{"spec", "forProvider", "config", "settings", "tags"},
			},
			want: want{
				result: true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := checkFieldPath(tc.args.schema, tc.args.path)

			if got != tc.want.result {
				t.Errorf("%s\ncheckFieldPath(): want %v, got %v", tc.reason, tc.want.result, got)
			}
		})
	}
}
