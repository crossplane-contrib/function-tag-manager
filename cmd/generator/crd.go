// Package main crd utilities for extracting information from Kubernetes CRDs
package main

import (
	"io/fs"
	"path/filepath"

	"github.com/crossplane-contrib/function-tag-manager/cmd/generator/render"
	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/util"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
)

// ExamineFieldFromCRDVersions walks a directory of CRDs and
// determines if a field exists.
func ExamineFieldFromCRDVersions(f billy.Filesystem, root string) (render.FilterList, error) {
	filter := render.FilterList{}
	err := util.Walk(f, root, func(path string, info fs.FileInfo, e error) error {
		if e != nil {
			return e
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		var u metav1.TypeMeta

		bs, err := util.ReadFile(f, path)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %q", path)
		}

		err = yaml.Unmarshal(bs, &u)
		if err != nil {
			return errors.Wrapf(err, "failed to parse file %q", path)
		}

		if u.GroupVersionKind().Kind == "CustomResourceDefinition" {
			var crd extv1.CustomResourceDefinition

			err := yaml.Unmarshal(bs, &crd)
			if err != nil {
				return errors.Wrapf(err, "failed to unmarshal CRD file %q", path)
			}

			storedVersion, err := GetCRDVersion(crd)
			if err != nil {
				return errors.Wrapf(err, "failed to determine CRD version %q", path)
			}

			schema := storedVersion.Schema
			// Look for the field at fieldpath "spec.forProvider.tags"
			if schema != nil && schema.OpenAPIV3Schema != nil {
				key := crd.Spec.Group + "/" + crd.Spec.Names.Kind
				hasField := checkFieldPath(schema.OpenAPIV3Schema, []string{"spec", "forProvider", "tags"})
				filter = append(filter, render.Filter{GroupKind: key, Enabled: hasField})
			}
		}

		return nil
	})

	return filter, err
}

// checkFieldPath traverses the OpenAPI schema to check if a field path exists.
func checkFieldPath(schema *extv1.JSONSchemaProps, path []string) bool {
	if schema == nil || len(path) == 0 {
		return false
	}
	// Get the first element of the path
	field := path[0]

	// Check if the field exists in the schema properties
	if schema.Properties == nil {
		return false
	}

	property, exists := schema.Properties[field]
	if !exists {
		return false
	}

	// If this is the last element in the path, we found it
	if len(path) == 1 {
		return property.Type == "object"
	}

	// Otherwise, recurse into the next level
	return checkFieldPath(&property, path[1:])
}

// GetCRDVersion returns the Stored and Served version of the CRD.
func GetCRDVersion(crd extv1.CustomResourceDefinition) (extv1.CustomResourceDefinitionVersion, error) {
	for _, version := range crd.Spec.Versions {
		if version.Served && version.Storage {
			return version, nil
		}
	}

	return extv1.CustomResourceDefinitionVersion{}, errors.New("no served and storage version found in CustomResourceDefinition")
}
