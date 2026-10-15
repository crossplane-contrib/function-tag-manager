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
		want      string
		wantOK    bool
	}{
		"AWSSupported": {
			reason:    "Resources that support tags map to the tags field",
			groupKind: "acm.aws.upbound.io/Certificate",
			want:      "tags",
			wantOK:    true,
		},
		"AWSUnsupported": {
			reason:    "Resources that don't support tags are absent",
			groupKind: "acmpca.aws.upbound.io/Certificate",
		},
		"AzureSupported": {
			reason:    "Azure resources that support tags map to the tags field",
			groupKind: "analysisservices.azure.upbound.io/Server",
			want:      "tags",
			wantOK:    true,
		},
		"Unknown": {
			reason:    "Unknown resources are absent",
			groupKind: "example.crossplane.io/Unknown",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			field, ok := got[tc.groupKind]
			if ok != tc.wantOK {
				t.Errorf("%s\nNewResourceFilter()[%q]: ok = %v, want %v", tc.reason, tc.groupKind, ok, tc.wantOK)
			}

			if diff := cmp.Diff(tc.want, field); diff != "" {
				t.Errorf("%s\nNewResourceFilter()[%q]: -want, +got:\n%s", tc.reason, tc.groupKind, diff)
			}
		})
	}
}

func TestNewResourceFilterFields(t *testing.T) {
	got := newResourceFilter(
		NewProviderFilter("tags", map[string]bool{
			"ec2.aws.upbound.io/VPC":             true,
			"ec2.aws.upbound.io/VPCEndpointRule": false,
		}),
		NewProviderFilter("labels", map[string]bool{
			"storage.gcp.upbound.io/Bucket":    true,
			"storage.gcp.upbound.io/BucketACL": false,
		}),
	)

	want := ResourceFilter{
		"ec2.aws.upbound.io/VPC":        "tags",
		"storage.gcp.upbound.io/Bucket": "labels",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("newResourceFilter(): -want, +got:\n%s", diff)
	}
}
