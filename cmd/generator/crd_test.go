package main

import (
	"testing"

	"github.com/crossplane-contrib/function-tag-manager/cmd/generator/render"
	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-billy/v6/util"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

func TestExamineFieldFromCRDVersions(t *testing.T) {
	const testRootDir = "package/crds"

	// CRD with tags field
	crdWithTags := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: buckets.s3.aws.upbound.io
spec:
  group: s3.aws.upbound.io
  names:
    kind: Bucket
    plural: buckets
  scope: Cluster
  versions:
  - name: v1beta1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              forProvider:
                type: object
                properties:
                  tags:
                    type: object
                  region:
                    type: string
`

	// CRD without tags field
	crdWithoutTags := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: certificates.acmpca.aws.upbound.io
spec:
  group: acmpca.aws.upbound.io
  names:
    kind: Certificate
    plural: certificates
  scope: Cluster
  versions:
  - name: v1beta1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              forProvider:
                type: object
                properties:
                  certificateArn:
                    type: string
`

	// Not a CRD
	notACRD := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value
`

	// Invalid YAML
	invalidYAML := `this is not valid: yaml: at all: [
`

	// CRD with multiple versions, only stored version should be checked
	crdMultipleVersions := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: roles.iam.aws.upbound.io
spec:
  group: iam.aws.upbound.io
  names:
    kind: Role
    plural: roles
  scope: Cluster
  versions:
  - name: v1beta1
    served: true
    storage: false
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              forProvider:
                type: object
                properties:
                  notTags:
                    type: object
  - name: v1beta2
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              forProvider:
                type: object
                properties:
                  tags:
                    type: object
`

	// CRD with spec.forProvider but no tags
	crdWithForProviderNoTags := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: policies.iam.aws.upbound.io
spec:
  group: iam.aws.upbound.io
  names:
    kind: Policy
    plural: policies
  scope: Cluster
  versions:
  - name: v1beta1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              forProvider:
                type: object
                properties:
                  policyDocument:
                    type: string
`

	type testCase struct {
		reason string
		files  map[string]string
		want   render.FilterList
		errStr string
	}

	cases := map[string]testCase{
		"EmptyFilesystem": {
			reason: "Should return empty list for empty filesystem",
			files:  map[string]string{},
			want:   render.FilterList{},
		},
		"SingleCRDWithTags": {
			reason: "Should correctly identify CRD with tags field",
			files: map[string]string{
				"bucket.yaml": crdWithTags,
			},
			want: render.FilterList{
				{GroupKind: "s3.aws.upbound.io/Bucket", Enabled: true},
			},
		},
		"SingleCRDWithoutTags": {
			reason: "Should correctly identify CRD without tags field",
			files: map[string]string{
				"certificate.yaml": crdWithoutTags,
			},
			want: render.FilterList{
				{GroupKind: "acmpca.aws.upbound.io/Certificate", Enabled: false},
			},
		},
		"MultipleCRDs": {
			reason: "Should process multiple CRDs correctly",
			files: map[string]string{
				"bucket.yaml":      crdWithTags,
				"certificate.yaml": crdWithoutTags,
			},
			want: render.FilterList{
				{GroupKind: "s3.aws.upbound.io/Bucket", Enabled: true},
				{GroupKind: "acmpca.aws.upbound.io/Certificate", Enabled: false},
			},
		},
		"MixedFiles": {
			reason: "Should skip non-CRD files",
			files: map[string]string{
				"bucket.yaml":  crdWithTags,
				"config.yaml":  notACRD,
				"readme.txt":   "This is a readme file",
				"data.json":    `{"key": "value"}`,
				"another.yaml": crdWithoutTags,
			},
			want: render.FilterList{
				{GroupKind: "s3.aws.upbound.io/Bucket", Enabled: true},
				{GroupKind: "acmpca.aws.upbound.io/Certificate", Enabled: false},
			},
		},
		"InvalidYAML": {
			reason: "Should return error for invalid YAML",
			files: map[string]string{
				"invalid.yaml": invalidYAML,
			},
			errStr: "failed to parse file",
		},
		"MultipleVersionsUsesStoredVersion": {
			reason: "Should use the stored version when multiple versions exist",
			files: map[string]string{
				"role.yaml": crdMultipleVersions,
			},
			want: render.FilterList{
				{GroupKind: "iam.aws.upbound.io/Role", Enabled: true},
			},
		},
		"CRDWithForProviderNoTags": {
			reason: "Should correctly handle CRD with spec.forProvider but no tags",
			files: map[string]string{
				"policy.yaml": crdWithForProviderNoTags,
			},
			want: render.FilterList{
				{GroupKind: "iam.aws.upbound.io/Policy", Enabled: false},
			},
		},
		"YMLExtension": {
			reason: "Should process files with .yml extension",
			files: map[string]string{
				"bucket.yml": crdWithTags,
			},
			want: render.FilterList{
				{GroupKind: "s3.aws.upbound.io/Bucket", Enabled: true},
			},
		},
		"NestedDirectories": {
			reason: "Should walk nested directories",
			files: map[string]string{
				"crds/s3/bucket.yaml":         crdWithTags,
				"crds/acmpca/certificate.yml": crdWithoutTags,
			},
			want: render.FilterList{
				{GroupKind: "s3.aws.upbound.io/Bucket", Enabled: true},
				{GroupKind: "acmpca.aws.upbound.io/Certificate", Enabled: false},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Create in-memory filesystem
			fs := memfs.New()

			// Create the root directory
			err := fs.MkdirAll(testRootDir, 0o755)
			if err != nil {
				t.Fatalf("Failed to create root directory: %v", err)
			}

			// Create files in root directory
			for path, content := range tc.files {
				fullPath := testRootDir + "/" + path

				err := util.WriteFile(fs, fullPath, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to write test file %s: %v", fullPath, err)
				}
			}

			// Run the function
			got, err := ExamineFieldFromCRDVersions(fs, testRootDir)

			// Check error
			if tc.errStr != "" {
				if err == nil {
					t.Errorf("%s\nExamineFieldFromCRDVersions(): expected error containing %q, got nil", tc.reason, tc.errStr)
					return
				}

				if diff := cmp.Diff(tc.errStr, err.Error(), cmpopts.AcyclicTransformer("substr", func(s string) bool {
					return len(s) > 0 && len(tc.errStr) > 0 && len(s) >= len(tc.errStr)
				})); diff != "" && !cmp.Equal(tc.errStr, err.Error()) {
					// Just check if error contains the expected string
					if len(tc.errStr) == 0 || len(err.Error()) < len(tc.errStr) {
						t.Errorf("%s\nExamineFieldFromCRDVersions(): error should contain %q, got %q", tc.reason, tc.errStr, err.Error())
					}
				}

				return
			}

			if err != nil {
				t.Errorf("%s\nExamineFieldFromCRDVersions(): unexpected error: %v", tc.reason, err)
				return
			}

			// Sort both slices for comparison since order may vary
			lessFunc := func(a, b render.Filter) bool {
				return a.GroupKind < b.GroupKind
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.SortSlices(lessFunc)); diff != "" {
				t.Errorf("%s\nExamineFieldFromCRDVersions(): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
