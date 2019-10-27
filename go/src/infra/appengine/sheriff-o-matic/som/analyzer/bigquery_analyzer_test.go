package analyzer

import (
	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"regexp"
	"sort"
	"strings"
	"testing"

	"infra/appengine/sheriff-o-matic/som/analyzer/step"
	"infra/monitoring/messages"

	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/luci/common/logging/gologger"
)

type mockResults struct {
	failures []failureRow
	err      error
	curr     int
}

func (m *mockResults) Next(dst interface{}) error {
	if m.curr >= len(m.failures) {
		return iterator.Done
	}
	fdst := dst.(*failureRow)
	*fdst = m.failures[m.curr]
	m.curr++
	return m.err
}

func TestMockBQResults(t *testing.T) {
	Convey("no results", t, func() {
		mr := &mockResults{}
		r := &failureRow{}
		So(mr.Next(r), ShouldEqual, iterator.Done)
	})
	Convey("copy op works", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "foo",
				},
			},
		}
		r := failureRow{}
		err := mr.Next(&r)
		So(err, ShouldBeNil)
		So(r.StepName, ShouldEqual, "foo")
		So(mr.Next(&r), ShouldEqual, iterator.Done)
	})

}

func TestGenerateBuilderURL(t *testing.T) {
	Convey("Test builder with no space", t, func() {
		project := "chromium"
		bucket := "ci"
		builderName := "Win"
		url := generateBuilderURL(project, bucket, builderName)
		So(url, ShouldEqual, "https://ci.chromium.org/p/chromium/builders/ci/Win")
	})
	Convey("Test builder with some spaces", t, func() {
		project := "chromium"
		bucket := "ci"
		builderName := "Win 7 Test"
		url := generateBuilderURL(project, bucket, builderName)
		So(url, ShouldEqual, "https://ci.chromium.org/p/chromium/builders/ci/Win%207%20Test")
	})
	Convey("Test builder with special characters", t, func() {
		project := "chromium"
		bucket := "ci"
		builderName := "Mac 10.13 Tests (dbg)"
		url := generateBuilderURL(project, bucket, builderName)
		So(url, ShouldEqual, "https://ci.chromium.org/p/chromium/builders/ci/Mac%2010.13%20Tests%20%28dbg%29")
	})
}

func TestGenerateBuildURL(t *testing.T) {
	Convey("Test build url with build ID", t, func() {
		project := "chromium"
		bucket := "ci"
		builderName := "Win"
		buildID := bigquery.NullInt64{Int64: 8127364737474, Valid: true}
		url := generateBuildURL(project, bucket, builderName, buildID)
		So(url, ShouldEqual, "https://ci.chromium.org/p/chromium/builders/ci/Win/b8127364737474")
	})
	Convey("Test build url with empty buildID", t, func() {
		project := "chromium"
		bucket := "ci"
		builderName := "Win"
		buildID := bigquery.NullInt64{}
		url := generateBuildURL(project, bucket, builderName, buildID)
		So(url, ShouldEqual, "")
	})
}

// Make SQL query uniform, for the purpose of testing
func formatQuery(query string) string {
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")
	query = regexp.MustCompile(`\s?\(\s?`).ReplaceAllString(query, "(")
	query = regexp.MustCompile(`\s?\)\s?`).ReplaceAllString(query, ")")
	return query
}

