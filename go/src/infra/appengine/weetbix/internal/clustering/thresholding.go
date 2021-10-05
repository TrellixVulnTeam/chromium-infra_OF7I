// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import "infra/appengine/weetbix/internal/config"

func (c *Cluster) MeetsThreshold(t *config.ImpactThreshold) bool {
	if t.UnexpectedFailures_1D != nil && int64(c.UnexpectedFailures1d) >= *t.UnexpectedFailures_1D {
		return true
	}
	if t.UnexpectedFailures_3D != nil && int64(c.UnexpectedFailures3d) >= *t.UnexpectedFailures_3D {
		return true
	}
	if t.UnexpectedFailures_7D != nil && int64(c.UnexpectedFailures7d) >= *t.UnexpectedFailures_7D {
		return true
	}
	return false
}
