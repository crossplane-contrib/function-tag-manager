package filters

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewResourceFilter(t *testing.T) {
	got := NewResourceFilter()

	cases := map[string]struct {
		reason    string
		groupKind string
		want      TagFields
		wantOK    bool
	}{
		"AWSSupported": {
			reason:    "Resources that support tags map to the tags field",
			groupKind: "acm.aws.upbound.io/Certificate",
			want:      TagFields{Spec: "tags", Observed: "tags"},
			wantOK:    true,
		},
		"AWSUnsupported": {
			reason:    "Resources that don't support tags are absent",
			groupKind: "acmpca.aws.upbound.io/Certificate",
		},
		"AzureSupported": {
			reason:    "Azure resources that support tags map to the tags field",
			groupKind: "analysisservices.azure.upbound.io/Server",
			want:      TagFields{Spec: "tags", Observed: "tags"},
			wantOK:    true,
		},
		"Unknown": {
			reason:    "Unknown resources are absent",
			groupKind: "example.crossplane.io/Unknown",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fields, ok := got[tc.groupKind]
			if ok != tc.wantOK {
				t.Errorf("%s\nNewResourceFilter()[%q]: ok = %v, want %v", tc.reason, tc.groupKind, ok, tc.wantOK)
			}

			if diff := cmp.Diff(tc.want, fields); diff != "" {
				t.Errorf("%s\nNewResourceFilter()[%q]: -want, +got:\n%s", tc.reason, tc.groupKind, diff)
			}
		})
	}
}

func TestNewResourceFilterFields(t *testing.T) {
	cases := map[string]struct {
		reason    string
		providers []ProviderFilter
		want      ResourceFilter
	}{
		"PerProviderFields": {
			reason: "Each provider's resources map to that provider's field",
			providers: []ProviderFilter{
				NewProviderFilter("tags", map[string]bool{
					"ec2.aws.upbound.io/VPC":             true,
					"ec2.aws.upbound.io/VPCEndpointRule": false,
				}),
				NewProviderFilter("labels", map[string]bool{
					"storage.gcp.upbound.io/Bucket":    true,
					"storage.gcp.upbound.io/BucketACL": false,
				}),
			},
			want: ResourceFilter{
				"ec2.aws.upbound.io/VPC":        {Spec: "tags", Observed: "tags"},
				"storage.gcp.upbound.io/Bucket": {Spec: "labels", Observed: "labels"},
			},
		},
		"ObservedField": {
			reason: "A provider can read existing tags from a different field",
			providers: []ProviderFilter{
				NewProviderFilter("labels", map[string]bool{
					"storage.gcp.upbound.io/Bucket": true,
				}).WithObservedField("effectiveLabels"),
			},
			want: ResourceFilter{
				"storage.gcp.upbound.io/Bucket": {Spec: "labels", Observed: "effectiveLabels"},
			},
		},
		"ObservedFieldDefaultsToField": {
			reason: "An empty ObservedField is the same as Field",
			providers: []ProviderFilter{
				{Field: "labels", Resources: map[string]bool{"storage.gcp.upbound.io/Bucket": true}},
			},
			want: ResourceFilter{
				"storage.gcp.upbound.io/Bucket": {Spec: "labels", Observed: "labels"},
			},
		},
		"EmptyFieldSkipped": {
			reason: "A provider without a field would write tags to spec.forProvider. so it's skipped",
			providers: []ProviderFilter{
				NewProviderFilter("", map[string]bool{"storage.gcp.upbound.io/Bucket": true}),
				NewProviderFilter("tags", map[string]bool{"ec2.aws.upbound.io/VPC": true}),
			},
			want: ResourceFilter{
				"ec2.aws.upbound.io/VPC": {Spec: "tags", Observed: "tags"},
			},
		},
		"DottedFieldSkipped": {
			reason: "A dotted field is a path, not a field, so it's skipped",
			providers: []ProviderFilter{
				NewProviderFilter("foo.bar", map[string]bool{"storage.gcp.upbound.io/Bucket": true}),
			},
			want: ResourceFilter{},
		},
		"InvalidObservedFieldSkipped": {
			reason: "A provider with an invalid observed field is skipped",
			providers: []ProviderFilter{
				NewProviderFilter("labels", map[string]bool{"storage.gcp.upbound.io/Bucket": true}).WithObservedField("a.b"),
			},
			want: ResourceFilter{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := newResourceFilter(tc.providers...)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s\nnewResourceFilter(): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestValidateField(t *testing.T) {
	cases := map[string]struct {
		field   string
		wantErr bool
	}{
		"Tags":         {field: "tags"},
		"CamelCase":    {field: "effectiveLabels"},
		"Underscore":   {field: "my_tags"},
		"Empty":        {field: "", wantErr: true},
		"Dotted":       {field: "foo.bar", wantErr: true},
		"Bracketed":    {field: "foo[0]", wantErr: true},
		"Quote":        {field: `ta"gs`, wantErr: true},
		"Space":        {field: "my tags", wantErr: true},
		"LeadingDigit": {field: "1tags", wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := ValidateField(tc.field)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateField(%q): err = %v, wantErr %v", tc.field, err, tc.wantErr)
			}
		})
	}
}
