// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package step

import (
	te "infra/appengine/sheriff-o-matic/som/testexpectations"
	"infra/monitoring/messages"
)

// ArtifactLink is a link to a test artifact left by perf tests.
type ArtifactLink struct {
	// Name is the name of the artifact.
	Name string `json:"name"`
	// Location is the location of the artifact.
	Location string `json:"location"`
}

// TestWithResult stores the information provided by Findit for a specific test,
// for example if the test is flaky or is there a culprit for the test failure.
// Also contains test-specific details like expectations and any artifacts
// produced by the test run.
type TestWithResult struct {
	TestName     string                     `json:"test_name"`
	IsFlaky      bool                       `json:"is_flaky"`
	SuspectedCLs []messages.SuspectCL       `json:"suspected_cls"`
	Expectations []*te.ExpectationStatement `json:"expectations"`
	Artifacts    []ArtifactLink             `json:"artifacts"`
}