func TestGenerateSQLQuery(t *testing.T) {
	Convey("Test generate SQL query for android", t, func() {
		expected := `
			SELECT
			  Project,
			  Bucket,
			  Builder,
			  MasterName,
			  StepName,
			  TestNamesFingerprint,
			  TestNamesTrunc,
			  NumTests,
			  BuildIdBegin,
			  BuildIdEnd,
			  BuildNumberBegin,
			  BuildNumberEnd,
			  CPRangeOutputBegin,
			  CPRangeOutputEnd,
			  CPRangeInputBegin,
			  CPRangeInputEnd,
			  CulpritIdRangeBegin,
			  CulpritIdRangeEnd,
			  StartTime,
			  BuildStatus
			FROM
				` + "`sheriff-o-matic.chrome.sheriffable_failures`" + `
			WHERE
			  MasterName IN (
			    "internal.client.clank",
			    "internal.client.clank_tot",
			    "chromium.android")
			  OR (
			    MasterName = "chromium"
			    AND builder="Android")
			  OR (
			    MasterName = "chromium.webkit"
			    AND builder IN (
			      "Android Builder",
				    "Webkit Android (Nexus4)"))
			  OR (
			    MasterName = "official.chrome"
				  AND builder IN (
					  "android-arm-official-tests",
					  "android-arm64-official-tests",
					  "android-arm-beta-tests",
					  "android-arm64-beta-tests",
					  "android-arm-stable-tests",
					  "android-arm64-stable-tests"))
			  OR (
			    MasterName = "official.chrome.continuous"
				  AND builder IN (
					  "android-arm-beta",
					  "android-arm64-beta",
					  "android-arm64-stable",
					  "android-arm64-stable",
					  "android-arm-stable"))
			LIMIT
				1000
		`
		actual := generateSQLQuery("android", "sheriff-o-matic")
		So(formatQuery(actual), ShouldEqual, formatQuery(expected))
	})
	Convey("Test generate SQL query for chromium", t, func() {
		expected := `
			SELECT
			  Project,
			  Bucket,
			  Builder,
			  MasterName,
			  StepName,
			  TestNamesFingerprint,
			  TestNamesTrunc,
			  NumTests,
			  BuildIdBegin,
			  BuildIdEnd,
			  BuildNumberBegin,
			  BuildNumberEnd,
			  CPRangeOutputBegin,
			  CPRangeOutputEnd,
			  CPRangeInputBegin,
			  CPRangeInputEnd,
			  CulpritIdRangeBegin,
			  CulpritIdRangeEnd,
			  StartTime,
			  BuildStatus
			FROM
				` + "`sheriff-o-matic.chrome.sheriffable_failures`" + `
			WHERE
			MasterName IN(
				"chrome",
				"chromium",
				"chromium.chromiumos",
				"chromium.gpu",
				"chromium.linux",
				"chromium.mac",
				"chromium.memory",
				"chromium.win"
				)
			AND Bucket = "ci"
			LIMIT
				1000
		`
		actual := generateSQLQuery("chromium", "sheriff-o-matic")
		So(formatQuery(actual), ShouldEqual, formatQuery(expected))
	})
	Convey("Test generate SQL query for chromium.gpu.fyi", t, func() {
		expected := `
			SELECT
			  Project,
			  Bucket,
			  Builder,
			  MasterName,
			  StepName,
			  TestNamesFingerprint,
			  TestNamesTrunc,
			  NumTests,
			  BuildIdBegin,
			  BuildIdEnd,
			  BuildNumberBegin,
			  BuildNumberEnd,
			  CPRangeOutputBegin,
			  CPRangeOutputEnd,
			  CPRangeInputBegin,
			  CPRangeInputEnd,
			  CulpritIdRangeBegin,
			  CulpritIdRangeEnd,
			  StartTime,
			  BuildStatus
			FROM
				` + "`sheriff-o-matic.chromium.sheriffable_failures`" + `
			WHERE MasterName = "chromium.gpu.fyi"
		`
		actual := generateSQLQuery("chromium.gpu.fyi", "sheriff-o-matic")
		So(formatQuery(actual), ShouldEqual, formatQuery(expected))
	})
	Convey("Test generate SQL query for chromeos", t, func() {
		expected := `
			SELECT
			  Project,
			  Bucket,
			  Builder,
			  MasterName,
			  StepName,
			  TestNamesFingerprint,
			  TestNamesTrunc,
			  NumTests,
			  BuildIdBegin,
			  BuildIdEnd,
			  BuildNumberBegin,
			  BuildNumberEnd,
			  CPRangeOutputBegin,
			  CPRangeOutputEnd,
			  CPRangeInputBegin,
			  CPRangeInputEnd,
			  CulpritIdRangeBegin,
			  CulpritIdRangeEnd,
			  StartTime,
			  BuildStatus
			FROM
				` + "`sheriff-o-matic.chromeos.sheriffable_failures`" + `
			WHERE project = "chromeos"
				AND bucket IN ("postsubmit", "annealing")
				AND (critical != "NO" OR critical is NULL)
		`
		actual := generateSQLQuery("chromeos", "sheriff-o-matic")
		So(formatQuery(actual), ShouldEqual, formatQuery(expected))
	})
	Convey("Test generate SQL query for ios", t, func() {
		expected := `
			SELECT
			  Project,
			  Bucket,
			  Builder,
			  MasterName,
			  StepName,
			  TestNamesFingerprint,
			  TestNamesTrunc,
			  NumTests,
			  BuildIdBegin,
			  BuildIdEnd,
			  BuildNumberBegin,
			  BuildNumberEnd,
			  CPRangeOutputBegin,
			  CPRangeOutputEnd,
			  CPRangeInputBegin,
			  CPRangeInputEnd,
			  CulpritIdRangeBegin,
			  CulpritIdRangeEnd,
			  StartTime,
			  BuildStatus
			FROM
				` + "`sheriff-o-matic.chromium.sheriffable_failures`" + `
			WHERE project = "chromium"
				AND MasterName IN ("chromium.mac")
				AND builder IN (
					"ios-device",
					"ios-device-xcode-clang",
					"ios-simulator",
					"ios-simulator-full-configs",
					"ios-simulator-xcode-clang"
				)
		`
		actual := generateSQLQuery("ios", "sheriff-o-matic")
		So(formatQuery(actual), ShouldEqual, formatQuery(expected))
	})
	Convey("Test generate SQL query for fuchsia", t, func() {
		expected := `
			SELECT
			  Project,
			  Bucket,
			  Builder,
			  MasterName,
			  StepName,
			  TestNamesFingerprint,
			  TestNamesTrunc,
			  NumTests,
			  BuildIdBegin,
			  BuildIdEnd,
			  BuildNumberBegin,
			  BuildNumberEnd,
			  CPRangeOutputBegin,
			  CPRangeOutputEnd,
			  CPRangeInputBegin,
			  CPRangeInputEnd,
			  CulpritIdRangeBegin,
			  CulpritIdRangeEnd,
			  StartTime,
			  BuildStatus
			FROM
				` + "`sheriff-o-matic.fuchsia.sheriffable_failures`" + `
			WHERE
				(Project = "fuchsia" OR MasterName = "fuchsia")
				AND Bucket NOT IN ("try", "cq", "staging", "general")
				AND Builder NOT LIKE "%bisect%"
			LIMIT
				1000
		`
		actual := generateSQLQuery("fuchsia", "sheriff-o-matic")
		So(formatQuery(actual), ShouldEqual, formatQuery(expected))
	})
	Convey("Test generate SQL query for chromium.perf", t, func() {
		expected := `
			SELECT
			  Project,
			  Bucket,
			  Builder,
			  MasterName,
			  StepName,
			  TestNamesFingerprint,
			  TestNamesTrunc,
			  NumTests,
			  BuildIdBegin,
			  BuildIdEnd,
			  BuildNumberBegin,
			  BuildNumberEnd,
			  CPRangeOutputBegin,
			  CPRangeOutputEnd,
			  CPRangeInputBegin,
			  CPRangeInputEnd,
			  CulpritIdRangeBegin,
			  CulpritIdRangeEnd,
			  StartTime,
			  BuildStatus
			FROM
				` + "`sheriff-o-matic.chrome.sheriffable_failures`" + `
			WHERE
				(Project = "chromium.perf" OR MasterName = "chromium.perf")
				AND Bucket NOT IN ("try", "cq", "staging", "general")
				AND (Mastername IS NULL OR Mastername NOT LIKE "%.fyi")
				AND Builder NOT LIKE "%bisect%"
			LIMIT
				1000
		`
		actual := generateSQLQuery("chromium.perf", "sheriff-o-matic")
		So(formatQuery(actual), ShouldEqual, formatQuery(expected))
	})
	Convey("Test generate SQL query for other builders", t, func() {
		expected := `
			SELECT
			  Project,
			  Bucket,
			  Builder,
			  MasterName,
			  StepName,
			  TestNamesFingerprint,
			  TestNamesTrunc,
			  NumTests,
			  BuildIdBegin,
			  BuildIdEnd,
			  BuildNumberBegin,
			  BuildNumberEnd,
			  CPRangeOutputBegin,
			  CPRangeOutputEnd,
			  CPRangeInputBegin,
			  CPRangeInputEnd,
			  CulpritIdRangeBegin,
			  CulpritIdRangeEnd,
			  StartTime,
			  BuildStatus
			FROM
				` + "`sheriff-o-matic.chromium.sheriffable_failures`" + `
			WHERE
				(Project = "builder" OR MasterName = "builder")
				AND Bucket NOT IN ("try", "cq", "staging", "general")
				AND (Mastername IS NULL OR Mastername NOT LIKE "%.fyi")
				AND Builder NOT LIKE "%bisect%"
			LIMIT
				1000
		`
		actual := generateSQLQuery("builder", "sheriff-o-matic")
		So(formatQuery(actual), ShouldEqual, formatQuery(expected))
	})
}

