// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package api contains the lab_inventory service API.
package api

//go:generate cproto -proto-path ../../../../../../go.chromium.org/chromiumos/infra/proto/src
//go:generate svcdec -type InventoryServer
