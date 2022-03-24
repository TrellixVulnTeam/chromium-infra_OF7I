// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package periphearls provides subcommands to manage peripherals on a DUT
// where there can be more than one instance of the same kind of peripheral
// attached to the same DUT. Each invocation updates a single DUT but can
// act on multiple peripherals attached to it.
package peripherals
