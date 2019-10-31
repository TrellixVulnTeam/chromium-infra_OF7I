// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	tricium "infra/tricium/api/v1"
)

const (
	emptyPatch = "empty_diff.patch"
	inputDir   = "testdata/src"
	enumsPath  = "testdata/src/enums/enums.xml"
)

func analyzeHistogramTestFile(t *testing.T, filePath, patch, prevDir string) []*tricium.Data_Comment {
	// now mocks the current time for testing.
	now = func() time.Time { return time.Date(2019, time.September, 18, 0, 0, 0, 0, time.UTC) }
	// getMilestoneDate is a function that mocks getting the milestone date from server.
	getMilestoneDate = func(milestone int) (time.Time, error) {
		var date time.Time
		var err error
		switch milestone {
		// Use 50 to simulate if server responds with error.
		case 50:
			err = errors.New("Bad milestone request")
		case 77:
			date, _ = time.Parse(dateMilestoneFormat, "2019-07-25T00:00:00")
		case 79:
			date, _ = time.Parse(dateMilestoneFormat, "2019-10-17T00:00:00")
		case 83:
			date, _ = time.Parse(dateMilestoneFormat, "2020-04-23T00:00:00")
		case 87:
			date, _ = time.Parse(dateMilestoneFormat, "2020-10-22T00:00:00")
		case 101:
			date, _ = time.Parse(dateMilestoneFormat, "2022-08-11T00:00:00")
		default:
			t.Errorf("Invalid milestone date in test. Please add your own case")
		}
		return date, err
	}
	filesChanged, err := getDiffsPerFile([]string{filePath}, patch)
	if err != nil {
		t.Errorf("Failed to get diffs per file for %s: %v", filePath, err)
	}
	if patch == filepath.Join(inputDir, emptyPatch) {
		// Assumes all test files are less than 100 lines in length.
		// This is necessary to ensure all lines in the test file are analyzed.
		filesChanged.addedLines[filePath] = makeRange(1, 100)
		filesChanged.removedLines[filePath] = makeRange(1, 100)
	}
	singletonEnums := getSingleElementEnums(enumsPath)
	inputPath := filepath.Join(inputDir, filePath)
	f := openFileOrDie(inputPath)
	defer closeFileOrDie(f)
	return analyzeHistogramFile(f, filePath, prevDir, filesChanged, singletonEnums)
}

