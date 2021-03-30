// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	New  Template
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
		New:  Template{tmpls, "new.tmpl"},
	}, nil
}
