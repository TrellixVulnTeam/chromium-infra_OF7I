// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

// This file contains partial Datastore entities

// Issue contains partial document structure in Datastore for kind Issue
type Issue struct {
	Private bool `datastore:"private"`
}

// Patch contains partial document structure in Datastore for kind Patch
type Patch struct {
	Filename string `datastore:"filename"`
}
