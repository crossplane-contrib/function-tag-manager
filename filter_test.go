package main

import (
	"testing"

	"github.com/crossplane-contrib/function-tag-manager/filters"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSupportedManagedResource(t *testing.T) {
	ResourceFilter := filters.NewResourceFilter()

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
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "False",
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
				filter: ResourceFilter,
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
								"annotations": map[string]any{
									IgnoreResourceAnnotation: "False",
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
				filter: ResourceFilter,
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
				filter: ResourceFilter,
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
				filter: ResourceFilter,
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
				filter: ResourceFilter,
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
				filter: ResourceFilter,
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
				filter: ResourceFilter,
			},
			want: true,
		},
		"AzureResourceGroupInclude": {
			reason: "Include Azure ResourceGroup that supports tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "azure.upbound.io/v1beta1",
							"kind":       "ResourceGroup",
							"metadata": map[string]any{
								"name": "test-resource-group",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"location": "eastus",
								},
							},
						},
					}},
				},
				filter: ResourceFilter,
			},
			want: true,
		},
		"AzureNamespacedResourceGroupInclude": {
			reason: "Include Azure ResourceGroup with .m namespaced group that supports tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "azure.m.upbound.io/v1beta1",
							"kind":       "ResourceGroup",
							"metadata": map[string]any{
								"name": "test-resource-group",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"location": "eastus",
								},
							},
						},
					}},
				},
				filter: ResourceFilter,
			},
			want: true,
		},
		"AzureAutomationAccountInclude": {
			reason: "Include Azure Automation Account that supports tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "automation.azure.upbound.io/v1beta1",
							"kind":       "Account",
							"metadata": map[string]any{
								"name": "test-automation-account",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"location":          "eastus",
									"resourceGroupName": "test-rg",
								},
							},
						},
					}},
				},
				filter: ResourceFilter,
			},
			want: true,
		},
		"AzureNamespacedAutomationAccountInclude": {
			reason: "Include Azure Automation Account with .m namespaced group that supports tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "automation.azure.m.upbound.io/v1beta2",
							"kind":       "Account",
							"metadata": map[string]any{
								"name": "test-automation-account",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"location":          "eastus",
									"resourceGroupName": "test-rg",
								},
							},
						},
					}},
				},
				filter: ResourceFilter,
			},
			want: true,
		},
		"AzureAPIManagementAPIExclude": {
			reason: "Exclude Azure API Management API that doesn't support tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "apimanagement.azure.upbound.io/v1beta1",
							"kind":       "API",
							"metadata": map[string]any{
								"name": "test-api",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"resourceGroupName": "test-rg",
								},
							},
						},
					}},
				},
				filter: ResourceFilter,
			},
			want: false,
		},
		"AzureNamespacedAPIManagementAPIExclude": {
			reason: "Exclude Azure API Management API with .m namespaced group that doesn't support tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "apimanagement.azure.m.upbound.io/v1beta1",
							"kind":       "API",
							"metadata": map[string]any{
								"name": "test-api",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"resourceGroupName": "test-rg",
								},
							},
						},
					}},
				},
				filter: ResourceFilter,
			},
			want: false,
		},
		"AzureMonitorAlertInclude": {
			reason: "Include Azure Monitor Alert Processing Rule that supports tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "alertsmanagement.azure.m.upbound.io/v1beta1",
							"kind":       "MonitorAlertProcessingRuleActionGroup",
							"metadata": map[string]any{
								"name": "test-alert-rule",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"resourceGroupName": "test-rg",
								},
							},
						},
					}},
				},
				filter: ResourceFilter,
			},
			want: true,
		},
		"AzureAppConfigurationInclude": {
			reason: "Include Azure App Configuration that supports tags",
			args: args{
				desired: &resource.DesiredComposed{
					Resource: &composed.Unstructured{Unstructured: unstructured.Unstructured{
						Object: map[string]any{
							"apiVersion": "appconfiguration.azure.m.upbound.io/v1beta2",
							"kind":       "Configuration",
							"metadata": map[string]any{
								"name": "test-app-config",
							},
							"spec": map[string]any{
								"forProvider": map[string]any{
									"location":          "eastus",
									"resourceGroupName": "test-rg",
								},
							},
						},
					}},
				},
				filter: ResourceFilter,
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
