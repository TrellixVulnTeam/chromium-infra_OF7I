// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"testing"
	"time"

	"infra/cros/recovery/logger"
)

// Use cases to test parseServodLogTime function.
var parseServodLogTimeTestCases = []struct {
	testName    string
	got         string
	exp         time.Time
	expectedErr bool
}{
	{
		"good",
		"2022-11-22--20-10-15.000",
		time.Date(2022, time.November, 22, 20, 10, 15, 0, time.UTC),
		false,
	},
	{
		"good2",
		"2020-02-02--01-01-01.000",
		time.Date(2020, time.February, 2, 1, 1, 1, 0, time.UTC),
		false,
	},
	{
		"bad format 1",
		"2020-02-02-01-01-01.000",
		time.Now(),
		true,
	},
	{
		"bad format 2",
		"2020-02-02 15:00:00",
		time.Now(),
		true,
	},
}

// TestParseServodLogTimeTest performs table-testing of parseServodLogTime function.
func TestParseServodLogTimeTest(t *testing.T) {
	t.Parallel()
	log := logger.NewLogger()
	for _, tt := range parseServodLogTimeTestCases {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			got, err := parseServodLogTime(tt.got, log)
			if err != nil && !tt.expectedErr {
				t.Errorf("Not expected error but got %q", err)
			} else if err == nil && tt.expectedErr {
				t.Errorf("Expected to fail but passed ")
			} else if err == nil && got == nil {
				t.Errorf("Did not received parsed value")
			}
			if got != nil {
				if !tt.exp.Equal(*got) {
					t.Errorf("Expected to get %v, but got %v", tt.exp, got)
				}
			}
		})
	}
}

// Use cases to test extractTimeFromServoLog function.
var extractTimeFromServoLogTestCase = []struct {
	testName    string
	got         string
	exp         time.Time
	expectedErr bool
}{
	{
		"good",
		"log.2022-11-22--20-10-15.000.DEBUG",
		time.Date(2022, time.November, 22, 20, 10, 15, 0, time.UTC),
		false,
	},
	{
		"good2",
		"log.2020-02-02--01-01-01.000.INFO",
		time.Date(2020, time.February, 2, 1, 1, 1, 0, time.UTC),
		false,
	},
	{
		"bad tail format",
		"log.2020-02-02--01-01-01.000.1INFO",
		time.Now(),
		true,
	},
	{
		"bad time format",
		"log.2020-02-02-01-01-01.000.DEBUG",
		time.Now(),
		true,
	},
	{
		"bad format 2",
		"latest.DEBUG",
		time.Now(),
		true,
	},
}

// TestExtractTimeFromServoLog performs table-testing of extractTimeFromServoLog function.
func TestExtractTimeFromServoLog(t *testing.T) {
	t.Parallel()
	log := logger.NewLogger()
	for _, tt := range extractTimeFromServoLogTestCase {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			got, err := extractTimeFromServoLog(tt.got, log)
			if err != nil && !tt.expectedErr {
				t.Errorf("Not expected error but got %q", err)
			} else if err == nil && tt.expectedErr {
				t.Errorf("Expected to fail but passed ")
			} else if err == nil && got == nil {
				t.Errorf("Did not received parsed value")
			}
			if got != nil {
				if !tt.exp.Equal(*got) {
					t.Errorf("Expected to get %v, but got %v", tt.exp, got)
				}
			}
		})
	}
}
