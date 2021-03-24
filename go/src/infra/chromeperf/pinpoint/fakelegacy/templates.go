package fakelegacy

import (
	"io"
	"path/filepath"
	"text/template"

	"go.chromium.org/luci/common/errors"
)

// Templates represents the set of templates needed by the fake legacy service.
// Each specific template will have its own field. Use loadTemplates to create
// a new set of Templates.
type Templates struct {
	Job  Template
	List Template
}

// Template represents a single API response template.
type Template struct {
	tmpls *template.Template
	name  string
}

// Execute applies the data to the API response template and writes the result
// to the provided Writer.
func (t *Template) Execute(dst io.Writer, data interface{}) error {
	return t.tmpls.ExecuteTemplate(dst, t.name, data)
}

// loadTemplates parses every file ending in ".tmpl" in the provided directory
// as a Go text template.
func loadTemplates(dir string) (*Templates, error) {
	tmpls := template.New("fakelegacy-templates")
	tmpls, err := tmpls.ParseGlob(filepath.Join(dir, "*.tmpl"))
	if err != nil {
		return nil, errors.Annotate(err, "failed to loadTemplates").Err()
	}
	return &Templates{
		Job:  Template{tmpls, "job.tmpl"},
		List: Template{tmpls, "list.tmpl"},
	}, nil
}
