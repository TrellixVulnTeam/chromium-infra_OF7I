// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"bytes"
	"context"
	"text/template"

	"cloud.google.com/go/bigquery"
)

// MustMakeTemplate takes the name of a template and the body and produces a template.
// In the event of an error it panics. Templates are not intended to be created dynamically.
func MustMakeTemplate(name string, body string) *template.Template {
	return template.Must(template.New(name).Parse("{{$tick := \"`\"}}" + body))
}

// InstantiateSQLQuery takes a template, a normalizer function, and a bundle of parameters and// creates a SQL query as a string.
func InstantiateSQLQuery(ctx context.Context, template *template.Template, params interface{}) (string, error) {
	var out bytes.Buffer
	if err := template.Execute(&out, params); err != nil {
		return "", err
	}
	return out.String(), nil
}

// RunSQL takes a bigquery client and a sql query and returns an iterator over
// the result set.
func RunSQL(ctx context.Context, client *bigquery.Client, sql string) (*bigquery.RowIterator, error) {
	query := client.Query(sql)
	it, err := query.Read(ctx)
	return it, err
}
