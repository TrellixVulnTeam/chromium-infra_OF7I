// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestInstantiateSQLQueryWithoutNormalizer tests that instantiating a SQL
// query works for replacing a digit.
func TestInstantiateSQLQueryWithoutNormalizer(t *testing.T) {
	bg := context.Background()
	sql, err := instantiateSQLQuery(
		bg,
		mustMakeTemplate(
			"name",
			`f({{.Var | printf "%d"}})`,
		),
		&fakeTemplateParams{
			Var: 2,
		},
	)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff("f(2)", sql); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// FakeTemplateParams is a placeholder structure with a single field.
type fakeTemplateParams struct {
	Var int
}
