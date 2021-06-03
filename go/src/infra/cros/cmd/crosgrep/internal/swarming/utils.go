// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"bytes"
	"text/template"
)

// TmplPreamble contains definitions that will be used in SQL templates such as `
// A literal ` cannot appear in a raw string.
const tmplPreamble = "{{$tick := \"`\"}}"

// TemplateOrPanic is a helper function that creates a template or panics.
func templateOrPanic(name string, body string) *template.Template {

	return template.Must(template.New(name).Parse(body))
}

// TemplateToString converts a template and its arguments to a string or fails
// if it cannot.
func templateToString(tmpl *template.Template, input interface{}) (string, error) {
	out := &bytes.Buffer{}
	if err := tmpl.Execute(out, input); err != nil {
		return "", err
	}
	return out.String(), nil
}
