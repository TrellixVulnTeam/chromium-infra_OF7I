// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	tricium "infra/tricium/api/v1"
)

const (
	emptyPatch = "empty_diff.patch"
	inputDir   = "testdata"
	enumsPath  = "testdata/src/enums/enums.xml"
)

func analyzeHistogramTestFile(t *testing.T, filePath string, patch string, tempDir string) []*tricium.Data_Comment {
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
		case 87:
			date, _ = time.Parse(dateMilestoneFormat, "2020-10-22T00:00:00")
		case 101:
			date, _ = time.Parse(dateMilestoneFormat, "2022-08-11T00:00:00")
		default:
			t.Errorf("Invalid milestone date in test. Please add your own case")
		}
		return date, err
	}
	filesChanged, err := getDiffsPerFile([]string{filePath}, filepath.Join(inputDir, patch))
	if err != nil {
		t.Errorf("Failed to get diffs per file for %s: %v", filePath, err)
	}
	// Previous files will be put into tempDir.
	getPreviousFiles([]string{filePath}, inputDir, tempDir, patch)
	if patch == emptyPatch {
		// Assumes all test files are less than 100 lines in length.
		// This is necessary to ensure all lines in the test file are analyzed.
		filesChanged.addedLines[filePath] = makeRange(1, 100)
		filesChanged.removedLines[filePath] = makeRange(1, 100)
	}

	singletonEnums := getSingleElementEnums(enumsPath)
	inputPath := filepath.Join(inputDir, filePath)
	f := openFileOrDie(inputPath)
	defer closeFileOrDie(f)
	return analyzeHistogramFile(f, filePath, inputDir, tempDir, filesChanged, singletonEnums)
}

