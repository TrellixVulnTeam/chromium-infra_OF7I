// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package module

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestModule(t *testing.T) {
	t.Parallel()

	Convey("Extracts name", t, func() {
		m, err := parseYAML([]byte(`service: zzz`))
		So(err, ShouldBeNil)
		So(m.Name, ShouldEqual, "zzz")

		m, err = parseYAML([]byte(`module: zzz`))
		So(err, ShouldBeNil)
		So(m.Name, ShouldEqual, "zzz")

		m, err = parseYAML([]byte(`stuff: blah`))
		So(err, ShouldBeNil)
		So(m.Name, ShouldEqual, "default")

		m, err = parseYAML([]byte(`module: 123`))
		So(err, ShouldErrLike, "not a string")
	})

	Convey("Process drops unsupported fields", t, func() {
		m, err := parseYAML([]byte(`{
			"application": "app",
			"version": "ver",
			"module": "zzz",
			"luci_gae_vars": {"app-id": {"ZZZ": 123}}
		}`))
		So(err, ShouldBeNil)
		_, err = m.Process("app-id", nil)
		So(err, ShouldBeNil)

		blob, err := m.DumpYAML()
		So(err, ShouldBeNil)
		So(string(blob), ShouldEqual, "service: zzz\n")
	})

	Convey("Process rejects bad luci_gae_vars section", t, func() {
		cases := []string{
			`luci_gae_vars: 123`,
			`luci_gae_vars: {123: {}}`,
			`luci_gae_vars: {"zzz": 123}`,
			`luci_gae_vars: {"zzz": {123: "zzz"}}`,
		}
		for _, body := range cases {
			body := body
			Convey(fmt.Sprintf("Case %q", body), func() {
				m, err := parseYAML([]byte(body))
				So(err, ShouldBeNil)
				_, err = m.Process("app-id", nil)
				So(err, ShouldErrLike, "should")
			})
		}
	})

	Convey("Render vars", t, func() {
		Convey("Unsupported type in decl", func() {
			decl := varsDecl{
				"app-id": {"VAR": 123.123},
			}
			_, _, err := renderVars(nil, "app-id", decl, nil)
			So(err, ShouldErrLike, `variable "VAR" has unsupported type float64`)
		})

		Convey("Inconsistent type in decl", func() {
			decl := varsDecl{
				"app-id-1": {"VAR": "blah-1"},
				"app-id-2": {"VAR": 123},
			}
			_, _, err := renderVars(nil, "app-id", decl, nil)
			So(err, ShouldErrLike, `variable "VAR" has ambiguous type`)
		})

		Convey("Bad int var", func() {
			decl := varsDecl{
				"app-id": {"VAR": 0},
			}
			_, _, err := renderVars(nil, "app-id", decl, map[string]string{"VAR": "haha"})
			So(err, ShouldErrLike, `the value of variable "VAR" is expected to be an integer, got "haha"`)
		})

		Convey("Substitutes", func() {
			decl := varsDecl{
				"app-id": {
					"STR_VAR": "blah-1",
					"INT_VAR": 123,
					"UNUSED":  "!!!",
				},
				"another-app-id": {
					"STR_VAR": "blah-2",
					"INT_VAR": 456,
				},
			}
			v := map[string]interface{}{
				"str_key1": "blah ${STR_VAR}",
				"str_key2": "blah ${INT_VAR}",
				"int_var":  "${INT_VAR}",
				"str_var":  "${STR_VAR}",
				"a bunch":  "${ANOTHER_VAR} ${STR_VAR}",
			}
			out, consumed, err := renderVars(v, "app-id", decl, map[string]string{
				"INT_VAR":     "42", // will be converted to int
				"ANOTHER_VAR": "zzz",
				"UNUSED_TOO":  "!!!",
			})
			So(err, ShouldBeNil)
			So(out, ShouldResemble, map[string]interface{}{
				"str_key1": "blah blah-1",
				"str_key2": "blah 42",
				"int_var":  42, // yay, int
				"str_var":  "blah-1",
				"a bunch":  "zzz blah-1",
			})
			So(consumed.ToSortedSlice(), ShouldResemble, []string{
				"ANOTHER_VAR",
				"INT_VAR",
				"STR_VAR",
			})
		})

		Convey("Recurses", func() {
			v := map[string]interface{}{
				"top": "${VAR}",
				"dict": map[interface{}]interface{}{
					"deeper": map[interface{}]interface{}{
						"key": "${VAR}",
					},
					123: "huh",
				},
				"list": []interface{}{
					"${VAR}",
					[]interface{}{"${VAR}"},
				},
				"null": nil,
			}
			out, _, err := renderVars(v, "app-id", nil, map[string]string{"VAR": "zzz"})
			So(err, ShouldBeNil)
			So(out, ShouldResemble, map[string]interface{}{
				"top": "zzz",
				"dict": map[interface{}]interface{}{
					"deeper": map[interface{}]interface{}{
						"key": "zzz",
					},
					123: "huh",
				},
				"list": []interface{}{
					"zzz",
					[]interface{}{"zzz"},
				},
				"null": nil,
			})
		})

		Convey("Undefined key", func() {
			_, _, err := renderVars(map[string]interface{}{"top": "${VAR} ${ANOTHER}"}, "app-id", nil, nil)
			So(err, ShouldErrLike, `a value for variable "VAR" is not provided`)
		})
	})
}
