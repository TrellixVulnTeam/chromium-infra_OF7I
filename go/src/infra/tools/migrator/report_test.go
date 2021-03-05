// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

import (
	"testing"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/config"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestReportID(t *testing.T) {
	t.Parallel()

	Convey(`ReportID`, t, func() {
		Convey(`ConfigSet`, func() {
			So(ReportID{Project: "foo"}.ConfigSet(), ShouldResemble,
				config.Set("projects/foo"))
			So(ReportID{Project: "foo", ConfigFile: "irrelevant"}.ConfigSet(), ShouldResemble,
				config.Set("projects/foo"))
		})

		Convey(`String`, func() {
			So(ReportID{Project: "foo"}.String(), ShouldResemble, "foo")
			So(ReportID{Project: "foo", ConfigFile: "file"}.String(), ShouldResemble, "foo|file")
		})
	})
}

func TestReport(t *testing.T) {
	t.Parallel()

	Convey(`Report`, t, func() {
		r := &Report{
			ReportID: ReportID{"proj-foo", "config.file"},
			Tag:      "SOME_TAG",
			Problem:  "This is a problem.",
			Metadata: map[string]stringset.Set{
				"meta": stringset.NewFromSlice("value"),
			},
		}

		Convey(`Clone`, func() {
			newR := r.Clone()
			So(r.ReportID, ShouldResemble, newR.ReportID)
			So(r.Tag, ShouldResemble, newR.Tag)
			So(r.Problem, ShouldResemble, newR.Problem)
			So(r.Metadata, ShouldNotEqual, newR.Metadata)                 // different maps
			So(r.Metadata["meta"], ShouldNotEqual, newR.Metadata["meta"]) // different sets
			So(r.Metadata["meta"].ToSlice(), ShouldResemble, newR.Metadata["meta"].ToSlice())
		})

		Convey(`ToCSVRow`, func() {
			So(r.ToCSVRow(), ShouldResemble, []string{
				"proj-foo", "config.file", "SOME_TAG", "This is a problem.", "false",
				"meta:value",
			})

			r.Actionable = true
			So(r.ToCSVRow(), ShouldResemble, []string{
				"proj-foo", "config.file", "SOME_TAG", "This is a problem.", "true",
				"meta:value",
			})
		})

		Convey(`NewReportFromCSVRow`, func() {
			Convey(`Good`, func() {
				report, err := NewReportFromCSVRow([]string{
					"proj-foo", "config.file", "SOME_TAG", "This is a problem.",
					"meta:value", "meta:other_value", "other_meta:1",
				})
				So(err, ShouldBeNil)
				So(report.ReportID, ShouldResemble, ReportID{"proj-foo", "config.file"})
				So(report.Tag, ShouldResemble, "SOME_TAG")
				So(report.Problem, ShouldResemble, "This is a problem.")
				So(report.Metadata, ShouldResemble, map[string]stringset.Set{
					"meta":       stringset.NewFromSlice("value", "other_value"),
					"other_meta": stringset.NewFromSlice("1"),
				})
			})

			Convey(`Bad`, func() {
				Convey(`no Project`, func() {
					_, err := NewReportFromCSVRow(nil)
					So(err, ShouldErrLike, "Project field")

					_, err = NewReportFromCSVRow([]string{""})
					So(err, ShouldErrLike, "Project field")
				})

				Convey(`no ConfigFile`, func() {
					_, err := NewReportFromCSVRow([]string{"proj-foo"})
					So(err, ShouldErrLike, "ConfigFile field")

					_, err = NewReportFromCSVRow([]string{"proj-foo", ""})
					So(err, ShouldErrLike, "Tag field")
				})

				Convey(`no Tag`, func() {
					_, err := NewReportFromCSVRow([]string{"proj-foo", ""})
					So(err, ShouldErrLike, "Tag field")

					_, err = NewReportFromCSVRow([]string{"proj-foo", "", ""})
					So(err, ShouldErrLike, "Tag field")
				})

				Convey(`no Problem`, func() {
					_, err := NewReportFromCSVRow([]string{"proj-foo", "", "TAG"})
					So(err, ShouldErrLike, "Problem field")

					_, err = NewReportFromCSVRow([]string{"proj-foo", "", "TAG", ""})
					So(err, ShouldBeNil)
				})

				Convey(`bad metadata`, func() {
					_, err := NewReportFromCSVRow([]string{"proj-foo", "", "TAG", "", "bad"})
					So(err, ShouldErrLike, "Malformed metadata")

					_, err = NewReportFromCSVRow([]string{"proj-foo", "", "TAG", "", "ok:value"})
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
