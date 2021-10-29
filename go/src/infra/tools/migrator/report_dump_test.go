// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

import (
	"bytes"
	"encoding/csv"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func mkReport(row ...string) *Report {
	ret, err := NewReportFromCSVRow(row)
	So(err, ShouldBeNil)
	return ret
}

func TestReportDump(t *testing.T) {
	t.Parallel()

	Convey(`ReportDump`, t, func() {
		rd := &ReportDump{}

		So(rd.Empty(), ShouldBeTrue)

		Convey(`Add`, func() {
			rd.Add(
				mkReport("checkout", "proj-foo", "some.file", "TAG_THIRD", "another problem", "true"),
				mkReport("checkout", "proj-foo", "", "TAG", "problem", "true"),
				mkReport("checkout", "proj-foo", "", "TAG_OTHER", "problem", "true"),
				mkReport("checkout", "other-proj", "", "TAG", "problem", "true"),
			)

			So(rd.Empty(), ShouldBeFalse)
			So(rd.data, ShouldHaveLength, 3)
			So(rd.data[ReportID{"checkout", "proj-foo", ""}], ShouldHaveLength, 2)
			So(rd.data[ReportID{"checkout", "proj-foo", ""}][0].Tag, ShouldResemble, "TAG")
			So(rd.data[ReportID{"checkout", "proj-foo", ""}][1].Tag, ShouldResemble, "TAG_OTHER")
			So(rd.data[ReportID{"checkout", "proj-foo", "some.file"}][0].Tag, ShouldResemble, "TAG_THIRD")
			So(rd.data[ReportID{"checkout", "other-proj", ""}], ShouldHaveLength, 1)
			So(rd.data[ReportID{"checkout", "other-proj", ""}][0].Tag, ShouldResemble, "TAG")

			Convey(`Iterate`, func() {
				reports := []*Report{}
				rd.Iterate(func(rid ReportID, reps []*Report) bool {
					for _, r := range reps {
						So(rid, ShouldResemble, r.ReportID)
					}
					reports = append(reports, reps...)
					return true
				})

				So(reports, ShouldHaveLength, 4)
				So(reports[0].ReportID, ShouldResemble, ReportID{"checkout", "other-proj", ""})
				So(reports[1].ReportID, ShouldResemble, ReportID{"checkout", "proj-foo", ""})
				So(reports[2].ReportID, ShouldResemble, ReportID{"checkout", "proj-foo", ""})
				So(reports[3].ReportID, ShouldResemble, ReportID{"checkout", "proj-foo", "some.file"})
			})

			Convey(`UpdateFrom`, func() {
				So(rd.UpdateFrom(rd.Clone()), ShouldEqual, 4)
				So(rd.data, ShouldHaveLength, 3)
				So(rd.data[ReportID{"checkout", "proj-foo", ""}], ShouldHaveLength, 4)
				So(rd.data[ReportID{"checkout", "proj-foo", ""}][0].Tag, ShouldResemble, "TAG")
				So(rd.data[ReportID{"checkout", "proj-foo", ""}][1].Tag, ShouldResemble, "TAG_OTHER")
				So(rd.data[ReportID{"checkout", "proj-foo", ""}][2].Tag, ShouldResemble, "TAG")
				So(rd.data[ReportID{"checkout", "proj-foo", ""}][3].Tag, ShouldResemble, "TAG_OTHER")
				So(rd.data[ReportID{"checkout", "proj-foo", "some.file"}][0].Tag, ShouldResemble, "TAG_THIRD")
				So(rd.data[ReportID{"checkout", "proj-foo", "some.file"}][1].Tag, ShouldResemble, "TAG_THIRD")
				So(rd.data[ReportID{"checkout", "other-proj", ""}], ShouldHaveLength, 2)
				So(rd.data[ReportID{"checkout", "other-proj", ""}][0].Tag, ShouldResemble, "TAG")
				So(rd.data[ReportID{"checkout", "other-proj", ""}][1].Tag, ShouldResemble, "TAG")
			})

			Convey(`Clone`, func() {
				c := rd.Clone()
				So(c.data, ShouldNotEqual, rd.data)
				So(c.data, ShouldResemble, rd.data)
			})
		})
	})
}

func TestReportCSV(t *testing.T) {
	t.Parallel()

	Convey(`Report CSV`, t, func() {
		Convey(`Write`, func() {
			rd := &ReportDump{}
			rd.Add(
				mkReport("checkout", "proj-foo", "some.file", "TAG_THIRD", "another problem", "true"),
				mkReport("checkout", "proj-foo", "", "TAG", "problem", "true", "meta:data", "a:value"),
				mkReport("checkout", "proj-foo", "", "TAG_OTHER", "problem", "true"),
				mkReport("checkout", "other-proj", "", "TAG", "problem", "true"),
				mkReport("checkout", "a-third-prog", "", "TAG", "problem", "true"),
			)

			buf := &bytes.Buffer{}
			So(rd.WriteToCSV(buf), ShouldBeNil)

			csvRead := csv.NewReader(buf)
			csvRead.FieldsPerRecord = -1 // variable
			lines, err := csvRead.ReadAll()
			So(err, ShouldBeNil)
			So(lines, ShouldResemble, [][]string{
				csvHeader,
				{"checkout", "a-third-prog", "", "TAG", "problem", "true"},
				{"checkout", "other-proj", "", "TAG", "problem", "true"},
				{"checkout", "proj-foo", "", "TAG", "problem", "true", "a:value", "meta:data"},
				{"checkout", "proj-foo", "", "TAG_OTHER", "problem", "true"},
				{"checkout", "proj-foo", "some.file", "TAG_THIRD", "another problem", "true"},
			})
		})

		Convey(`Read`, func() {
			Convey(`OK`, func() {
				buf := &bytes.Buffer{}
				csvWrite := csv.NewWriter(buf)
				csvWrite.WriteAll([][]string{
					csvHeader,
					{"checkout", "a-third-prog", "", "TAG", "problem", "true"},
					{"checkout", "other-proj", "", "TAG", "problem", "true"},
					{"checkout", "proj-foo", "", "TAG", "problem", "true", "a:value", "meta:data"},
					{"checkout", "proj-foo", "", "TAG_OTHER", "problem", "true"},
					{"checkout", "proj-foo", "some.file", "TAG_THIRD", "another problem", "true"},
				})
				csvWrite.Flush()

				rd, err := NewReportDumpFromCSV(buf)
				So(err, ShouldBeNil)
				So(rd.data, ShouldResemble, map[ReportID][]*Report{
					{"checkout", "proj-foo", ""}: {
						mkReport("checkout", "proj-foo", "", "TAG", "problem", "true", "meta:data", "a:value"),
						mkReport("checkout", "proj-foo", "", "TAG_OTHER", "problem", "true"),
					},
					{"checkout", "proj-foo", "some.file"}: {
						mkReport("checkout", "proj-foo", "some.file", "TAG_THIRD", "another problem", "true"),
					},
					{"checkout", "other-proj", ""}: {
						mkReport("checkout", "other-proj", "", "TAG", "problem", "true"),
					},
					{"checkout", "a-third-prog", ""}: {
						mkReport("checkout", "a-third-prog", "", "TAG", "problem", "true"),
					},
				})
			})

			Convey(`Empty Header`, func() {
				buf := &bytes.Buffer{}
				buf.WriteString("\n")

				_, err := NewReportDumpFromCSV(buf)
				So(err, ShouldErrLike, "header was missing")
			})

			Convey(`Bad Schema`, func() {
				buf := &bytes.Buffer{}
				csvWrite := csv.NewWriter(buf)
				header := append([]string(nil), csvHeader[:len(csvHeader)-1]...)
				header = append(header, "{schema=v4}")
				csvWrite.Write(header)
				csvWrite.Flush()

				_, err := NewReportDumpFromCSV(buf)
				So(err, ShouldErrLike, "unexpected version: \"v4\", expected \"v3\"")
			})

			Convey(`Bad Header length`, func() {
				buf := &bytes.Buffer{}
				csvWrite := csv.NewWriter(buf)
				csvWrite.WriteAll([][]string{
					{"some", "stuff"},
				})
				csvWrite.Flush()

				_, err := NewReportDumpFromCSV(buf)
				So(err, ShouldErrLike, "unexpected header:")
			})

			Convey(`Bad Header content`, func() {
				buf := &bytes.Buffer{}
				csvWrite := csv.NewWriter(buf)
				csvWrite.WriteAll([][]string{
					{"1", "2", "3", "4", "5", "6"},
				})
				csvWrite.Flush()

				_, err := NewReportDumpFromCSV(buf)
				So(err, ShouldErrLike, "unexpected header:")
			})

			Convey(`Bad Row`, func() {
				buf := &bytes.Buffer{}
				csvWrite := csv.NewWriter(buf)
				csvWrite.WriteAll([][]string{
					csvHeader,
					{"wat"},
				})
				csvWrite.Flush()

				_, err := NewReportDumpFromCSV(buf)
				So(err, ShouldErrLike, "reading row 2")
			})

		})
	})
}
