// Package render renders a go file that determines whether a resource supports tags
package render

import (
	"io"
	"path/filepath"
	"text/template"
)

// Filter contains a Kubernetes GroupKind and whether it supports tags.
type Filter struct {
	GroupKind string
	Enabled   bool
}

// FilterList is a list of Filters.
type FilterList []Filter

// Render renders resource.
func Render(writer io.Writer, resources []Filter, templateFile string) error {
	tmpl, err := template.New(filepath.Base(templateFile)).ParseFiles(templateFile)
	if err != nil {
		return err
	}

	err = tmpl.Execute(writer, resources)
	if err != nil {
		return err
	}

	return nil
}
