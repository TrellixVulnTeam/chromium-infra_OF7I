package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	tricium "infra/tricium/api/v1"
)

const (
	emptyPatch = "testdata/empty_diff.patch"
	enumPath   = "testdata/src/enums/enums.xml"
)

func analyzeTestFile(t *testing.T, name string, patch string, tempDir string) []*tricium.Data_Comment {
	filePath := "testdata/src/" + name
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
			t.Fatalf("Invalid milestone date in test. Please add your own case")
		}
		return date, err
	}
	filesChanged, err := getDiffsPerFile(patch)
	if err != nil {
		t.Fatalf("Failed to get diffs per file for %s: %v", name, err)
	}
	// Original files will be put into tempDir.
	getOriginalFiles([]string{filePath}, tempDir, patch)
	if patch == emptyPatch {
		// Assumes all test files are less than 100 lines in length
		// Necessary to ensure all lines in the test file are analyzed
		filesChanged.addedLines[filePath] = makeRange(1, 100)
		filesChanged.removedLines[filePath] = makeRange(1, 100)
	}
	singletonEnums := getSingleElementEnums(enumPath)
	return analyzeFile(filePath, tempDir, filesChanged, singletonEnums)
}

func TestHistogramsCheck(t *testing.T) {
	// Set up the temporary directory where we'll put original files.
	// The temporary directory should be cleaned up before exiting.
	tempDir, err := ioutil.TempDir("testdata", "get-original-file")
	if err != nil {
		t.Fatalf("Failed to setup temporary directory: %v", err)
	}
	defer func() {
		if err = os.RemoveAll(tempDir); err != nil {
			t.Fatalf("Failed to clean up temporary directory %q: %v", tempDir, err)
		}
	}()
	patchFile, err := os.Create(emptyPatch)
	if err != nil {
		t.Fatalf("Failed to create empty patch file %s: %v", emptyPatch, err)
	}
	patchFile.Close()
	defer os.Remove(emptyPatch)

	// ENUM tests
	Convey("Analyze XML file with no errors: single element enum with baseline", t, func() {
		results := analyzeTestFile(t, "enums/enum_tests/single_element_baseline.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("testdata/src/enums/enum_tests/single_element_baseline.xml"),
		})
	})

	Convey("Analyze XML file with no errors: multi element enum no baseline", t, func() {
		results := analyzeTestFile(t, "enums/enum_tests/multi_element_no_baseline.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("testdata/src/enums/enum_tests/multi_element_no_baseline.xml"),
		})
	})

	Convey("Analyze XML file with error: single element enum with no baseline", t, func() {
		results := analyzeTestFile(t, "enums/enum_tests/single_element_no_baseline.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Enums",
				Message:   singleElementEnumWarning,
				StartLine: 3,
				Path:      "testdata/src/enums/enum_tests/single_element_no_baseline.xml",
			},
			defaultExpiryInfo("testdata/src/enums/enum_tests/single_element_no_baseline.xml"),
		})
	})

	// EXPIRY tests
	Convey("Analyze XML file with no errors: good expiry date", t, func() {
		results := analyzeTestFile(t, "expiry/good_date.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("testdata/src/expiry/good_date.xml"),
		})
	})

	Convey("Analyze XML file with no expiry", t, func() {
		results := analyzeTestFile(t, "expiry/no_expiry.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   noExpiryError,
				StartLine: 3,
				Path:      "testdata/src/expiry/no_expiry.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry of never", t, func() {
		results := analyzeTestFile(t, "expiry/never_expiry_with_comment.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   neverExpiryInfo,
				StartLine: 3,
				Path:      "testdata/src/expiry/never_expiry_with_comment.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry of never and no comment", t, func() {
		results := analyzeTestFile(t, "expiry/never_expiry_no_comment.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   neverExpiryInfo,
				StartLine: 3,
				Path:      "testdata/src/expiry/never_expiry_no_comment.xml",
			},
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   neverExpiryError,
				StartLine: 3,
				Path:      "testdata/src/expiry/never_expiry_no_comment.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year", t, func() {
		results := analyzeTestFile(t, "expiry/over_year_expiry.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   farExpiryWarning,
				StartLine: 3,
				Path:      "testdata/src/expiry/over_year_expiry.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in past", t, func() {
		results := analyzeTestFile(t, "expiry/past_expiry.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   pastExpiryWarning,
				StartLine: 3,
				Path:      "testdata/src/expiry/past_expiry.xml",
			},
		})
	})

	Convey("Analyze XML file with badly formatted expiry", t, func() {
		results := analyzeTestFile(t, "expiry/unformatted_expiry.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   badExpiryError,
				StartLine: 3,
				Path:      "testdata/src/expiry/unformatted_expiry.xml",
			},
		})
	})

	// EXPIRY MILESTONE tests
	Convey("Analyze XML file with no errors: good milestone expiry", t, func() {
		results := analyzeTestFile(t, "expiry/milestone/good_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   fmt.Sprintf("[INFO]: Expiry date is in 30 days"),
				StartLine: 3,
				Path:      "testdata/src/expiry/milestone/good_milestone.xml",
			},
		})
	})

	Convey("Simulate failure in fetching milestone data from server", t, func() {
		results := analyzeTestFile(t, "expiry/milestone/milestone_fetch_failed.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   milestoneFailure,
				StartLine: 3,
				Path:      "testdata/src/expiry/milestone/milestone_fetch_failed.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year: milestone", t, func() {
		results := analyzeTestFile(t, "expiry/milestone/over_year_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   farExpiryWarning,
				StartLine: 3,
				Path:      "testdata/src/expiry/milestone/over_year_milestone.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in over one year: 3-number milestone", t, func() {
		results := analyzeTestFile(t, "expiry/milestone/over_year_milestone_3.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   farExpiryWarning,
				StartLine: 3,
				Path:      "testdata/src/expiry/milestone/over_year_milestone_3.xml",
			},
		})
	})

	Convey("Analyze XML file with expiry in past: milestone", t, func() {
		results := analyzeTestFile(t, "expiry/milestone/past_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   pastExpiryWarning,
				StartLine: 3,
				Path:      "testdata/src/expiry/milestone/past_milestone.xml",
			},
		})
	})

	Convey("Analyze XML file with badly formatted expiry: similar to milestone", t, func() {
		results := analyzeTestFile(t, "expiry/milestone/unformatted_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   badExpiryError,
				StartLine: 3,
				Path:      "testdata/src/expiry/milestone/unformatted_milestone.xml",
			},
		})
	})

	// OBSOLETE tests
	Convey("Analyze XML file with no obsolete message and no errors", t, func() {
		currPath := "obsolete/good_obsolete_date.xml"
		fullPath := "testdata/src/" + currPath
		results := analyzeTestFile(t, currPath, emptyPatch, tempDir)
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
		results := analyzeTestFile(t, "obsolete/good_obsolete_milestone.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
				Message:   fmt.Sprintf("[INFO]: Expiry date is in 30 days"),
				StartLine: 3,
				Path:      "testdata/src/obsolete/good_obsolete_milestone.xml",
			},
		})
	})

	Convey("Analyze XML file with no date in obsolete message", t, func() {
		results := analyzeTestFile(t, "obsolete/obsolete_no_date.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Obsolete"),
				Message:   obsoleteDateError,
				StartLine: 4,
				Path:      "testdata/src/obsolete/obsolete_no_date.xml",
			},
			defaultExpiryInfo("testdata/src/obsolete/obsolete_no_date.xml"),
		})
	})

	Convey("Analyze XML file with badly formatted date in obsolete message", t, func() {
		currPath := "obsolete/obsolete_unformatted_date.xml"
		fullPath := "testdata/src/" + currPath
		results := analyzeTestFile(t, currPath, emptyPatch, tempDir)
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
		results := analyzeTestFile(t, "owners/good_individuals.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("testdata/src/owners/good_individuals.xml"),
		})
	})

	Convey("Analyze XML file with error: only one owner", t, func() {
		results := analyzeTestFile(t, "owners/one_owner.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Owners"),
				Message:   oneOwnerError,
				StartLine: 4,
				Path:      "testdata/src/owners/one_owner.xml",
			},
			defaultExpiryInfo("testdata/src/owners/one_owner.xml"),
		})
	})

	Convey("Analyze XML file with error: no owners", t, func() {
		results := analyzeTestFile(t, "owners/no_owners.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Owners"),
				Message:   oneOwnerError,
				StartLine: 3,
				Path:      "testdata/src/owners/no_owners.xml",
			},
			defaultExpiryInfo("testdata/src/owners/no_owners.xml"),
		})
	})

	Convey("Analyze XML file with error: first owner is team", t, func() {
		results := analyzeTestFile(t, "owners/first_team_owner.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Owners"),
				Message:   firstOwnerTeamError,
				StartLine: 4,
				Path:      "testdata/src/owners/first_team_owner.xml",
			},
			defaultExpiryInfo("testdata/src/owners/first_team_owner.xml"),
		})
	})

	Convey("Analyze XML file with multiple owner errors", t, func() {
		results := analyzeTestFile(t, "owners/first_team_one_owner.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Owners"),
				Message:   oneOwnerError,
				StartLine: 4,
				Path:      "testdata/src/owners/first_team_one_owner.xml",
			},
			{
				Category:  fmt.Sprintf("%s/%s", category, "Owners"),
				Message:   firstOwnerTeamError,
				StartLine: 4,
				Path:      "testdata/src/owners/first_team_one_owner.xml",
			},
			defaultExpiryInfo("testdata/src/owners/first_team_one_owner.xml"),
		})
	})

	// UNITS tests
	Convey("Analyze XML file no errors, units of microseconds, all users", t, func() {
		results := analyzeTestFile(t, "units/microseconds_all_users.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("testdata/src/units/microseconds_all_users.xml"),
		})
	})

	Convey("Analyze XML file no errors, units of microseconds, high-resolution", t, func() {
		results := analyzeTestFile(t, "units/microseconds_high_res.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("testdata/src/units/microseconds_high_res.xml"),
		})
	})

	Convey("Analyze XML file no errors, units of microseconds, low-resolution", t, func() {
		results := analyzeTestFile(t, "units/microseconds_low_res.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfo("testdata/src/units/microseconds_low_res.xml"),
		})
	})

	Convey("Analyze XML file with error: units of microseconds, bad summary", t, func() {
		results := analyzeTestFile(t, "units/microseconds_bad_summary.xml", emptyPatch, tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  fmt.Sprintf("%s/%s", category, "Units"),
				Message:   unitsMicrosecondsWarning,
				StartLine: 3,
				Path:      "testdata/src/units/microseconds_bad_summary.xml",
			},
			defaultExpiryInfo("testdata/src/units/microseconds_bad_summary.xml"),
		})
	})

	// REMOVED HISTOGRAM tests
	Convey("Analyze XML file with error: histogram deleted", t, func() {
		results := analyzeTestFile(t, "rm/remove_histogram.xml", "testdata/tricium_generated_diff.patch", tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category: fmt.Sprintf("%s/%s", category, "Removed"),
				Message:  fmt.Sprintf(removedHistogramError, "Test.Histogram2"),
				Path:     "testdata/src/rm/remove_histogram.xml",
			},
		})
	})

	// ADDED NAMESPACE tests
	Convey("Analyze XML file with no error: added histogram with same namespace", t, func() {
		results := analyzeTestFile(t, "namespace/same_namespace.xml", "testdata/tricium_same_namespace.patch", tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfoLine("testdata/src/namespace/same_namespace.xml", 8),
		})
	})

	Convey("Analyze XML file with warning: added namespace", t, func() {
		results := analyzeTestFile(t, "namespace/add_namespace.xml", "testdata/tricium_namespace_diff.patch", tempDir)
		So(results, ShouldResemble, []*tricium.Data_Comment{
			defaultExpiryInfoLine("testdata/src/namespace/add_namespace.xml", 8),
			{
				Category:  fmt.Sprintf("%s/%s", category, "Namespace"),
				Message:   fmt.Sprintf(addedNamespaceWarning, "Test2"),
				Path:      "testdata/src/namespace/add_namespace.xml",
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
		Category:  fmt.Sprintf("%s/%s", category, "Expiry"),
		Message:   fmt.Sprintf("[INFO]: Expiry date is in 104 days"),
		StartLine: int32(startLine),
		Path:      path,
	}
}

func makeObsoleteDateError(path string, startLine int) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  fmt.Sprintf("%s/%s", category, "Obsolete"),
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
