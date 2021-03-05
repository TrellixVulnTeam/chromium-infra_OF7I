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
				mkReport("proj-foo", "some.file", "TAG_THIRD", "another problem"),
				mkReport("proj-foo", "", "TAG", "problem"),
				mkReport("proj-foo", "", "TAG_OTHER", "problem"),
				mkReport("other-proj", "", "TAG", "problem"),
			)

			So(rd.Empty(), ShouldBeFalse)
			So(rd.data, ShouldHaveLength, 3)
			So(rd.data[ReportID{"proj-foo", ""}], ShouldHaveLength, 2)
			So(rd.data[ReportID{"proj-foo", ""}][0].Tag, ShouldResemble, "TAG")
			So(rd.data[ReportID{"proj-foo", ""}][1].Tag, ShouldResemble, "TAG_OTHER")
			So(rd.data[ReportID{"proj-foo", "some.file"}][0].Tag, ShouldResemble, "TAG_THIRD")
			So(rd.data[ReportID{"other-proj", ""}], ShouldHaveLength, 1)
			So(rd.data[ReportID{"other-proj", ""}][0].Tag, ShouldResemble, "TAG")

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
				So(reports[0].ReportID, ShouldResemble, ReportID{"other-proj", ""})
				So(reports[1].ReportID, ShouldResemble, ReportID{"proj-foo", ""})
				So(reports[2].ReportID, ShouldResemble, ReportID{"proj-foo", ""})
				So(reports[3].ReportID, ShouldResemble, ReportID{"proj-foo", "some.file"})
			})

			Convey(`UpdateFrom`, func() {
				So(rd.UpdateFrom(rd.Clone()), ShouldEqual, 4)
				So(rd.data, ShouldHaveLength, 3)
				So(rd.data[ReportID{"proj-foo", ""}], ShouldHaveLength, 4)
				So(rd.data[ReportID{"proj-foo", ""}][0].Tag, ShouldResemble, "TAG")
				So(rd.data[ReportID{"proj-foo", ""}][1].Tag, ShouldResemble, "TAG_OTHER")
				So(rd.data[ReportID{"proj-foo", ""}][2].Tag, ShouldResemble, "TAG")
				So(rd.data[ReportID{"proj-foo", ""}][3].Tag, ShouldResemble, "TAG_OTHER")
				So(rd.data[ReportID{"proj-foo", "some.file"}][0].Tag, ShouldResemble, "TAG_THIRD")
				So(rd.data[ReportID{"proj-foo", "some.file"}][1].Tag, ShouldResemble, "TAG_THIRD")
				So(rd.data[ReportID{"other-proj", ""}], ShouldHaveLength, 2)
				So(rd.data[ReportID{"other-proj", ""}][0].Tag, ShouldResemble, "TAG")
				So(rd.data[ReportID{"other-proj", ""}][1].Tag, ShouldResemble, "TAG")
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
				mkReport("proj-foo", "some.file", "TAG_THIRD", "another problem"),
				mkReport("proj-foo", "", "TAG", "problem", "meta:data", "a:value"),
				mkReport("proj-foo", "", "TAG_OTHER", "problem"),
				mkReport("other-proj", "", "TAG", "problem"),
				mkReport("a-third-prog", "", "TAG", "problem"),
			)

			buf := &bytes.Buffer{}
			So(rd.WriteToCSV(buf), ShouldBeNil)

			csvRead := csv.NewReader(buf)
			csvRead.FieldsPerRecord = -1 // variable
			lines, err := csvRead.ReadAll()
			So(err, ShouldBeNil)
			So(lines, ShouldResemble, [][]string{
				csvHeader,
				{"a-third-prog", "", "TAG", "problem", "false"},
				{"other-proj", "", "TAG", "problem", "false"},
				{"proj-foo", "", "TAG", "problem", "false", "a:value", "meta:data"},
				{"proj-foo", "", "TAG_OTHER", "problem", "false"},
				{"proj-foo", "some.file", "TAG_THIRD", "another problem", "false"},
			})
		})

		Convey(`Read`, func() {
			Convey(`OK`, func() {
				buf := &bytes.Buffer{}
				csvWrite := csv.NewWriter(buf)
				csvWrite.WriteAll([][]string{
					csvHeader,
					{"a-third-prog", "", "TAG", "problem"},
					{"other-proj", "", "TAG", "problem"},
					{"proj-foo", "", "TAG", "problem", "a:value", "meta:data"},
					{"proj-foo", "", "TAG_OTHER", "problem"},
					{"proj-foo", "some.file", "TAG_THIRD", "another problem"},
				})
				csvWrite.Flush()

				rd, err := NewReportDumpFromCSV(buf)
				So(err, ShouldBeNil)
				So(rd.data, ShouldResemble, map[ReportID][]*Report{
					{"proj-foo", ""}: {
						mkReport("proj-foo", "", "TAG", "problem", "meta:data", "a:value"),
						mkReport("proj-foo", "", "TAG_OTHER", "problem"),
					},
					{"proj-foo", "some.file"}: {
						mkReport("proj-foo", "some.file", "TAG_THIRD", "another problem"),
					},
					{"other-proj", ""}: {
						mkReport("other-proj", "", "TAG", "problem"),
					},
					{"a-third-prog", ""}: {
						mkReport("a-third-prog", "", "TAG", "problem"),
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
				header = append(header, "{schema=v3}")
				csvWrite.Write(header)
				csvWrite.Flush()

				_, err := NewReportDumpFromCSV(buf)
				So(err, ShouldErrLike, "unexpected version: \"v3\", expected \"v2\"")
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
