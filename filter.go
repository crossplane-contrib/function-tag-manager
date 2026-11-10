package main

import (
	"github.com/crossplane-contrib/function-tag-manager/filters"
	"github.com/crossplane/function-sdk-go/resource"
)

// ManagedField returns the fields that hold tags for a resource, and true if
// the resource supports tags.
func ManagedField(desired *resource.DesiredComposed, filter filters.ResourceFilter) (filters.TagFields, bool) {
	gvk := desired.Resource.GroupVersionKind()

	resource := gvk.Group + "/" + gvk.Kind
	if fields, ok := filter[resource]; ok {
		return fields, true
	}

	// Filter out any remaining resources
	return filters.TagFields{}, false
}
