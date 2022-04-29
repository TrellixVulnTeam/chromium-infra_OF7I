// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package internal

// Status of a Verdict.
// It is determined by all the test results of the verdict, and exonerations are
// ignored(i.e. failure is treated as a failure, even if it is exonerated).
type VerdictStatus int32

const (
	// A verdict must not have this status.
	// This is only used when filtering verdicts.
	VerdictStatus_VERDICT_STATUS_UNSPECIFIED VerdictStatus = 0
	// All results of the verdict are unexpected.
	VerdictStatus_UNEXPECTED VerdictStatus = 10
	// The verdict has both expected and unexpected results.
	// To be differentiated with AnalyzedTestVariantStatus.FLAKY.
	VerdictStatus_VERDICT_FLAKY VerdictStatus = 30
	// All results of the verdict are expected.
	VerdictStatus_EXPECTED VerdictStatus = 50
)

func (x VerdictStatus) String() string {
	switch x {
	case VerdictStatus_VERDICT_STATUS_UNSPECIFIED:
		return "VERDICT_STATUS_UNSPECIFIED"
	case VerdictStatus_UNEXPECTED:
		return "UNEXPECTED"
	case VerdictStatus_VERDICT_FLAKY:
		return "VERDICT_FLAKY"
	case VerdictStatus_EXPECTED:
		return "EXPECTED"
	default:
		return "UNKNOWN"
	}
}
