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
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with no errors: multi element enum no baseline", t, func() {
		results := analyzeHistogramTestFile(t, "enums/enum_tests/multi_element_no_baseline.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with error: single element enum with no baseline", t, func() {
		results := analyzeHistogramTestFile(t, "enums/enum_tests/single_element_no_baseline.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Enums",
				Message:              singleElementEnumWarning,
				StartLine:            3,
				EndLine:              3,
				StartChar:            42,
				EndChar:              66,
				Path:                 "enums/enum_tests/single_element_no_baseline.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	// EXPIRY tests

	Convey("Analyze XML file with no errors: good expiry date", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/good_date.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with no errors: good expiry date, expiry on new line", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/expiry_new_line.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with no expiry", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/no_expiry.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              noExpiryError,
				StartLine:            3,
				EndLine:              3,
				Path:                 "expiry/no_expiry.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with no expiry, obsolete", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/no_expiry_obsolete.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with expiry of never", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/never_expiry_with_comment.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              neverExpiryInfo,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              77,
				Path:                 "expiry/never_expiry_with_comment.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with expiry of never and no comment", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/never_expiry_no_comment.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              neverExpiryError,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              77,
				Path:                 "expiry/never_expiry_no_comment.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with expiry of never and no comment, expiry on new line", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/never_expiry_new_line.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              neverExpiryError,
				StartLine:            4,
				EndLine:              4,
				StartChar:            16,
				EndChar:              37,
				Path:                 "expiry/never_expiry_new_line.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/over_year_expiry.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              farExpiryWarning,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              82,
				Path:                 "expiry/over_year_expiry.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with expiry in past", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/past_expiry.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              pastExpiryWarning,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              82,
				Path:                 "expiry/past_expiry.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with badly formatted expiry", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/unformatted_expiry.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              badExpiryError,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              82,
				Path:                 "expiry/unformatted_expiry.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	// EXPIRY MILESTONE tests

	Convey("Analyze XML file with no errors: good milestone expiry", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/good_milestone.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with no errors: good milestone expiry, but greater than 6 months out", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/over_6months_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              changeMilestoneExpiry,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              75,
				Path:                 "expiry/milestone/over_6months_milestone.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Simulate failure in fetching milestone data from server", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/milestone_fetch_failed.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              milestoneFailure,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              75,
				Path:                 "expiry/milestone/milestone_fetch_failed.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year: milestone", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/over_year_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              farExpiryWarning + changeMilestoneExpiry,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              75,
				Path:                 "expiry/milestone/over_year_milestone.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year: 3-number milestone", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/over_year_milestone_3.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              farExpiryWarning + changeMilestoneExpiry,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              76,
				Path:                 "expiry/milestone/over_year_milestone_3.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with expiry in past: milestone", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/past_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              pastExpiryWarning,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              75,
				Path:                 "expiry/milestone/past_milestone.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with badly formatted expiry: similar to milestone", t, func() {
		results := analyzeHistogramTestFile(t, "expiry/milestone/unformatted_milestone.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Expiry",
				Message:              badExpiryError,
				StartLine:            3,
				EndLine:              3,
				StartChar:            56,
				EndChar:              76,
				Path:                 "expiry/milestone/unformatted_milestone.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	// OBSOLETE tests

	Convey("Analyze XML file with no obsolete message and no errors", t, func() {
		results := analyzeHistogramTestFile(t, "obsolete/good_obsolete_date.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with no errors and good obsolete milestone", t, func() {
		results := analyzeHistogramTestFile(t, "obsolete/good_obsolete_milestone.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with no date in obsolete message", t, func() {
		results := analyzeHistogramTestFile(t, "obsolete/obsolete_no_date.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Obsolete",
				Message:              obsoleteDateError,
				StartLine:            4,
				EndLine:              6,
				Path:                 "obsolete/obsolete_no_date.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with badly formatted date in obsolete message", t, func() {
		fullPath := "obsolete/obsolete_unformatted_date.xml"
		results := analyzeHistogramTestFile(t, "obsolete/obsolete_unformatted_date.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			makeObsoleteDateError(fullPath, 4, 6),
			makeObsoleteDateError(fullPath, 14, 16),
			makeObsoleteDateError(fullPath, 24, 26),
			makeObsoleteDateError(fullPath, 34, 36),
			makeObsoleteDateError(fullPath, 44, 46),
		})
	})

	// OWNER tests

	Convey("Analyze XML file with no errors: both owners individuals", t, func() {
		results := analyzeHistogramTestFile(t, "owners/good_individuals.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with error: only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "owners/one_owner.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Owners",
				Message:              oneOwnerError,
				StartLine:            4,
				EndLine:              4,
				Path:                 "owners/one_owner.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with error: no owners", t, func() {
		results := analyzeHistogramTestFile(t, "owners/no_owners.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Owners",
				Message:              oneOwnerError,
				StartLine:            3,
				EndLine:              3,
				Path:                 "owners/no_owners.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with error: first owner is team", t, func() {
		results := analyzeHistogramTestFile(t, "owners/first_team_owner.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Owners",
				Message:              firstOwnerTeamError,
				StartLine:            4,
				EndLine:              5,
				Path:                 "owners/first_team_owner.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with error: first owner is OWNERS file", t, func() {
		results := analyzeHistogramTestFile(t, "owners/first_owner_file.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Owners",
				Message:              firstOwnerTeamError,
				StartLine:            4,
				EndLine:              5,
				Path:                 "owners/first_owner_file.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with error: first owner is team, only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "owners/first_team_one_owner.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Owners",
				Message:              oneOwnerTeamError,
				StartLine:            4,
				EndLine:              4,
				Path:                 "owners/first_team_one_owner.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with error: first owner is OWNERS file, only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "owners/first_file_one_owner.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Owners",
				Message:              oneOwnerTeamError,
				StartLine:            4,
				EndLine:              4,
				Path:                 "owners/first_file_one_owner.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	// UNITS tests

	Convey("Analyze XML file no errors, units of microseconds, all users", t, func() {
		results := analyzeHistogramTestFile(t, "units/microseconds_all_users.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file no errors, units of microseconds, high-resolution", t, func() {
		results := analyzeHistogramTestFile(t, "units/microseconds_high_res.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file no errors, units of microseconds, low-resolution", t, func() {
		results := analyzeHistogramTestFile(t, "units/microseconds_low_res.xml", patchPath, inputDir)
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with error: units of microseconds, bad summary", t, func() {
		results := analyzeHistogramTestFile(t, "units/microseconds_bad_summary.xml", patchPath, inputDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Units",
				Message:              unitsMicrosecondsWarning,
				StartLine:            3,
				EndLine:              3,
				StartChar:            42,
				EndChar:              62,
				Path:                 "units/microseconds_bad_summary.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	// REMOVED HISTOGRAM tests

	Convey("Analyze XML file with error: histogram deleted", t, func() {
		results := analyzeHistogramTestFile(t, "rm/remove_histogram.xml", "prevdata/tricium_generated_diff.patch", "prevdata/src")
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Removed",
				Message:              removedHistogramError,
				Path:                 "rm/remove_histogram.xml",
				ShowOnUnchangedLines: true,
			},
		})
	})

	Convey("Analyze XML file with no error: only owner line deleted", t, func() {
		results := analyzeHistogramTestFile(t, "rm/remove_owner_line.xml", "prevdata/tricium_owner_line_diff.patch", "prevdata/src")
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with no error: only attribute changed", t, func() {
		results := analyzeHistogramTestFile(t, "rm/change_attribute.xml", "prevdata/tricium_attribute_diff.patch", "prevdata/src")
		So(results, ShouldBeNil)
	})

	// ADDED NAMESPACE tests

	Convey("Analyze XML file with no error: added histogram with same namespace", t, func() {
		results := analyzeHistogramTestFile(t, "namespace/same_namespace.xml", "prevdata/tricium_same_namespace.patch", "prevdata/src")
		So(results, ShouldBeNil)
	})

	Convey("Analyze XML file with warning: added namespace", t, func() {
		results := analyzeHistogramTestFile(t, "namespace/add_namespace.xml", "prevdata/tricium_namespace_diff.patch", "prevdata/src")
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:             category + "/Namespace",
				Message:              fmt.Sprintf(addedNamespaceWarning, "Test2"),
				Path:                 "namespace/add_namespace.xml",
				StartLine:            8,
				EndLine:              8,
				ShowOnUnchangedLines: true,
			},
		})
	})
}

func makeObsoleteDateError(path string, startLine int, endLine int) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:             category + "/Obsolete",
		Message:              obsoleteDateError,
		StartLine:            int32(startLine),
		EndLine:              int32(endLine),
		Path:                 path,
		ShowOnUnchangedLines: true,
	}
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}
