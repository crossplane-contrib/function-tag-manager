package main

import (
	"testing"

	"github.com/crossplane-contrib/function-tag-manager/filters"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSupportedManagedResource(t *testing.T) {
	AWSResourceFilter := filters.NewResourceFilter()

	type args struct {
		desired *resource.DesiredComposed
		filter  filters.ResourceFilter
	}

	cases := map[string]struct {
		reason string
		args   args
		want   bool
	}{
		"MalformedGroupKnd": {
			reason: "Kubernetes GVK is invalid",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"metadata": map[string]any{
								"name": "test-resource",
								"labels": map[string]any{
									IgnoreResourceLabel: "False",
								},
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"region": "us-west-1",
								},
							},
						},
					}},
				},
				filter: AWSResourceFilter,
			},
			want: false,
		},
		"APIGroupExclude": {
			reason: "Filter Due to API group",
			args: args{
				desired: &resource.DesiredComposed{
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
							"spec": map[string]any{
								"forProvider": map[string]any{
									"region": "us-west-1",
								},
							},
						},
					}},
				},
				filter: AWSResourceFilter,
			},
			want: false,
		},
		"APIGroupInclude": {
			reason: "Include due to API group",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "ec2.aws.upbound.io/v1beta1",
							"kind":       "VPC",
							"metadata": map[string]any{
								"name": "test-resource",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"region": "us-west-1",
								},
							},
						},
					}},
				},
				filter: AWSResourceFilter,
			},
			want: true,
		},
		"KindExclude": {
			reason: "Filter Kinds that don't support tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "aws.upbound.io/v1beta1",
							"kind":       "ProviderConfig",
							"metadata": map[string]any{
								"name": "test-resource",
							},
						},
					}},
				},
				filter: AWSResourceFilter,
			},
			want: false,
		},
		"NotManagedResource": {
			reason: "Filter Resources that aren't a Managed Resources",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "aws.upbound.io/v1beta1",
							"kind":       "NotAnMR",
							"metadata": map[string]any{
								"name": "test-resource",
							},
							"spec": map[string]any{
								"parameters": map[string]any{
									"crossplane": "rocks",
								},
							},
						},
					}},
				},
				filter: filters.NewResourceFilter(),
			},
			want: false,
		},
		"NamespacedGroupInclude": {
			reason: "Include resources with .m namespaced groups that support tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "acm.aws.m.upbound.io/v1beta2",
							"kind":       "Certificate",
							"metadata": map[string]any{
								"name": "test-certificate",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"domainName": "example.com",
								},
							},
						},
					}},
				},
				filter: AWSResourceFilter,
			},
			want: true,
		},
		"NamespacedGroupExclude": {
			reason: "Exclude resources with .m namespaced groups that don't support tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "acmpca.aws.m.upbound.io/v1beta1",
							"kind":       "Certificate",
							"metadata": map[string]any{
								"name": "test-pca-certificate",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"certificateAuthorityArn": "arn:aws:acm-pca:us-west-2:123456789012:certificate-authority/12345678-1234-1234-1234-123456789012",
								},
							},
						},
					}},
				},
				filter: AWSResourceFilter,
			},
			want: false,
		},
		"NamespacedGroupAmplifyApp": {
			reason: "Include Amplify App with .m namespaced group that supports tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "amplify.aws.m.upbound.io/v1beta2",
							"kind":       "App",
							"metadata": map[string]any{
								"name": "test-amplify-app",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"name": "my-app",
								},
							},
						},
					}},
				},
				filter: AWSResourceFilter,
			},
			want: true,
		},
	}
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			if got := SupportedManagedResource(tt.args.desired, tt.args.filter); got != tt.want {
				t.Errorf("SupportedManagedResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
