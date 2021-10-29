// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bqutil

import (
	"infra/appengine/weetbix/internal/config"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// DatasetForProject returns the name of the BigQuery dataset that contain's
// the given project's data, in the Weetbix GCP project.
func DatasetForProject(luciProject string) (string, error) {
	// The returned dataset may be used in SQL expressions, so we want to
	// be absolutely sure no SQL Injection is possible.
	if !config.ProjectRe.MatchString(luciProject) {
		return "", errors.New("invalid LUCI Project")
	}

	// The valid alphabet of LUCI project names [1] is [a-z0-9-] whereas
	// the valid alphabet of BQ dataset names [2] is [a-zA-Z0-9_].
	// [1]: https://source.chromium.org/chromium/infra/infra/+/main:luci/appengine/components/components/config/common.py?q=PROJECT_ID_PATTERN
	// [2]: https://cloud.google.com/bigquery/docs/datasets#dataset-naming
	return strings.ReplaceAll(luciProject, "-", "_"), nil
}