func TestProcessBQResults(t *testing.T) {
	ctx := context.Background()
	ctx = gologger.StdConfig.Use(ctx)

	Convey("smoke", t, func() {
		mr := &mockResults{}
		got, err := processBQResults(ctx, mr)
		So(err, ShouldEqual, nil)
		So(got, ShouldBeEmpty)
	})

	Convey("single result, only start/end build numbers", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "some step",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "some builder",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 1)
	})

	Convey("single result, only end build number", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "some step",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "some builder",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 1)
	})

	Convey("single result, start/end build numbers, single test name", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "some step",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "some builder",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 1)
		reason := got[0].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "some step",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 1,
			Tests: []step.TestWithResult{{
				TestName: "some/test/name",
			}},
		})
		So(len(got[0].Builders), ShouldEqual, 1)
	})

	Convey("multiple results, start/end build numbers, same step, same test name", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "some step",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "builder 1",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
				},
				{
					StepName: "some step",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "builder 2",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 1)
		reason := got[0].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "some step",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 1,
			Tests: []step.TestWithResult{{
				TestName: "some/test/name",
			}},
		})
		So(len(got[0].Builders), ShouldEqual, 2)
	})

	Convey("multiple results, start/end build numbers, different steps, different sets of test names", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "some step 1",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "builder 1",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name/1\nsome/test/name/2",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 2,
						Valid: true,
					},
				},
				{
					StepName: "some step 2",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "builder 2",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 2,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name/3",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		sort.Sort(byStepName(got))
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 2)

		reason := got[0].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "some step 1",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 2,
			Tests: []step.TestWithResult{{
				TestName: "some/test/name/1",
			},
				{
					TestName: "some/test/name/2",
				}},
		})
		So(len(got[0].Builders), ShouldEqual, 1)

		reason = got[1].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "some step 2",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 1,
			Tests: []step.TestWithResult{{
				TestName: "some/test/name/3",
			}},
		})
		So(len(got[0].Builders), ShouldEqual, 1)
	})

	Convey("multiple results, start/end build numbers, same step, different sets of test names", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "some step 1",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "builder 1",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name/1\nsome/test/name/2",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 2,
						Valid: true,
					},
				},
				{
					StepName: "some step 1",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "builder 2",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 2,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name/3",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		sort.Sort(byTests(got))
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 2)

		reason := got[0].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "some step 1",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 2,
			Tests: []step.TestWithResult{{
				TestName: "some/test/name/1",
			},
				{
					TestName: "some/test/name/2",
				}},
		})
		So(len(got[0].Builders), ShouldEqual, 1)
		So(got[0].Builders[0].Name, ShouldEqual, "builder 1")

		reason = got[1].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "some step 1",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 1,
			Tests: []step.TestWithResult{{
				TestName: "some/test/name/3",
			}},
		})
		So(len(got[1].Builders), ShouldEqual, 1)
		So(got[1].Builders[0].Name, ShouldEqual, "builder 2")
	})

	Convey("chromium.perf case: multiple results, different start build numbers, same end build number, same step, different sets of test names", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "performance_test_suite",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "win-10-perf",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 100,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 110,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "A1\nA2\nA3",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 3,
						Valid: true,
					},
				},
				{
					StepName: "performance_test_suite",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "win-10-perf",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 102,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 110,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 2,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "B1\nB2\nB3",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 3,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		sort.Sort(byTests(got))
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 2)

		reason := got[0].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "performance_test_suite",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 3,
			Tests: []step.TestWithResult{
				{
					TestName: "A1",
				},
				{
					TestName: "A2",
				},
				{
					TestName: "A3",
				},
			},
		})
		So(len(got[0].Builders), ShouldEqual, 1)
		So(got[0].Builders[0].Name, ShouldEqual, "win-10-perf")
		So(got[0].Builders[0].FirstFailure, ShouldEqual, 100)
		So(got[0].Builders[0].LatestFailure, ShouldEqual, 110)

		reason = got[1].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "performance_test_suite",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 3,
			Tests: []step.TestWithResult{
				{
					TestName: "B1",
				},
				{
					TestName: "B2",
				},
				{
					TestName: "B3",
				},
			},
		})
		So(len(got[1].Builders), ShouldEqual, 1)
		So(got[1].Builders[0].Name, ShouldEqual, "win-10-perf")
		So(got[1].Builders[0].FirstFailure, ShouldEqual, 102)
		So(got[1].Builders[0].LatestFailure, ShouldEqual, 110)
	})

	Convey("chromium.perf case: multiple results, same step, same truncated list of test names, different test name fingerprints", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "performance_test_suite",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "win-10-perf",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 100,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 110,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "A1\nA2\nA3",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 3,
						Valid: true,
					},
				},
				{
					StepName: "performance_test_suite",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "win-10-perf",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 102,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 110,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 2,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "A1\nA2\nA3",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 3,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		sort.Sort(byFirstFailure(got))
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 2)

		reason := got[0].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "performance_test_suite",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 3,
			Tests: []step.TestWithResult{
				{
					TestName: "A1",
				},
				{
					TestName: "A2",
				},
				{
					TestName: "A3",
				},
			},
		})
		So(len(got[0].Builders), ShouldEqual, 1)

		So(got[0].Builders[0].Name, ShouldEqual, "win-10-perf")
		So(got[0].Builders[0].FirstFailure, ShouldEqual, 100)
		So(got[0].Builders[0].LatestFailure, ShouldEqual, 110)

		reason = got[1].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "performance_test_suite",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 3,
			Tests: []step.TestWithResult{
				{
					TestName: "A1",
				},
				{
					TestName: "A2",
				},
				{
					TestName: "A3",
				},
			},
		})
		So(len(got[1].Builders), ShouldEqual, 1)
		So(got[1].Builders[0].Name, ShouldEqual, "win-10-perf")
		So(got[1].Builders[0].FirstFailure, ShouldEqual, 102)
		So(got[1].Builders[0].LatestFailure, ShouldEqual, 110)
	})

	Convey("multiple results, start/end build numbers, different steps, same set of test names", t, func() {
		mr := &mockResults{
			failures: []failureRow{
				{
					StepName: "some step 1",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "builder 1",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name/1\nsome/test/name/2",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 2,
						Valid: true,
					},
				},
				{
					StepName: "some step 2",
					MasterName: bigquery.NullString{
						StringVal: "some master",
						Valid:     true,
					},
					Builder: "builder 2",
					Project: "some project",
					Bucket:  "some bucket",
					BuildIDBegin: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					BuildIDEnd: bigquery.NullInt64{
						Int64: 10,
						Valid: true,
					},
					TestNamesFingerprint: bigquery.NullInt64{
						Int64: 1,
						Valid: true,
					},
					TestNamesTrunc: bigquery.NullString{
						StringVal: "some/test/name/1\nsome/test/name/2",
						Valid:     true,
					},
					NumTests: bigquery.NullInt64{
						Int64: 2,
						Valid: true,
					},
				},
			},
		}
		got, err := processBQResults(ctx, mr)
		sort.Sort(byStepName(got))
		So(err, ShouldEqual, nil)
		So(len(got), ShouldEqual, 2)
		reason := got[0].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "some step 1",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 2,
			Tests: []step.TestWithResult{{
				TestName: "some/test/name/1",
			},
				{
					TestName: "some/test/name/2",
				}},
		})
		So(len(got[0].Builders), ShouldEqual, 1)

		reason = got[1].Reason
		So(reason, ShouldNotBeNil)
		So(reason.Raw, ShouldResemble, &bqFailure{
			Name:            "some step 2",
			kind:            "test",
			severity:        messages.ReliableFailure,
			NumFailingTests: 2,
			Tests: []step.TestWithResult{{
				TestName: "some/test/name/1",
			},
				{
					TestName: "some/test/name/2",
				}},
		})
		So(len(got[1].Builders), ShouldEqual, 1)
	})
}

