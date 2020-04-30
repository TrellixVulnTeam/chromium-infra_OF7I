// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build windows

package atutil

// Signaled returns true if autoserv exited from a signal.
func (r *Result) Signaled() bool {
	panic("not implemented on windows")
}
