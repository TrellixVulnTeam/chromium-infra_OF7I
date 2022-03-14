// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package filterexp

import (
	"github.com/google/go-cmp/cmp"
)

// cmpopts allows unexported fields on types that are owned
// by this karte package.
var cmpopts = []cmp.Option{
	cmp.AllowUnexported(Identifier{}),
	cmp.AllowUnexported(Constant{}),
	cmp.AllowUnexported(Application{}),
	cmp.AllowUnexported(comparisonParseResult{}),
}