type byFirstFailure []*messages.BuildFailure

func (f byFirstFailure) Len() int      { return len(f) }
func (f byFirstFailure) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f byFirstFailure) Less(i, j int) bool {
	return f[i].Builders[0].FirstFailure < f[j].Builders[0].FirstFailure
}

type byTests []*messages.BuildFailure

func (f byTests) Len() int      { return len(f) }
func (f byTests) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f byTests) Less(i, j int) bool {
	iTests, jTests := []string{}, []string{}
	for _, t := range f[i].Reason.Raw.(*bqFailure).Tests {
		iTests = append(iTests, t.TestName)
	}
	for _, t := range f[j].Reason.Raw.(*bqFailure).Tests {
		jTests = append(jTests, t.TestName)
	}

	return strings.Join(iTests, "\n") < strings.Join(jTests, "\n")
}

func TestFilterHierarchicalSteps(t *testing.T) {
	Convey("smoke", t, func() {
		failures := []*messages.BuildFailure{}
		got := filterHierarchicalSteps(failures)
		So(len(got), ShouldEqual, 0)
	})

	Convey("single step, single builder", t, func() {
		failures := []*messages.BuildFailure{
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results",
					},
				},
			},
		}

		got := filterHierarchicalSteps(failures)
		So(len(got), ShouldEqual, 1)
		So(len(got[0].Builders), ShouldEqual, 1)
	})

	Convey("nested step, single builder", t, func() {
		failures := []*messages.BuildFailure{
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results|build results",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results|build results|chromeos.postsubmit.beaglebone_servo-postsubmit",
					},
				},
			},
		}

		got := filterHierarchicalSteps(failures)
		So(len(got), ShouldEqual, 1)
		So(len(got[0].Builders), ShouldEqual, 1)
	})

	Convey("single step, multiple builders", t, func() {
		failures := []*messages.BuildFailure{
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results",
					},
				},
			},
		}

		got := filterHierarchicalSteps(failures)
		So(len(got), ShouldEqual, 1)
		So(len(got[0].Builders), ShouldEqual, 2)
	})

	Convey("nested step, multiple builder", t, func() {
		failures := []*messages.BuildFailure{
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results|build results",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results|build results|chromeos.postsubmit.beaglebone_servo-postsubmit",
					},
				},
			},
		}

		got := filterHierarchicalSteps(failures)
		So(len(got), ShouldEqual, 1)
		So(len(got[0].Builders), ShouldEqual, 2)
		So(got[0].StepAtFault.Step.Name, ShouldEqual, "check build results|build results|chromeos.postsubmit.beaglebone_servo-postsubmit")
	})

	Convey("mixed nested steps, multiple builder", t, func() {
		failures := []*messages.BuildFailure{
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "test foo",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "test bar",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "test baz",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results|build results",
					},
				},
			},
			{
				Builders: []*messages.AlertedBuilder{
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name A",
					},
					{
						Project: "project",
						Bucket:  "bucket",
						Name:    "builder name B",
					},
				},
				StepAtFault: &messages.BuildStep{
					Step: &messages.Step{
						Name: "check build results|build results|chromeos.postsubmit.beaglebone_servo-postsubmit",
					},
				},
			},
		}

		got := filterHierarchicalSteps(failures)
		So(len(got), ShouldEqual, 4)
		So(len(got[0].Builders), ShouldEqual, 2)
		So(got[0].StepAtFault.Step.Name, ShouldEqual, "test foo")
		So(len(got[1].Builders), ShouldEqual, 1)
		So(got[1].StepAtFault.Step.Name, ShouldEqual, "test bar")
		So(len(got[2].Builders), ShouldEqual, 1)
		So(got[2].StepAtFault.Step.Name, ShouldEqual, "test baz")
		So(len(got[3].Builders), ShouldEqual, 2)
		So(got[3].StepAtFault.Step.Name, ShouldEqual, "check build results|build results|chromeos.postsubmit.beaglebone_servo-postsubmit")
	})
}
