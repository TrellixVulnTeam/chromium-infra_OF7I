// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hwid

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
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
var hwidErrorResponseJSON string = `
{
 "error": "No metadata present for the requested board: NOCTURNE TEST 3421",
 "possible_labels": [
  "sku",
  "phase",
  "touchscreen",
  "touchpad",
  "variant",
  "stylus",
  "hwid_component"
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
			hwid := "HWID 1"

			data, err := GetHwidData(ctx, hwid, "secret")
			So(err, ShouldBeNil)
			So(data.Sku, ShouldEqual, hwidSku)
			So(data.Variant, ShouldEqual, hwidVariant)

			Convey("Use cached data", func() {
				mockHwidServerForDutLabel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprintln(w, hwidErrorResponseJSON)
				}))
				defer mockHwidServerForDutLabel.Close()
				hwidServerURL = mockHwidServerForDutLabel.URL + "/%s/%s/%s"

				data, err := GetHwidData(ctx, hwid, "secret")
				So(err, ShouldBeNil)
				So(data.Sku, ShouldEqual, hwidSku)
				So(data.Variant, ShouldEqual, hwidVariant)

			})
		})
		Convey("Use stale cached data", func() {
			longTimeAgo := time.Now().UTC().Add(-72 * time.Hour)
			hwidForStaleData := "hwid for stale data"
			e2 := hwidEntity{
				ID: hwidForStaleData,
				Data: Data{
					Sku:     "stale sku",
					Variant: "stale variant",
				},
				Updated: longTimeAgo,
			}
			err := datastore.Put(ctx, &e2)
			So(err, ShouldBeNil)

			e3 := hwidEntity{ID: hwidForStaleData}
			datastore.Get(ctx, &e3)
			data, err := GetHwidData(ctx, hwidForStaleData, "secret")
			So(err, ShouldBeNil)
			So(data.Sku, ShouldEqual, "stale sku")
			So(data.Variant, ShouldEqual, "stale variant")
		})

		Convey("Error response", func() {
			mockHwidServerForDutLabel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, hwidErrorResponseJSON)
			}))
			defer mockHwidServerForDutLabel.Close()
			hwidServerURL = mockHwidServerForDutLabel.URL + "/%s/%s/%s"

			_, err := GetHwidData(ctx, "HWID 2", "secret")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "No metadata present for the requested board")
		})

		Convey("Invaid key", func() {
			mockHwidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "bad key")
			}))
			defer mockHwidServer.Close()
			hwidServerURL = mockHwidServer.URL + "/%s/%s/%s"

			_, err := GetHwidData(ctx, "HWID 3", "secret")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "bad key")
		})
	})
}
