// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCreatePlaceholderProject(t *testing.T) {

	Convey("Given a project creator", t, func() {
		Convey("When using the default values", func() {
			projectConfig := CreatePlaceholderProjectConfig()

			So(projectConfig.ProjectMetadata.DisplayName, ShouldEqual, "Chromium")
			So(projectConfig.Monorail.Project, ShouldEqual, "chromium")
			So(projectConfig.Realms[0].TestVariantAnalysis.BqExports[0].Table.Dataset, ShouldEqual, "chromium")
		})

		Convey("When using a key", func() {
			chromeOsKey := "chromeos"

			projectConfig := CreatePlaceholderProjectConfigWithKey(chromeOsKey)

			So(projectConfig.ProjectMetadata.DisplayName, ShouldEqual, strings.Title(chromeOsKey))
			So(projectConfig.Monorail.Project, ShouldEqual, chromeOsKey)
			So(projectConfig.Realms[0].TestVariantAnalysis.BqExports[0].Table.Dataset, ShouldEqual, chromeOsKey)
		})
	})

}
