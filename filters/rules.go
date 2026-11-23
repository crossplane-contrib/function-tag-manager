package filters

import (
	"unicode"
	"unicode/utf8"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
)

const (
	gcpMaxLabels      = 64
	gcpMaxLabelLength = 63
)

// Rules are the limits that a provider places on the tags of a resource.
type Rules struct {
	// MaxTags is the maximum number of tags on a resource. Zero means no limit.
	MaxTags int
	// ValidateTag returns an error if a tag key and value aren't valid.
	ValidateTag func(key, value string) error
}

// ResourceRules maps a resource's GroupKind to the Rules for its tags.
type ResourceRules map[string]Rules

// GCPLabelRules returns the requirements for GCP labels, see
// https://cloud.google.com/resource-manager/docs/labels-overview
func GCPLabelRules() Rules {
	return Rules{
		MaxTags:     gcpMaxLabels,
		ValidateTag: validateGCPLabel,
	}
}

func validateGCPLabel(key, value string) error {
	if key == "" {
		return errors.New("a label key can't be empty")
	}

	if utf8.RuneCountInString(key) > gcpMaxLabelLength {
		return errors.Errorf("a label key can't be longer than %d characters", gcpMaxLabelLength)
	}

	first, _ := utf8.DecodeRuneInString(key)
	if !isGCPLetter(first) {
		return errors.New("a label key must start with a lowercase letter")
	}

	if !onlyGCPLabelCharacters(key) {
		return errors.New("a label key can only contain lowercase letters, numbers, underscores and dashes")
	}

	if utf8.RuneCountInString(value) > gcpMaxLabelLength {
		return errors.Errorf("a label value can't be longer than %d characters", gcpMaxLabelLength)
	}

	if !onlyGCPLabelCharacters(value) {
		return errors.New("a label value can only contain lowercase letters, numbers, underscores and dashes")
	}

	return nil
}

// isGCPLetter returns true for lowercase letters, and letters that have no case.
func isGCPLetter(r rune) bool {
	return unicode.IsLower(r) || unicode.Is(unicode.Lo, r)
}

func onlyGCPLabelCharacters(s string) bool {
	for _, r := range s {
		if !isGCPLetter(r) && !unicode.IsNumber(r) && r != '_' && r != '-' {
			return false
		}
	}

	return true
}
