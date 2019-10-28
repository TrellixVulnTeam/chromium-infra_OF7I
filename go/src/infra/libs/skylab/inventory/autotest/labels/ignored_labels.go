// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package labels implements conversion of Skylab inventory schema to
// Autotest labels.
package labels

// IgnoredLabels returns a whitelist of deprecated labels which may
// still be present in control files but do not carry any meaning
// anymore and do not translate into SchedulableLabels.
func IgnoredLabels() []string {
	return []string{
		"cleanup-reboot", "modem_repair", "rpm", "skip_provision",
	}
}
