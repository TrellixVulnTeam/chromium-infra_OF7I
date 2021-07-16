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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIsolatedServiceHandling(t *testing.T) {
	Convey("Parse isolated url to object without namespace", t, func() {
		obj, err := fromIsolatedURL("https://chrome-isolated.appspot.com/browse?digest=0d054ad7c14d3444bb38db0f60164978b089f221")
		So(err, ShouldBeNil)
		So(obj.host, ShouldEqual, "https://chrome-isolated.appspot.com")
		So(obj.namespace, ShouldEqual, "default-gzip")
		So(obj.digest, ShouldEqual, "0d054ad7c14d3444bb38db0f60164978b089f221")
	})
	Convey("Parse isolated url to object with namespace", t, func() {
		obj, err := fromIsolatedURL("https://chrome-isolated.appspot.com/browse?digest=0d054ad7c14d3444bb38db0f60164978b089f221&namespace=some-namespace")
		So(err, ShouldBeNil)
		So(obj.host, ShouldEqual, "https://chrome-isolated.appspot.com")
		So(obj.namespace, ShouldEqual, "some-namespace")
		So(obj.digest, ShouldEqual, "0d054ad7c14d3444bb38db0f60164978b089f221")
	})
}

func TestCasHandling(t *testing.T) {
	Convey("Parse components from CAS URL", t, func() {
		instance, hash, bytes, err := extractCasParamsFromURL("https://cas-viewer.appspot.com/projects/chrome-swarming/instances/default_instance/blobs/327d759be13ebe68392ab8deec4fba29243b96eea2cdc10a2a3b7eac44088123/176/tree")
		So(err, ShouldBeNil)
		So(string(instance), ShouldEqual, "projects/chrome-swarming/instances/default_instance")
		So(string(hash), ShouldEqual, "327d759be13ebe68392ab8deec4fba29243b96eea2cdc10a2a3b7eac44088123")
		So(int64(bytes), ShouldEqual, 176)
	})

	Convey("No URLs is not CAS", t, func() {
		So(isCas(make(abExperimentURLs)), ShouldBeFalse)
	})
	Convey("One CAS Url is CAS", t, func() {
		urls := make(abExperimentURLs)
		urls["01"] = "https://cas-viewer.appspot.com/projects/chrome-swarming/instances/default_instance/blobs/d00a400ac4bae7b1d59be2828724972c26aa1d55801a578dbfdab235428ca5da/176/tree"
		So(isCas(urls), ShouldBeTrue)
	})
	Convey("One Isolate Url is not CAS", t, func() {
		urls := make(abExperimentURLs)
		urls["01"] = "https://chrome-isolated.appspot.com/browse?digest=0d054ad7c14d3444bb38db0f60164978b089f221&namespace=some-namespace"
		So(isCas(urls), ShouldBeFalse)
	})
}
