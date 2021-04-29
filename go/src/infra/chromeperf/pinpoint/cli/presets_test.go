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
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/data/text"
)

func TestLoadPresets(t *testing.T) {
	t.Parallel()

	Convey("Given a simple presets file", t, func() {
		sp, err := os.Open("testdata/simple-presets.yaml")
		So(err, ShouldBeNil)
		defer sp.Close()
		Convey("When we load the presets", func() {
			pd, err := loadPresets(sp)
			So(err, ShouldBeNil)
			Convey("Then we can find the \"basic\" preset", func() {
				_, err := pd.GetPreset("basic")
				So(err, ShouldBeNil)
			})
			Convey("And we can find the \"complex\" preset", func() {
				_, err := pd.GetPreset("complex")
				So(err, ShouldBeNil)
			})

			Convey("And the \"basic\" and \"complex\" presets only differ with extra args", func() {
				b, err := pd.GetPreset("basic")
				So(err, ShouldBeNil)
				c, err := pd.GetPreset("complex")
				So(err, ShouldBeNil)
				So(b.TelemetryExperiment.ExtraArgs, ShouldNotEqual, c.TelemetryExperiment.ExtraArgs)
				So(b.TelemetryExperiment.Benchmark, ShouldEqual, c.TelemetryExperiment.Benchmark)
			})

		})
	})

	Convey("Given an invalid presets file", t, func() {
		sp, err := os.Open("testdata/invalid-presets.yaml")
		So(err, ShouldBeNil)
		defer sp.Close()
		Convey("When we load the presets", func() {
			pd, err := loadPresets(sp)
			So(err, ShouldBeNil)
			Convey("Then looking up a preset with invalid story selection fails", func() {
				_, err := pd.GetPreset("conflicting-story-selection")
				expected := text.Doc(`
					telemetry experiments must only have exactly one of story
					or story_tags in story_selection
				`)
				So(err, ShouldBeError, expected)
				_, err = pd.GetPreset("empty-story-selection")
				So(err, ShouldBeError, expected)
			})
			Convey("And looking up a preset with no config fails", func() {
				_, err := pd.GetPreset("empty-config")
				So(err, ShouldBeError, "telemetry experiments must have a non-empty config")
			})
		})
	})

}
