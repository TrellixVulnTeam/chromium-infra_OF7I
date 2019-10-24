// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	tricium "infra/tricium/api/v1"
)

func analyzeJSONTestFile(t *testing.T, name string) []*tricium.Data_Comment {
	// Mock current time for testing
	filePath := "testdata/src/" + name
	f := openFileOrDie(filePath)
	defer closeFileOrDie(f)
	return analyzeFieldTrialTestingConfig(f, filePath)
}

func TestConfigCheck(t *testing.T) {
	Convey("Analyze Config JSON file with no errors: one experiment", t, func() {
		results := analyzeJSONTestFile(t, "configs/one_experiment.json")
		So(results, ShouldBeNil)
	})

	Convey("Analyze Config JSON file with no errors: many configs one experiment", t, func() {
		results := analyzeJSONTestFile(t, "configs/many_configs_one_exp.json")
		So(results, ShouldBeNil)
	})

	Convey("Analyze Config JSON file with warning: many experiments", t, func() {
		results := analyzeJSONTestFile(t, "configs/many_experiments.json")
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Experiments",
				Message:   fmt.Sprintf(manyExperimentsWarning, "TestConfig1"),
				StartLine: 7,
				Path:      "testdata/src/configs/many_experiments.json",
			},
		})
	})

	Convey("Analyze Config JSON file with two warnings: many configs many experiments", t, func() {
		results := analyzeJSONTestFile(t, "configs/many_configs_many_exp.json")
		So(results, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  category + "/Experiments",
				Message:   fmt.Sprintf(manyExperimentsWarning, "TestConfig1"),
				StartLine: 7,
				Path:      "testdata/src/configs/many_configs_many_exp.json",
			},
			{
				Category:  category + "/Experiments",
				Message:   fmt.Sprintf(manyExperimentsWarning, "TestConfig1"),
				StartLine: 26,
				Path:      "testdata/src/configs/many_configs_many_exp.json",
			},
		})
	})
}
