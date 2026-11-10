package render

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
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
		observed     string
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

			err := Render(&out, tc.args.field, tc.args.observed, tc.args.resources, tc.args.templateFile)

			if (err != nil) != tc.want.hasErr {
				t.Fatalf("%s\nRender(): unexpected error state: %v", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want.out, out.String()); diff != "" {
				t.Errorf("%s\nRender(): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestRenderProviderTemplates(t *testing.T) {
	resources := []Filter{
		{GroupKind: "storage.gcp.upbound.io/Bucket", Enabled: true},
		{GroupKind: "storage.gcp.upbound.io/BucketACL", Enabled: false},
	}

	cases := map[string]struct {
		reason   string
		field    string
		observed string
		want     []string
		notWant  []string
	}{
		"DefaultObserved": {
			reason:  "Should not override the observed field when none is given",
			field:   "tags",
			want:    []string{`NewProviderFilter("tags", map[string]bool{`, `"storage.gcp.upbound.io/Bucket":`},
			notWant: []string{"WithObservedField"},
		},
		"SameObserved": {
			reason:   "Should not override the observed field when it matches the field",
			field:    "labels",
			observed: "labels",
			want:     []string{`NewProviderFilter("labels", map[string]bool{`},
			notWant:  []string{"WithObservedField"},
		},
		"DistinctObserved": {
			reason:   "Should override the observed field when it differs",
			field:    "labels",
			observed: "effectiveLabels",
			want:     []string{`NewProviderFilter("labels", map[string]bool{`, `}).WithObservedField("effectiveLabels")`},
		},
		"QuotedField": {
			reason: "Should quote the field so it can't break the generated Go",
			field:  `a"b`,
			want:   []string{`NewProviderFilter("a\"b", map[string]bool{`},
		},
	}

	for _, provider := range []string{"aws", "azure"} {
		templateFile := filepath.Join("..", "..", "..", "templates", provider+".tmpl")

		for name, tc := range cases {
			t.Run(provider+"/"+name, func(t *testing.T) {
				var out bytes.Buffer

				err := Render(&out, tc.field, tc.observed, resources, templateFile)
				if err != nil {
					t.Fatalf("%s\nRender(): unexpected error: %v", tc.reason, err)
				}

				// The output must always be valid Go.
				formatted, err := format.Source(out.Bytes())
				if err != nil {
					t.Fatalf("%s\nRender(): output isn't valid Go: %v\n%s", tc.reason, err, out.String())
				}

				for _, w := range tc.want {
					if !strings.Contains(string(formatted), w) {
						t.Errorf("%s\nRender(): output doesn't contain %q:\n%s", tc.reason, w, formatted)
					}
				}

				for _, nw := range tc.notWant {
					if strings.Contains(string(formatted), nw) {
						t.Errorf("%s\nRender(): output contains %q:\n%s", tc.reason, nw, formatted)
					}
				}
			})
		}
	}
}
