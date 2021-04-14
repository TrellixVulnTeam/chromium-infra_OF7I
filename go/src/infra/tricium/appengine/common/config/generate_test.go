// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	admin "infra/tricium/api/admin/v1"
	tricium "infra/tricium/api/v1"
)

const (
	platform = tricium.Platform_UBUNTU
)

func TestGenerate(t *testing.T) {
	Convey("Test Environment", t, func() {
		sc := &tricium.ServiceConfig{
			BuildbucketServerHost: "cr-buildbucket-dev.appspot.com",
			Platforms: []*tricium.Platform_Details{
				{
					Name:       platform,
					Dimensions: []string{"pool:Chrome", "os:Ubuntu13.04"},
					HasRuntime: true,
				},
			},
			DataDetails: []*tricium.Data_TypeDetails{
				{
					Type:               tricium.Data_GIT_FILE_DETAILS,
					IsPlatformSpecific: false,
				},
				{
					Type:               tricium.Data_RESULTS,
					IsPlatformSpecific: true,
				},
			},
		}
		wrapperAnalyzer := "Wrapper"
		pc := &tricium.ProjectConfig{
			Functions: []*tricium.Function{
				{
					Type:     tricium.Function_ANALYZER,
					Name:     wrapperAnalyzer,
					Needs:    tricium.Data_GIT_FILE_DETAILS,
					Provides: tricium.Data_RESULTS,
					Impls: []*tricium.Impl{
						{
							ProvidesForPlatform: platform,
							RuntimePlatform:     platform,
							Impl: &tricium.Impl_Recipe{
								Recipe: &tricium.Recipe{
									Project: "infra",
									Bucket:  "try",
									Builder: "analysis",
								},
							},
						},
					},
				},
			},
			Selections: []*tricium.Selection{
				{
					Function: wrapperAnalyzer,
					Platform: platform,
				},
			},
		}

		Convey("Correct selection generates workflow", func() {
			wf, err := Generate(sc, pc, []*tricium.Data_File{}, "refs/1234/2", "https://chromium-review.googlesource.com/infra")
			So(err, ShouldBeNil)
			So(len(wf.Workers), ShouldEqual, 1)
		})
	})
}

func TestIncludeFunction(t *testing.T) {
	Convey("No paths means function is included", t, func() {
		ok, err := includeFunction(&tricium.Function{
			Type:        tricium.Function_ANALYZER,
			PathFilters: []string{"*.cc", "*.cpp"},
		}, nil)
		So(err, ShouldBeNil)
		So(ok, ShouldBeTrue)
	})

	Convey("No path filters means function is included", t, func() {
		ok, err := includeFunction(&tricium.Function{
			Type: tricium.Function_ANALYZER,
		}, []*tricium.Data_File{
			{Path: "README.md"},
			{Path: "path/foo.cc"},
		})
		So(err, ShouldBeNil)
		So(ok, ShouldBeTrue)
	})

	Convey("Analyzer is included when any path matches filter", t, func() {
		ok, err := includeFunction(&tricium.Function{
			Type:        tricium.Function_ANALYZER,
			PathFilters: []string{"*.cc", "*.cpp"},
		}, []*tricium.Data_File{
			{Path: "README.md"},
			{Path: "path/foo.cc"},
		})
		So(err, ShouldBeNil)
		So(ok, ShouldBeTrue)
	})

	Convey("Analyzer function is not included when there is no match", t, func() {
		ok, err := includeFunction(&tricium.Function{
			Type:        tricium.Function_ANALYZER,
			PathFilters: []string{"*.cc", "*.cpp"},
		}, []*tricium.Data_File{
			{Path: "whitespace.txt"},
		})
		So(err, ShouldBeNil)
		So(ok, ShouldBeFalse)
	})
}

func TestCreateWorker(t *testing.T) {
	Convey("Test Environment", t, func() {
		analyzer := "wrapper"
		gitRef := "refs/1234/2"
		gitURL := "https://chromium-review.googlesource.com/infra"
		selection := &tricium.Selection{
			Function: analyzer,
			Platform: platform,
		}
		dimension := "pool:Default"
		sc := &tricium.ServiceConfig{
			Platforms: []*tricium.Platform_Details{
				{
					Name:       platform,
					Dimensions: []string{dimension},
				},
			},
		}

		Convey("Correctly creates recipe-based worker", func() {
			f := &tricium.Function{
				Name:     analyzer,
				Needs:    tricium.Data_GIT_FILE_DETAILS,
				Provides: tricium.Data_RESULTS,
				Impls: []*tricium.Impl{
					{
						ProvidesForPlatform: platform,
						RuntimePlatform:     platform,
						Impl: &tricium.Impl_Recipe{
							Recipe: &tricium.Recipe{
								Project: "chromium",
								Bucket:  "try",
								Builder: "analysis",
							},
						},
					},
				},
			}
			w, err := createWorker(selection, sc, f, gitRef, gitURL)
			So(err, ShouldBeNil)
			So(w.Name, ShouldEqual, fmt.Sprintf("%s_%s", analyzer, platform))
			So(w.Needs, ShouldEqual, f.Needs)
			So(w.Provides, ShouldEqual, f.Provides)
			So(w.ProvidesForPlatform, ShouldEqual, platform)
			wi := w.Impl.(*admin.Worker_Recipe)
			if wi == nil {
				fail("Incorrect worker type")
			}
			So(wi.Recipe.Project, ShouldEqual, "chromium")
		})
	})
}

func fail(str string) {
	So(nil, func(a interface{}, b ...interface{}) string { return str }, nil)
}
