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
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFactorySettingFallbackCasese(t *testing.T) {
	Convey("Given a custom PINPOINT_CACHE_DIR", t, func() {
		ctx := context.Background()
		td, err := ioutil.TempDir(os.TempDir(), "pinpoint-test-*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(td)
		os.Setenv("PINPOINT_CACHE_DIR", td)
		defer os.Unsetenv("PINPOINT_CACHE_DIR")
		tc, err := newTokenCache(ctx, td)
		So(err, ShouldBeNil)
		So(tc, ShouldNotBeNil)
		tc, creds, err := getFactorySettings(ctx, "pinpoint-stable.endpoints.chromeperf.cloud.goog")
		So(err, ShouldBeNil)
		So(tc, ShouldNotBeNil)
		So(err, ShouldBeNil)
		So(tc.cacheFile, ShouldEqual, filepath.Join(td, "cached-token"))
		So(creds, ShouldNotBeNil)
	})
	Convey("Valid domain provided", t, func() {
		ctx := context.Background()
		td, err := ioutil.TempDir(os.TempDir(), "pinpoint-test-*")
		So(err, ShouldBeNil)
		defer os.RemoveAll(td)
		os.Setenv("PINPOINT_CACHE_DIR", td)
		defer os.Unsetenv("PINPOINT_CACHE_DIR")
		tc, err := newTokenCache(ctx, td)
		So(err, ShouldBeNil)
		So(tc, ShouldNotBeNil)
		tc, creds, err := getFactorySettings(ctx, "undefined")
		So(err, ShouldBeNil)
		So(tc, ShouldNotBeNil)
		So(err, ShouldBeNil)
		So(tc.cacheFile, ShouldEqual, filepath.Join(td, "cached-token"))
		So(creds, ShouldNotBeNil)
	})
}