func TestHistogramsCheck(t *testing.T) {
	patchPath := filepath.Join(inputDir, emptyPatch)
	patchFile, err := os.Create(patchPath)
	if err != nil {
		t.Errorf("Failed to create empty patch file %s: %v", patchPath, err)
		return
	}
	patchFile.Close()
	defer os.Remove(patchPath)

	// ENUM tests

	Convey("Analyze XML file with no errors: single element enum with baseline", t, func() {
		results := analyzeHistogramTestFile(t, "enums/enum_tests/single_element_baseline.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("enums/enum_tests/single_element_baseline.xml"),
		})
	})

	Convey("Analyze XML file with no errors: multi element enum no baseline", t, func() {
		results := analyzeHistogramTestFile(t, "enums/enum_tests/multi_element_no_baseline.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("enums/enum_tests/multi_element_no_baseline.xml"),
		})
	})

	Convey("Analyze XML file with error: single element enum with no baseline", t, func() {
		results := analyzeHistogramTestFile(t, "enums/enum_tests/single_element_no_baseline.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Enums",
				Message:   singleElementEnumWarning,
				StartLine: 3,
				Path:      "enums/enum_tests/single_element_no_baseline.xml",
			},
			defaultExpiryInfo("enums/enum_tests/single_element_no_baseline.xml"),
		})
	})

	// EXPIRY tests

	Convey("Analyze XML file with no errors: good expiry date", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/good_date.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("expiry/good_date.xml"),
		})
	})

	Convey("Analyze XML file with no expiry", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/no_expiry.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   noExpiryError,
				StartLine: 3,
				Path:      "expiry/no_expiry.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry of never", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/never_expiry_with_comment.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   neverExpiryInfo,
				StartLine: 3,
				Path:      "expiry/never_expiry_with_comment.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry of never and no comment", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/never_expiry_no_comment.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   neverExpiryError,
				StartLine: 3,
				Path:      "expiry/never_expiry_no_comment.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/over_year_expiry.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   farExpiryWarning,
				StartLine: 3,
				Path:      "expiry/over_year_expiry.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in past", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/past_expiry.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   pastExpiryWarning,
				StartLine: 3,
				Path:      "expiry/past_expiry.xml",
			},
		})
	})

	Convey("Analyze XML file with badly formatted expiry", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/unformatted_expiry.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   badExpiryError,
				StartLine: 3,
				Path:      "expiry/unformatted_expiry.xml",
			},
		})
	})

	// EXPIRY MILESTONE tests

	Convey("Analyze XML file with no errors: good milestone expiry", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/good_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   "[INFO]: Expiry date is in 30 days.",
				StartLine: 3,
				Path:      "expiry/milestone/good_milestone.xml",
			},
		})
	})

	Convey("Analyze XML file with no errors: good milestone expiry, but greater than 6 months out", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/over_6months_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   "[INFO]: Expiry date is in 219 days." + changeMilestoneExpiry,
				StartLine: 3,
				Path:      "expiry/milestone/over_6months_milestone.xml",
			},
		})
	})

	Convey("Simulate failure in fetching milestone data from server", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/milestone_fetch_failed.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   milestoneFailure,
				StartLine: 3,
				Path:      "expiry/milestone/milestone_fetch_failed.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year: milestone", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/over_year_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   farExpiryWarning + changeMilestoneExpiry,
				StartLine: 3,
				Path:      "expiry/milestone/over_year_milestone.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year: 3-number milestone", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/over_year_milestone_3.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   farExpiryWarning + changeMilestoneExpiry,
				StartLine: 3,
				Path:      "expiry/milestone/over_year_milestone_3.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in past: milestone", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/past_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   pastExpiryWarning,
				StartLine: 3,
				Path:      "expiry/milestone/past_milestone.xml",
			},
		})
	})

	Convey("Analyze XML file with badly formatted expiry: similar to milestone", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/unformatted_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   badExpiryError,
				StartLine: 3,
				Path:      "expiry/milestone/unformatted_milestone.xml",
			},
		})
	})

	// OBSOLETE tests

	Convey("Analyze XML file with no obsolete message and no errors", t, func() {
		fullPath := "obsolete/good_obsolete_date.xml"
		results := analyzeHistogramTestFile(t, "obsolete/good_obsolete_date.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfoLine(fullPath, 3),
			defaultExpiryInfoLine(fullPath, 13),
			defaultExpiryInfoLine(fullPath, 23),
			defaultExpiryInfoLine(fullPath, 33),
			defaultExpiryInfoLine(fullPath, 43),
			defaultExpiryInfoLine(fullPath, 53),
		})
	})

	Convey("Analyze XML file with no errors and good obsolete milestone", t, func() {
		results := analyzeHistogramTestFile(t, "obsolete/good_obsolete_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   "[INFO]: Expiry date is in 30 days.",
				StartLine: 3,
				Path:      "obsolete/good_obsolete_milestone.xml",
			},
		})
	})

	Convey("Analyze XML file with no date in obsolete message", t, func() {
		results := analyzeHistogramTestFile(t, "obsolete/obsolete_no_date.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Obsolete",
				Message:   obsoleteDateError,
				StartLine: 4,
				Path:      "obsolete/obsolete_no_date.xml",
			},
			defaultExpiryInfo("obsolete/obsolete_no_date.xml"),
		})
	})

	Convey("Analyze XML file with badly formatted date in obsolete message", t, func() {
		fullPath := "obsolete/obsolete_unformatted_date.xml"
		results := analyzeHistogramTestFile(t, "obsolete/obsolete_unformatted_date.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			makeObsoleteDateError(fullPath, 4),
			defaultExpiryInfoLine(fullPath, 3),
			makeObsoleteDateError(fullPath, 14),
			defaultExpiryInfoLine(fullPath, 13),
			makeObsoleteDateError(fullPath, 24),
			defaultExpiryInfoLine(fullPath, 23),
			makeObsoleteDateError(fullPath, 34),
			defaultExpiryInfoLine(fullPath, 33),
			makeObsoleteDateError(fullPath, 44),
			defaultExpiryInfoLine(fullPath, 43),
		})
	})

	// OWNER tests

	Convey("Analyze XML file with no errors: both owners individuals", t, func() {
		results := analyzeHistogramTestFile(t, "owners/good_individuals.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("owners/good_individuals.xml"),
		})
	})

	Convey("Analyze XML file with error: only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "owners/one_owner.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   oneOwnerError,
				StartLine: 4,
				Path:      "owners/one_owner.xml",
			},
			defaultExpiryInfo("owners/one_owner.xml"),
		})
	})

	Convey("Analyze XML file with error: no owners", t, func() {
		results := analyzeHistogramTestFile(t, "owners/no_owners.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   oneOwnerError,
				StartLine: 3,
				Path:      "owners/no_owners.xml",
			},
			defaultExpiryInfo("owners/no_owners.xml"),
		})
	})

	Convey("Analyze XML file with error: first owner is team", t, func() {
		results := analyzeHistogramTestFile(t, "owners/first_team_owner.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   firstOwnerTeamError,
				StartLine: 4,
				Path:      "owners/first_team_owner.xml",
			},
			defaultExpiryInfo("owners/first_team_owner.xml"),
		})
	})

	Convey("Analyze XML file with error: first owner is OWNERS file", t, func() {
		results := analyzeHistogramTestFile(t, "owners/first_owner_file.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   firstOwnerTeamError,
				StartLine: 4,
				Path:      "owners/first_owner_file.xml",
			},
			defaultExpiryInfo("owners/first_owner_file.xml"),
		})
	})

	Convey("Analyze XML file with error: first owner is team, only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "owners/first_team_one_owner.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   oneOwnerTeamError,
				StartLine: 4,
				Path:      "owners/first_team_one_owner.xml",
			},
			defaultExpiryInfo("owners/first_team_one_owner.xml"),
		})
	})

	Convey("Analyze XML file with error: first owner is OWNERS file, only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "owners/first_file_one_owner.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   oneOwnerTeamError,
				StartLine: 4,
				Path:      "owners/first_file_one_owner.xml",
			},
			defaultExpiryInfo("owners/first_file_one_owner.xml"),
		})
	})

	// UNITS tests

	Convey("Analyze XML file no errors, units of microseconds, all users", t, func() {
		results := analyzeHistogramTestFile(t, "units/microseconds_all_users.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("units/microseconds_all_users.xml"),
		})
	})

	Convey("Analyze XML file no errors, units of microseconds, high-resolution", t, func() {
		results := analyzeHistogramTestFile(t, "units/microseconds_high_res.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("units/microseconds_high_res.xml"),
		})
	})

	Convey("Analyze XML file no errors, units of microseconds, low-resolution", t, func() {
		results := analyzeHistogramTestFile(t, "units/microseconds_low_res.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("units/microseconds_low_res.xml"),
		})
	})

	Convey("Analyze XML file with error: units of microseconds, bad summary", t, func() {
		results := analyzeHistogramTestFile(t, "units/microseconds_bad_summary.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Units",
				Message:   unitsMicrosecondsWarning,
				StartLine: 3,
				Path:      "units/microseconds_bad_summary.xml",
			},
			defaultExpiryInfo("units/microseconds_bad_summary.xml"),
		})
	})

	// REMOVED HISTOGRAM tests

	Convey("Analyze XML file with error: histogram deleted", t, func() {
		results := analyzeHistogramTestFile(t, "rm/remove_histogram.xml", "prevdata/tricium_generated_diff.patch", "prevdata/src")
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category: category + "/Removed",
				Message:  removedHistogramError,
				Path:     "rm/remove_histogram.xml",
			},
		})
	})

	// ADDED NAMESPACE tests

	Convey("Analyze XML file with no error: added histogram with same namespace", t, func() {
		results := analyzeHistogramTestFile(t, "namespace/same_namespace.xml", "prevdata/tricium_same_namespace.patch", "prevdata/src")
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfoLine("namespace/same_namespace.xml", 8),
		})
	})

	Convey("Analyze XML file with warning: added namespace", t, func() {
		results := analyzeHistogramTestFile(t, "namespace/add_namespace.xml", "prevdata/tricium_namespace_diff.patch", "prevdata/src")
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfoLine("namespace/add_namespace.xml", 8),
			{
				Category:  category + "/Namespace",
				Message:   fmt.Sprintf(addedNamespaceWarning, "Test2"),
				Path:      "namespace/add_namespace.xml",
				StartLine: 8,
			},
		})
	})
}

func defaultExpiryInfo(path string) *tricium.Data_Comment {
	return defaultExpiryInfoLine(path, 3)
}

func defaultExpiryInfoLine(path string, startLine int) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  category + "/Expiry",
		Message:   "[INFO]: Expiry date is in 104 days.",
		StartLine: int32(startLine),
		Path:      path,
	}
}

func makeObsoleteDateError(path string, startLine int) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  category + "/Obsolete",
		Message:   obsoleteDateError,
		StartLine: int32(startLine),
		Path:      path,
	}
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}
