package filters

import (
	"strings"
	"testing"
)

func TestGCPLabelRules(t *testing.T) {
	rules := GCPLabelRules()

	cases := map[string]struct {
		reason  string
		key     string
		value   string
		wantErr bool
	}{
		"Simple":           {reason: "A lowercase key and value are valid", key: "team", value: "platform"},
		"Punctuation":      {reason: "Underscores, dashes and digits are valid", key: "cost-center_1", value: "a-b_2"},
		"EmptyValue":       {reason: "Values can be empty", key: "team", value: ""},
		"International":    {reason: "International characters are allowed", key: "ключ", value: "значение"},
		"KeyMaxLength":     {reason: "Keys can be 63 characters", key: strings.Repeat("a", 63), value: "v"},
		"ValueMaxLength":   {reason: "Values can be 63 characters", key: "k", value: strings.Repeat("a", 63)},
		"InternationalMax": {reason: "Length is counted in characters, not bytes", key: strings.Repeat("ж", 63), value: strings.Repeat("ж", 63)},
		"EmptyKey":         {reason: "Keys cannot be empty", key: "", value: "v", wantErr: true},
		"KeyTooLong":       {reason: "Keys can't be longer than 63 characters", key: strings.Repeat("a", 64), value: "v", wantErr: true},
		"ValueTooLong":     {reason: "Values can't be longer than 63 characters", key: "k", value: strings.Repeat("a", 64), wantErr: true},
		"UppercaseKey":     {reason: "Keys can't contain uppercase letters", key: "Environment", value: "v", wantErr: true},
		"UppercaseValue":   {reason: "Values can't contain uppercase letters", key: "env", value: "Prod", wantErr: true},
		"KeyStartsDigit":   {reason: "Keys must start with a letter", key: "1team", value: "v", wantErr: true},
		"KeyStartsDash":    {reason: "Keys must start with a letter", key: "-team", value: "v", wantErr: true},
		"KeyStartsUnder":   {reason: "Keys must start with a letter", key: "_team", value: "v", wantErr: true},
		"ValueStartsDash":  {reason: "Values don't have to start with a letter", key: "k", value: "-1"},
		"KeyWithSlash":     {reason: "Keys can't contain a slash", key: "example.com/team", value: "v", wantErr: true},
		"KeyWithDot":       {reason: "Keys can't contain a dot", key: "a.b", value: "v", wantErr: true},
		"ValueWithSpace":   {reason: "Values can't contain a space", key: "k", value: "a b", wantErr: true},
		"ValueWithColon":   {reason: "Values can't contain a colon", key: "k", value: "a:b", wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := rules.ValidateTag(tc.key, tc.value)
			if (err != nil) != tc.wantErr {
				t.Errorf("%s\nValidateTag(%q, %q): error = %v, wantErr %v", tc.reason, tc.key, tc.value, err, tc.wantErr)
			}
		})
	}
}

func TestGCPLabelRulesMaxTags(t *testing.T) {
	if got, want := GCPLabelRules().MaxTags, 64; got != want {
		t.Errorf("GCPLabelRules().MaxTags = %d, want %d", got, want)
	}
}

func TestNewResourceRules(t *testing.T) {
	got := NewResourceRules()

	if _, ok := got["storage.gcp.upbound.io/Bucket"]; !ok {
		t.Error("NewResourceRules(): no rules for a GCP resource that supports labels")
	}

	if _, ok := got["storage.gcp.m.upbound.io/Bucket"]; !ok {
		t.Error("NewResourceRules(): no rules for a namespaced GCP resource that supports labels")
	}

	for _, groupKind := range []string{
		"storage.gcp.upbound.io/BucketACL", // GCP, but doesn't support labels.
		"acm.aws.upbound.io/Certificate",
		"analysisservices.azure.upbound.io/Server",
	} {
		if _, ok := got[groupKind]; ok {
			t.Errorf("NewResourceRules(): unexpected rules for %q", groupKind)
		}
	}
}
