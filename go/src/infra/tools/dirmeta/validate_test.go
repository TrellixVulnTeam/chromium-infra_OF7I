// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmeta

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestValidateFile(t *testing.T) {
	t.Parallel()

	Convey(`ValidateFile`, t, func() {
		suite := func(path string, valid bool) {
			dir, err := os.Open(path)
			So(err, ShouldBeNil)
			defer dir.Close()

			names, err := dir.Readdirnames(1000)
			So(err, ShouldBeNil)
			for _, name := range names {
				fullName := filepath.Join(path, name)
				Convey(fullName, func() {
					err := ValidateFile(fullName)
					if valid {
						So(err, ShouldBeNil)
					} else {
						So(err, ShouldNotBeNil)
					}
				})
			}
		}

		suite("testdata/validation/valid", true)
		suite("testdata/validation/invalid", false)
	})
}
