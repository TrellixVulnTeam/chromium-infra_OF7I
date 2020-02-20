// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
)

// Client bundles local interfaces to various remote services used by Runner.
type Client struct {
	Swarming      swarming.Client
	IsolateGetter isolate.GetterFactory
}
