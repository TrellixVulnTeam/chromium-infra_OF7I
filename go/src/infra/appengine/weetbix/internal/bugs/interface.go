// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"infra/appengine/weetbix/internal/clustering"
)

type BugToUpdate struct {
	BugName string
	// Current cluster statistics for the given bug.
	Cluster *clustering.Cluster
}
