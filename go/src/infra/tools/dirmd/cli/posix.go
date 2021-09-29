// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package cli

import "path/filepath"

func canonicalFSPath(path string) (string, error) {
	return filepath.Abs(path)
}
