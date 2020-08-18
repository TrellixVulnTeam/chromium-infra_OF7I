// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
)

var (
	ufsGRPCServerCount = metric.NewCounter(
		"grpc/ufs/server/count",
		"Total number of RPCs.",
		nil,
		field.String("method"), // full name of the grpc method
		field.Int("code"),      // grpc.Code of the result
		field.String("caller")) // Caller of this method
)
