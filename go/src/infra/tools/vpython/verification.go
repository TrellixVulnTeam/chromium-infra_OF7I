// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vpython

import (
	"go.chromium.org/luci/vpython/api/vpython"
)

var verificationScenarios = []*vpython.PEP425Tag{
	{Python: "cp27", Abi: "cp27mu", Platform: "linux_x86_64"},
	{Python: "cp27", Abi: "cp27mu", Platform: "linux_arm64"},

	// NOTE: CIPD generalizes "platform" to "armv6l" even on armv7l platforms.
	{Python: "cp27", Abi: "cp27mu", Platform: "linux_armv6l"},
	{Python: "cp27", Abi: "cp27mu", Platform: "linux_armv7l"},

	{Python: "cp27", Abi: "cp27m", Platform: "macosx_10_10_intel"},

	{Python: "cp27", Abi: "cp27m", Platform: "win32"},
	{Python: "cp27", Abi: "cp27m", Platform: "win_amd64"},
}

var verificationScenarios38 = []*vpython.PEP425Tag{
	{Python: "cp38", Abi: "cp38", Platform: "linux_x86_64"},
	{Python: "cp38", Abi: "cp38", Platform: "linux_arm64"},

	// NOTE: CIPD generalizes "platform" to "armv6l" even on armv7l platforms.
	{Python: "cp38", Abi: "cp38", Platform: "linux_armv6l"},
	{Python: "cp38", Abi: "cp38", Platform: "linux_armv7l"},

	{Python: "cp38", Abi: "cp38", Platform: "macosx_10_10_intel"},

	{Python: "cp38", Abi: "cp38", Platform: "win32"},
	{Python: "cp38", Abi: "cp38", Platform: "win_amd64"},
}
