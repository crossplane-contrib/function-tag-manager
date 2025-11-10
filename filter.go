package main

import (
	"github.com/crossplane/function-sdk-go/resource"
)

// ResourceFilter is a map indicating whether a resource supports tags.
type ResourceFilter map[string]bool

// SupportedManagedResource returns true if a resource supports tags.
func SupportedManagedResource(desired *resource.DesiredComposed, filter ResourceFilter) bool {
	gvk := desired.Resource.GroupVersionKind()

	resource := gvk.Group + "/" + gvk.Kind
	if val, ok := filter[resource]; ok {
		return val
	}

	// Filter out any remaining resources
	return false
}
