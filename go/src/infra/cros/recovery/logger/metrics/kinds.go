// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

const (
	// Cr50FwReflashKind is the name/kind in the karte metrics
	// used for query or update the cr 50 fw reflash information.
	Cr50FwReflashKind = "cr50_flash"
	// PerResourceTaskKindGlob is the template for the name/kind for
	// query the complete record for each task for each resource.
	PerResourceTaskKindGlob = "run_task_%s"
	// RunLibraryKind is the actionKind for query the
	// record for overall PARIS recovery result each run.
	RunLibraryKind = "run_recovery"
)
