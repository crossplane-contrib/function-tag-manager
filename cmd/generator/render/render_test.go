package render

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRender(t *testing.T) {
	const tmpl = `{{.Field}}:
{{- range .Resources }}
{{.GroupKind}}={{.Enabled}}
{{- end }}
`

	templateFile := filepath.Join(t.TempDir(), "test.tmpl")

	err := os.WriteFile(templateFile, []byte(tmpl), 0o600)
	if err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	type args struct {
		field        string
		resources    []Filter
		templateFile string
	}

	type want struct {
		out    string
		hasErr bool
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"FieldAndResources": {
			reason: "Should render the field and each resource",
			args: args{
				field: "labels",
				resources: []Filter{
					{GroupKind: "storage.gcp.upbound.io/Bucket", Enabled: true},
					{GroupKind: "storage.gcp.upbound.io/BucketACL", Enabled: false},
				},
				templateFile: templateFile,
			},
			want: want{
				out: "labels:\nstorage.gcp.upbound.io/Bucket=true\nstorage.gcp.upbound.io/BucketACL=false\n",
			},
		},
		"MissingTemplate": {
			reason: "Should return an error if the template can't be read",
			args: args{
				field:        "tags",
				templateFile: filepath.Join(t.TempDir(), "missing.tmpl"),
			},
			want: want{hasErr: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer

			err := Render(&out, tc.args.field, tc.args.resources, tc.args.templateFile)

			if (err != nil) != tc.want.hasErr {
				t.Fatalf("%s\nRender(): unexpected error state: %v", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want.out, out.String()); diff != "" {
				t.Errorf("%s\nRender(): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
