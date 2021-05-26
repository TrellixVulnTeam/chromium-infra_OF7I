// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"math"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAnalyzeTelemetryExperiment(t *testing.T) {
	t.Parallel()
	// TODO: add tests for unhappy paths with errors
	// TODO: add more fine-grained unit tests for processing in-memory data
	// structures without requiring files
	Convey("Given a telemetry experiment manifest with known significant differences", t, func() {
		m, err := loadManifest("testdata/sample-experiment-artifacts/manifest.yaml")
		So(err, ShouldBeNil)
		Convey("When we analyze the artifacts", func() {
			rootDir, err := filepath.Abs("testdata/sample-experiment-artifacts")
			So(err, ShouldBeNil)
			r, err := analyzeExperiment(m, rootDir)
			So(err, ShouldBeNil)
			So(r, ShouldNotBeNil)
			Convey("Then we find the overall p-value", func() {
				So(r.OverallPValue, ShouldAlmostEqual, 0.0)
			})
			Convey("And we see that some p-values are less than 0.05", func() {
				less := 0
				for _, s := range r.Reports {
					if s.PValue < 0.05 {
						less += 1
					}
				}
				So(less, ShouldBeGreaterThan, 0)
			})
			Convey("And we see the stats for all metrics", func() {
				nan := 0
				withStats := 0
				for _, s := range r.Reports {
					if math.IsNaN(s.PValue) {
						nan += 1
						continue
					}
					if len(s.Measurements) > 1 {
						withStats += 1
					}
				}
				So(nan, ShouldNotEqual, 0)
				So(withStats, ShouldNotEqual, 0)
				So(nan+withStats, ShouldEqual, len(r.Reports))
			})
		})
	})

}
