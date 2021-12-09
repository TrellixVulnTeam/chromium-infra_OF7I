// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	tricium "infra/tricium/api/v1"
)

func TestPylintParsing(t *testing.T) {

	Convey("parsePylintOutput", t, func() {

		Convey("Parsing empty buffer gives no warnings", func() {
			comments, err := parsePylintOutput([]byte("[]"))
			if err != nil {
				t.Fatal(err)
			}
			So(comments, ShouldBeEmpty)
		})

		Convey("Parsing normal pylint output generates the appropriate comments", func() {
			output := `
				[
					{
						"type": "convention",
						"line": 6,
						"column": 0,
						"path": "test.py",
						"symbol": "empty-docstring",
						"message": "Empty function docstring"
					},
					{
						"type": "warning",
						"line": 6,
						"column": 15,
						"path": "test.py",
						"symbol": "unused-argument",
						"message": "Unused argument 'y'"
					},
					{
						"type": "warning",
						"line": 6,
						"column": 18,
						"path": "test.py",
						"symbol": "unused-argument",
						"message": "Unused argument 'z'"
					},
					{
						"type": "warning",
						"line": 12,
						"column": 2,
						"path": "test.py",
						"symbol": "unnecessary-pass",
						"message": "Unnecessary pass statement"
					},
					{
						"type": "warning",
						"line": 19,
						"column": 10,
						"path": "test.py",
						"symbol": "undefined-loop-variable",
						"message": "Using possibly undefined loop variable 'a'"
					},
					{
						"type": "warning",
						"line": 18,
						"column": 6,
						"path": "test.py",
						"symbol": "unused-variable",
						"message": "Unused variable 'a'"
					},
					{
						"type": "error",
						"line": 26,
						"column": 0,
						"path": "test.py",
						"symbol": "undefined-variable",
						"message": "Undefined variable 'main'"
					}
				]
			`

			expected := []*tricium.Data_Comment{
				{
					Path: "test.py",
					Message: "Empty function docstring.\n" +
						"To disable, add: # pylint: disable=empty-docstring",
					Category:  "Pylint/convention/empty-docstring",
					StartLine: 6,
					StartChar: 0,
				},
				{
					Path: "test.py",
					Message: "Unused argument 'y'.\n" +
						"To disable, add: # pylint: disable=unused-argument",
					Category:  "Pylint/warning/unused-argument",
					StartLine: 6,
					StartChar: 15,
				},
				{
					Path: "test.py",
					Message: "Unused argument 'z'.\n" +
						"To disable, add: # pylint: disable=unused-argument",
					Category:  "Pylint/warning/unused-argument",
					StartLine: 6,
					StartChar: 18,
				},
				{
					Path: "test.py",
					Message: "Unnecessary pass statement.\n" +
						"To disable, add: # pylint: disable=unnecessary-pass",
					Category:  "Pylint/warning/unnecessary-pass",
					StartLine: 12,
					StartChar: 2,
				},
				{
					Path: "test.py",
					Message: "Using possibly undefined loop variable 'a'.\n" +
						"To disable, add: # pylint: disable=undefined-loop-variable",
					Category:  "Pylint/warning/undefined-loop-variable",
					StartLine: 19,
					StartChar: 10,
				},
				{
					Path: "test.py",
					Message: "Unused variable 'a'.\n" +
						"To disable, add: # pylint: disable=unused-variable",
					Category:  "Pylint/warning/unused-variable",
					StartLine: 18,
					StartChar: 6,
				},
				{
					Path: "test.py",
					Message: "Undefined variable 'main'.\n" +
						"This check could give false positives when there are wildcard imports\n" +
						"(from module import *). It is recommended to avoid wildcard imports; see\n" +
						"https://www.python.org/dev/peps/pep-0008/#imports.\n" +
						"To disable, add: # pylint: disable=undefined-variable",
					Category:  "Pylint/error/undefined-variable",
					StartLine: 26,
					StartChar: 0,
				},
			}

			comments, err := parsePylintOutput([]byte(output))
			if err != nil {
				t.Fatal(err)
			}
			So(comments, ShouldResemble, expected)
		})
	})
}
