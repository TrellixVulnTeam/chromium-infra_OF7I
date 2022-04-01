// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testverdictingester

import (
	"testing"

	"infra/appengine/weetbix/internal/testutil"
)

func TestMain(m *testing.M) {
	testutil.SpannerTestMain(m)
}
