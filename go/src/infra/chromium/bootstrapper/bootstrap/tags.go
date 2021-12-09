// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import "go.chromium.org/luci/common/errors"

var (
	// PatchRejected indicates that some portion of a patch was rejected.
	PatchRejected = errors.BoolTag{Key: errors.NewTagKey("the patch could not be applied")}
)