func TestHistogramsCheck(t *testing.T) {
	// Set up the temporary directory where we'll put previous files.
	// The temporary directory should be cleaned up before exiting.
	tempDir, err := ioutil.TempDir("testdata", "get-previous-file")
	if err != nil {
		t.Fatalf("Failed to setup temporary directory: %v", err)
	}
	defer func() {
		if err = os.RemoveAll(tempDir); err != nil {
			t.Fatalf("Failed to clean up temporary directory %q: %v", tempDir, err)
		}
	}()
	patchPath := filepath.Join(inputDir, emptyPatch)
	patchFile, err := os.Create(patchPath)
	if err != nil {
		t.Errorf("Failed to create empty patch file %s: %v", patchPath, err)
		return
	}
	patchFile.Close()
	defer os.Remove(patchPath)

	// ENUM tests
	enumTestPath := filepath.Join("testdata", "src", "enums", "enum_tests")

	Convey("Analyze XML file with no errors: single element enum with baseline", t, func() {
		results := analyzeHistogramTestFile(t, "src/enums/enum_tests/single_element_baseline.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo(filepath.Join(enumTestPath, "single_element_baseline.xml")),
		})
	})

	Convey("Analyze XML file with no errors: multi element enum no baseline", t, func() {
		results := analyzeHistogramTestFile(t, "src/enums/enum_tests/multi_element_no_baseline.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo(filepath.Join(enumTestPath, "multi_element_no_baseline.xml")),
		})
	})

	Convey("Analyze XML file with error: single element enum with no baseline", t, func() {
		results := analyzeHistogramTestFile(t, "src/enums/enum_tests/single_element_no_baseline.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Enums",
				Message:   singleElementEnumWarning,
				StartLine: 3,
				Path:      filepath.Join(enumTestPath, "single_element_no_baseline.xml"),
			},
			defaultExpiryInfo(filepath.Join(enumTestPath, "single_element_no_baseline.xml")),
		})
	})

	// EXPIRY tests
	expiryTestPath := filepath.Join("testdata", "src", "expiry")

	Convey("Analyze XML file with no errors: good expiry date", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/good_date.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo(filepath.Join(expiryTestPath, "good_date.xml")),
		})
	})

	Convey("Analyze XML file with no expiry", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/no_expiry.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   noExpiryError,
				StartLine: 3,
				Path:      filepath.Join(expiryTestPath, "no_expiry.xml"),
			},
		})
	})

	Convey("Analyze XML file with expiry of never", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/never_expiry_with_comment.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   neverExpiryInfo,
				StartLine: 3,
				Path:      filepath.Join(expiryTestPath, "never_expiry_with_comment.xml"),
			},
		})
	})

	Convey("Analyze XML file with expiry of never and no comment", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/never_expiry_no_comment.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   neverExpiryError,
				StartLine: 3,
				Path:      filepath.Join(expiryTestPath, "never_expiry_no_comment.xml"),
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/over_year_expiry.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   farExpiryWarning,
				StartLine: 3,
				Path:      filepath.Join(expiryTestPath, "over_year_expiry.xml"),
			},
		})
	})

	Convey("Analyze XML file with expiry in past", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/past_expiry.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   pastExpiryWarning,
				StartLine: 3,
				Path:      filepath.Join(expiryTestPath, "past_expiry.xml"),
			},
		})
	})

	Convey("Analyze XML file with badly formatted expiry", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/unformatted_expiry.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   badExpiryError,
				StartLine: 3,
				Path:      filepath.Join(expiryTestPath, "unformatted_expiry.xml"),
			},
		})
	})

	// EXPIRY MILESTONE tests
	milestoneTestPath := filepath.Join(expiryTestPath, "milestone")

	Convey("Analyze XML file with no errors: good milestone expiry", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/milestone/good_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   fmt.Sprintf("[INFO]: Expiry date is in 30 days"),
				StartLine: 3,
				Path:      filepath.Join(milestoneTestPath, "good_milestone.xml"),
			},
		})
	})

	Convey("Simulate failure in fetching milestone data from server", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/milestone/milestone_fetch_failed.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   milestoneFailure,
				StartLine: 3,
				Path:      filepath.Join(milestoneTestPath, "milestone_fetch_failed.xml"),
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year: milestone", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/milestone/over_year_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   farExpiryWarning,
				StartLine: 3,
				Path:      filepath.Join(milestoneTestPath, "over_year_milestone.xml"),
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year: 3-number milestone", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/milestone/over_year_milestone_3.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   farExpiryWarning,
				StartLine: 3,
				Path:      filepath.Join(milestoneTestPath, "over_year_milestone_3.xml"),
			},
		})
	})

	Convey("Analyze XML file with expiry in past: milestone", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/milestone/past_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   pastExpiryWarning,
				StartLine: 3,
				Path:      filepath.Join(milestoneTestPath, "past_milestone.xml"),
			},
		})
	})

	Convey("Analyze XML file with badly formatted expiry: similar to milestone", t, func() {
		results := analyzeHistogramTestFile(t, "src/expiry/milestone/unformatted_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   badExpiryError,
				StartLine: 3,
				Path:      filepath.Join(milestoneTestPath, "unformatted_milestone.xml"),
			},
		})
	})

	// OBSOLETE tests
	obsoleteTestPath := filepath.Join("testdata", "src", "obsolete")

	Convey("Analyze XML file with no obsolete message and no errors", t, func() {
		fullPath := filepath.Join(obsoleteTestPath, "good_obsolete_date.xml")
		results := analyzeHistogramTestFile(t, "src/obsolete/good_obsolete_date.xml", emptyPatch, tempDir)
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
		results := analyzeHistogramTestFile(t, "src/obsolete/good_obsolete_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Expiry",
				Message:   fmt.Sprintf("[INFO]: Expiry date is in 30 days"),
				StartLine: 3,
				Path:      filepath.Join(obsoleteTestPath, "good_obsolete_milestone.xml"),
			},
		})
	})

	Convey("Analyze XML file with no date in obsolete message", t, func() {
		results := analyzeHistogramTestFile(t, "src/obsolete/obsolete_no_date.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Obsolete",
				Message:   obsoleteDateError,
				StartLine: 4,
				Path:      filepath.Join(obsoleteTestPath, "obsolete_no_date.xml"),
			},
			defaultExpiryInfo(filepath.Join(obsoleteTestPath, "obsolete_no_date.xml")),
		})
	})

	Convey("Analyze XML file with badly formatted date in obsolete message", t, func() {
		fullPath := filepath.Join(obsoleteTestPath, "obsolete_unformatted_date.xml")
		results := analyzeHistogramTestFile(t, "src/obsolete/obsolete_unformatted_date.xml", emptyPatch, tempDir)
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
	ownerTestPath := filepath.Join("testdata", "src", "owners")

	Convey("Analyze XML file with no errors: both owners individuals", t, func() {
		results := analyzeHistogramTestFile(t, "src/owners/good_individuals.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo(filepath.Join(ownerTestPath, "good_individuals.xml")),
		})
	})

	Convey("Analyze XML file with error: only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "src/owners/one_owner.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   oneOwnerError,
				StartLine: 4,
				Path:      filepath.Join(ownerTestPath, "one_owner.xml"),
			},
			defaultExpiryInfo(filepath.Join(ownerTestPath, "one_owner.xml")),
		})
	})

	Convey("Analyze XML file with error: no owners", t, func() {
		results := analyzeHistogramTestFile(t, "src/owners/no_owners.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   oneOwnerError,
				StartLine: 3,
				Path:      filepath.Join(ownerTestPath, "no_owners.xml"),
			},
			defaultExpiryInfo(filepath.Join(ownerTestPath, "no_owners.xml")),
		})
	})

	Convey("Analyze XML file with error: first owner is team", t, func() {
		results := analyzeHistogramTestFile(t, "src/owners/first_team_owner.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   firstOwnerTeamError,
				StartLine: 4,
				Path:      filepath.Join(ownerTestPath, "first_team_owner.xml"),
			},
			defaultExpiryInfo(filepath.Join(ownerTestPath, "first_team_owner.xml")),
		})
	})

	Convey("Analyze XML file with error: first owner is OWNERS file", t, func() {
		results := analyzeHistogramTestFile(t, "src/owners/first_owner_file.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   firstOwnerTeamError,
				StartLine: 4,
				Path:      filepath.Join(ownerTestPath, "first_owner_file.xml"),
			},
			defaultExpiryInfo(filepath.Join(ownerTestPath, "first_owner_file.xml")),
		})
	})

	Convey("Analyze XML file with error: first owner is team, only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "src/owners/first_team_one_owner.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   oneOwnerTeamError,
				StartLine: 4,
				Path:      filepath.Join(ownerTestPath, "first_team_one_owner.xml"),
			},
			defaultExpiryInfo(filepath.Join(ownerTestPath, "first_team_one_owner.xml")),
		})
	})

	Convey("Analyze XML file with error: first owner is OWNERS file, only one owner", t, func() {
		results := analyzeHistogramTestFile(t, "src/owners/first_file_one_owner.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Owners",
				Message:   oneOwnerTeamError,
				StartLine: 4,
				Path:      filepath.Join(ownerTestPath, "first_file_one_owner.xml"),
			},
			defaultExpiryInfo(filepath.Join(ownerTestPath, "first_file_one_owner.xml")),
		})
	})

	// UNITS tests
	unitTestPath := filepath.Join("testdata", "src", "units")

	Convey("Analyze XML file no errors, units of microseconds, all users", t, func() {
		results := analyzeHistogramTestFile(t, "src/units/microseconds_all_users.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo(filepath.Join(unitTestPath, "microseconds_all_users.xml")),
		})
	})

	Convey("Analyze XML file no errors, units of microseconds, high-resolution", t, func() {
		results := analyzeHistogramTestFile(t, "src/units/microseconds_high_res.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo(filepath.Join(unitTestPath, "microseconds_high_res.xml")),
		})
	})

	Convey("Analyze XML file no errors, units of microseconds, low-resolution", t, func() {
		results := analyzeHistogramTestFile(t, "src/units/microseconds_low_res.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo(filepath.Join(unitTestPath, "microseconds_low_res.xml")),
		})
	})

	Convey("Analyze XML file with error: units of microseconds, bad summary", t, func() {
		results := analyzeHistogramTestFile(t, "src/units/microseconds_bad_summary.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Units",
				Message:   unitsMicrosecondsWarning,
				StartLine: 3,
				Path:      filepath.Join(unitTestPath, "microseconds_bad_summary.xml"),
			},
			defaultExpiryInfo(filepath.Join(unitTestPath, "microseconds_bad_summary.xml")),
		})
	})

	// REMOVED HISTOGRAM tests
	removeTestPath := filepath.Join("testdata", "src", "rm")

	Convey("Analyze XML file with error: histogram deleted", t, func() {
		results := analyzeHistogramTestFile(t, "src/rm/remove_histogram.xml", "tricium_generated_diff.patch", tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category: category + "/Removed",
				Message:  removedHistogramError,
				Path:     filepath.Join(removeTestPath, "remove_histogram.xml"),
			},
		})
	})

	// ADDED NAMESPACE tests
	namespaceTestPath := filepath.Join("testdata", "src", "namespace")

	Convey("Analyze XML file with no error: added histogram with same namespace", t, func() {
		results := analyzeHistogramTestFile(t, "src/namespace/same_namespace.xml", "tricium_same_namespace.patch", tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfoLine(filepath.Join(namespaceTestPath, "same_namespace.xml"), 8),
		})
	})

	Convey("Analyze XML file with warning: added namespace", t, func() {
		results := analyzeHistogramTestFile(t, "src/namespace/add_namespace.xml", "tricium_namespace_diff.patch", tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfoLine(filepath.Join(namespaceTestPath, "add_namespace.xml"), 8),
			{
				Category:  category + "/Namespace",
				Message:   fmt.Sprintf(addedNamespaceWarning, "Test2"),
				Path:      filepath.Join(namespaceTestPath, "add_namespace.xml"),
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
		Message:   fmt.Sprintf("[INFO]: Expiry date is in 104 days"),
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
