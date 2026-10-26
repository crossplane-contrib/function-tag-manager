package main

import (
	"github.com/crossplane-contrib/function-tag-manager/filters"
	"github.com/crossplane/function-sdk-go/resource"
)

// ManagedField returns the name of the spec.forProvider field that holds tags
// for a resource, and true if the resource supports tags.
func ManagedField(desired *resource.DesiredComposed, filter filters.ResourceFilter) (string, bool) {
	gvk := desired.Resource.GroupVersionKind()

	resource := gvk.Group + "/" + gvk.Kind
	if field, ok := filter[resource]; ok {
		return field, true
	}

	// Filter out any remaining resources
	return "", false
}
