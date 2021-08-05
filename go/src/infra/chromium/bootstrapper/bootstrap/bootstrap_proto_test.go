// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"errors"
	"testing"

	. "infra/chromium/bootstrapper/util"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestMultierror(t *testing.T) {
	t.Parallel()

	Convey("multierror", t, func() {

		Convey("reports a contained error", func() {
			m := &multierror{[]error{
				errors.New("foo error"),
			}}

			So(m, ShouldErrLike,
				"1 error occurred",
				"foo error")
		})

		Convey("reports all contained errors", func() {
			m := &multierror{[]error{
				errors.New("foo error"),
				errors.New("bar error"),
				errors.New("baz error"),
			}}

			So(m, ShouldErrLike,
				"3 errors occurred",
				"foo error",
				"bar error",
				"baz error")
		})

	})
}

type fakeValidatable struct {
	fn func(v *validator)
}

func (f *fakeValidatable) validate(v *validator) {
	if f.fn != nil {
		f.fn(v)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	Convey("validate", t, func() {

		Convey("calls validate on the validatable", func() {
			called := false
			x := &fakeValidatable{func(v *validator) {
				called = true
			}}

			err := validate(x, "$test")

			So(err, ShouldBeNil)
			So(called, ShouldBeTrue)
		})

		Convey("returns error if validator.errorf is called", func() {

			Convey("with ${} in format string replaced with validation context", func() {
				x := &fakeValidatable{func(v *validator) {
					v.errorf("failure to validate ${}")
				}}

				err := validate(x, "$test")

				So(err, ShouldErrLike, "failure to validate $test")

			})

			Convey("with ${} in format arguments not replace with validation context", func() {
				x := &fakeValidatable{func(v *validator) {
					v.errorf("failure to validate %s", "${}")
				}}

				err := validate(x, "$test")

				So(err, ShouldErrLike, "failure to validate ${}")
			})

			Convey("with ${} in format string replaced with updated validation context in nested validate call", func() {
				x := &fakeValidatable{func(v *validator) {
					v.errorf("failure to validate ${}")
				}}
				y := &fakeValidatable{func(v *validator) {
					v.validate(x, "x")
				}}

				err := validate(y, "$test")

				So(err, ShouldErrLike, "failure to validate $test.x")
			})

		})

	})
}

func createBootstrapProperties(propsJson []byte) *BootstrapProperties {
	props := &BootstrapProperties{}
	PanicOnError(protojson.Unmarshal(propsJson, props))
	return props
}

func TestBootstrapProtoValidation(t *testing.T) {
	t.Parallel()

	Convey("validate", t, func() {

		Convey("fails for unset required top-level fields", func() {
			props := createBootstrapProperties([]byte("{}"))

			err := validate(props, "$test")

			So(err, ShouldErrLike,
				"none of the config_project fields in $test is set",
				"$test.properties_file is not set",
				"$test.exe is not set")
		})

		Convey("fails for unset required fields in exe", func() {
			props := createBootstrapProperties([]byte(`{
				"exe": {}
			}`))

			err := validate(props, "$test")

			So(err, ShouldErrLike,
				"$test.exe.cipd_package is not set",
				"$test.exe.cipd_version is not set",
				"$test.exe.cmd is not set")
		})

		Convey("with a top level project", func() {

			Convey("fails for unset required fields in top_level_project", func() {
				props := createBootstrapProperties([]byte(`{
					"top_level_project": {}
				}`))

				err := validate(props, "$test")

				So(err, ShouldErrLike,
					"$test.top_level_project.repo is not set",
					"$test.top_level_project.ref is not set")
			})

			Convey("fails for unset required fields in top_level_project.repo", func() {
				props := createBootstrapProperties([]byte(`{
						"top_level_project": {
							"repo": {}
						}
					}`))

				err := validate(props, "$test")

				So(err, ShouldErrLike,
					"$test.top_level_project.repo.host is not set",
					"$test.top_level_project.repo.project is not set")
			})

			Convey("succeeds for valid properties", func() {
				props := createBootstrapProperties([]byte(`{
						"top_level_project": {
							"repo": {
								"host": "chromium.googlesource.com",
								"project": "top/level"
							},
							"ref": "refs/heads/top-level"
						},
						"properties_file": "infra/config/bucket/builder/properties.textpb",
						"exe": {
							"cipd_package": "fake-package",
							"cipd_version": "fake-version",
							"cmd": ["fake-cmd"]
						}
					}`))

				err := validate(props, "$test")

				So(err, ShouldBeNil)
			})
		})

		Convey("with a dependency project", func() {

			Convey("fails for unset required fields in dependency_project", func() {
				props := createBootstrapProperties([]byte(`{
						"dependency_project": {}
					}`))

				err := validate(props, "$test")

				So(err, ShouldErrLike,
					"$test.dependency_project.top_level_repo is not set",
					"$test.dependency_project.top_level_ref is not set",
					"$test.dependency_project.config_repo is not set",
					"$test.dependency_project.config_repo_path is not set")
			})

			Convey("fails for unset required fields in dependency_project.top_level_repo", func() {
				props := createBootstrapProperties([]byte(`{
						"dependency_project": {
							"top_level_repo": {}
						}
					}`))

				err := validate(props, "$test")

				So(err, ShouldErrLike,
					"$test.dependency_project.top_level_repo.host is not set",
					"$test.dependency_project.top_level_repo.project is not set")
			})

			Convey("fails for unset required fields in dependency_project.config_repo", func() {
				props := createBootstrapProperties([]byte(`{
						"dependency_project": {
							"config_repo": {}
						}
					}`))

				err := validate(props, "$test")

				So(err, ShouldErrLike,
					"$test.dependency_project.config_repo.host is not set",
					"$test.dependency_project.config_repo.project is not set")
			})

			Convey("succeeds for valid properties", func() {
				props := createBootstrapProperties([]byte(`{
						"dependency_project": {
							"top_level_repo": {
								"host": "chromium.googlesource.com",
								"project": "top/level"
							},
							"top_level_ref": "refs/heads/top-level",
							"config_repo": {
								"host": "chromium.googlesource.com",
								"project": "dependency"
							},
							"config_repo_path": "path/to/dependency"
						},
						"properties_file": "infra/config/generated/builders/bucket/builder/properties.textpb",
						"exe": {
							"cipd_package": "fake-package",
							"cipd_version": "fake-version",
							"cmd": ["fake-cmd"]
						}
					}`))

				err := validate(props, "$test")

				So(err, ShouldBeNil)
			})

		})

	})
}
