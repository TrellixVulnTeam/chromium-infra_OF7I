// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"google.golang.org/appengine"

	_ "go.chromium.org/luci/gae/service/datastore/crbug1242998safeget"
)

func main() {
	appengine.Main()
}
