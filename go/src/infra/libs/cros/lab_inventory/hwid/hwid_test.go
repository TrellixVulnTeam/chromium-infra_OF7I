// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hwid

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
)

var (
	hwidSku     string = "blacktip_IntelR_CeleronR_CPU_N3350_1_10GHz_4GB"
	hwidVariant string = "ampton"
)
var dutlabelJSON string = `
{
 "labels": [
  {
   "name": "sku",
   "value": "` + hwidSku + `"
  },
  {
   "name": "variant",
   "value": "` + hwidVariant + `"
  },
  {
   "name": "phase",
   "value": "PVT"
  },
  {
   "name": "touchscreen"
  }
 ],
 "possible_labels": [
  "sku",
  "phase",
  "touchscreen"
 ]
}
`

func TestGetHwidData(t *testing.T) {
	ctx := gaetesting.TestingContextWithAppID("go-test")

	Convey("Get hwid data", t, func() {
		Convey("Happy path", func() {
			mockHwidServerForDutLabel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, dutlabelJSON)
			}))
			defer mockHwidServerForDutLabel.Close()
			hwidServerURL = mockHwidServerForDutLabel.URL + "/%s/%s/%s"

			data, err := GetHwidData(ctx, "AMPTON C3B-A2B-D2K-H9I-A2S", "secret")
			So(err, ShouldBeNil)
			So(data.Sku, ShouldEqual, hwidSku)
			So(data.Variant, ShouldEqual, hwidVariant)
		})

		Convey("Invaid key", func() {
			mockHwidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "bad key")
			}))
			defer mockHwidServer.Close()
			hwidServerURL = mockHwidServer.URL + "/%s/%s/%s"

			_, err := GetHwidData(ctx, "AMPTON C3B-A2B-D2K-H9I-A2S", "secret")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "bad key")
		})
	})
}
