// Copyright 2020 The LUCI Authors.
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

package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

// Test if correct path has been passed for CloudKMS Key.
func TestPathValidation(t *testing.T) {
	Convey(`Path too short.`, t, func() {
		err := validateCryptoKeysKMSPath("bad/path")
		So(err, ShouldErrLike, "path should have the form")
	})
	Convey(`Path too long.`, t, func() {
		err := validateCryptoKeysKMSPath("bad/long/long/long/long/long/long/long/long/long/path")
		So(err, ShouldErrLike, "path should have the form")
	})
	Convey(`Path misspelling.`, t, func() {
		err := validateCryptoKeysKMSPath("projects/chromium/oops/global/keyRings/test/cryptoKeys/my_key")
		So(err, ShouldErrLike, "expected component 3")
	})
	Convey(`Good path.`, t, func() {
		err := validateCryptoKeysKMSPath("projects/chromium/locations/global/keyRings/test/cryptoKeys/my_key")
		So(err, ShouldBeNil)
	})
	Convey(`Good path.`, t, func() {
		err := validateCryptoKeysKMSPath("projects/chromium/locations/global/keyRings/test/cryptoKeys/my_key/cryptoKeyVersions/1")
		So(err, ShouldBeNil)
	})
	Convey(`Support leading slash.`, t, func() {
		err := validateCryptoKeysKMSPath("/projects/chromium/locations/global/keyRings/test/cryptoKeys/my_key")
		So(err, ShouldBeNil)
	})
}
