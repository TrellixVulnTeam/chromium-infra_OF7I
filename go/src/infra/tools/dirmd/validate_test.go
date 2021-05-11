// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmd

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
			err := filepath.Walk(path, func(fullName string, info os.FileInfo, err error) error {
				switch {
				case err != nil:
					return err
				case info.IsDir():
					return nil
				}

				Convey(fullName, func() {
					err := ValidateFile(fullName)
					if valid {
						So(err, ShouldBeNil)
					} else {
						So(err, ShouldNotBeNil)
					}
				})
				return nil
			})
			So(err, ShouldBeNil)
		}

		suite("testdata/validation/valid", true)
		suite("testdata/validation/invalid", false)
	})
}
