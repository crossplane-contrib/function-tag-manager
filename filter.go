package main

import (
	"github.com/crossplane-contrib/function-tag-manager/filters"
	"github.com/crossplane/function-sdk-go/resource"
)

// SupportedManagedResource returns true if a resource supports tags.
func SupportedManagedResource(desired *resource.DesiredComposed, filter filters.ResourceFilter) bool {
	gvk := desired.Resource.GroupVersionKind()

	resource := gvk.Group + "/" + gvk.Kind
	if val, ok := filter[resource]; ok {
		return val
	}

	// Filter out any remaining resources
	return false
}
