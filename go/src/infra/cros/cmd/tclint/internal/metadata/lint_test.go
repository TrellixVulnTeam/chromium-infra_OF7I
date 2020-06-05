// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package metadata provides functions to lint Chrome OS integration test
// metadata.
package metadata_test

import (
	"infra/cros/cmd/tclint/internal/metadata"
	"infra/cros/cmd/tclint/internal/util/difftests"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	metadataPB "go.chromium.org/chromiumos/config/go/api/test/metadata/v1"
)

const testDataDir = "testdata"

// Tests returned diagnostic messages by comparing against golden expectation
// files.
//
// Returned diagnostics are the public API for tclint tool. This test prevents
// unintended regressions in the messages. To avoid spurious failures due to
// changes in logic unrelated to the message creation, each test case should
// minimize the number of errors detected.
func TestErrorMessages(t *testing.T) {
	for _, tc := range difftests.DiscoverTestCases(t, testDataDir) {
		t.Run(tc.Name, func(t *testing.T) {
			var spec metadataPB.Specification
			tc.LoadInput(t, &spec)

			r := metadata.Lint(&spec)
			got := r.Display()
			defer tc.UpdateGoldenIfRequested(t, got)

			if want, ok := tc.LoadGolden(t); ok {
				if diff := pretty.Compare(want, got); diff != "" {
					t.Errorf("lint errors expectations mismatch, -want +got: \n%s", diff)
				}
			}
		})
	}
}
