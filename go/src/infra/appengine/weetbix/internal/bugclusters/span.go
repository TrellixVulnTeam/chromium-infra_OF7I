// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugclusters

import (
	"context"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/span"
)

// BugCluster represents a set of failure associated with a bug.
type BugCluster struct {
	// The LUCI Project for which this bug is being tracked.
	Project string
	// Bug is the identifier of the bug. For monorail, the scheme is
	// monorail/{monorail_project}/{numeric_id}.
	Bug string
	// AssociatedClusterID is the identifier of the associated failure cluster,
	// from which this bug cluster was created.
	AssociatedClusterID string
}

// ReadActive reads all active Weetbix bug clusters.
func ReadActive(ctx context.Context) ([]*BugCluster, error) {
	stmt := spanner.NewStatement(`
		SELECT Project, Bug, AssociatedClusterId
		FROM BugClusters
		WHERE IsActive
		ORDER BY Project, Bug
	`)
	it := span.Query(ctx, stmt)
	var bcs []*BugCluster
	err := it.Do(func(r *spanner.Row) error {
		var project, bugName, associatedClusterID string
		if err := r.Columns(&project, &bugName, &associatedClusterID); err != nil {
			return errors.Annotate(err, "read bug cluster row").Err()
		}
		bc := &BugCluster{
			Project:             project,
			Bug:                 bugName,
			AssociatedClusterID: associatedClusterID,
		}
		bcs = append(bcs, bc)
		return nil
	})
	if err != nil {
		return nil, errors.Annotate(err, "query active bug clusters").Err()
	}
	return bcs, nil
}
