// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package datastore

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/gae/service/datastore"
)

func TestGetLastScannedTime(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")

	Convey("Get last scanned time from datastore", t, func() {
		Convey("Get existing metadata entity from datastore", func() {
			lastScannedTime := time.Date(2020, 01, 01, 12, 34, 56, 0, time.UTC)
			e := &MRMetadataEntity{
				ID:          MRLastScannedID,
				LastScanned: lastScannedTime,
			}
			err := datastore.Put(ctx, e)
			So(err, ShouldBeNil)

			res, err := GetLastScannedTime(ctx)
			So(res.LastScanned, ShouldEqual, e.LastScanned)
			So(err, ShouldBeNil)

			// Clean up test
			err = datastore.Delete(ctx, e)
			So(err, ShouldBeNil)
		})
		Convey("No metadata entity exists in datastore", func() {
			_, err := GetLastScannedTime(ctx)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "datastore: no such entity")
		})
	})
}

func TestSaveLastScannedTime(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")

	Convey("Save last scanned time to datastore", t, func() {
		Convey("Save new metadata entity", func() {
			lastScannedTime := time.Date(2020, 01, 01, 12, 34, 56, 0, time.UTC)
			err := SaveLastScannedTime(ctx, lastScannedTime)
			So(err, ShouldBeNil)

			res, err := GetLastScannedTime(ctx)
			So(res.LastScanned, ShouldEqual, lastScannedTime)
			So(err, ShouldBeNil)
		})
		Convey("Update metadata entity", func() {
			oldScannedTime := time.Date(2020, 01, 01, 12, 34, 56, 0, time.UTC)
			res, err := GetLastScannedTime(ctx)
			So(res.LastScanned, ShouldEqual, oldScannedTime)
			So(err, ShouldBeNil)

			newScannedTime := time.Date(2020, 01, 01, 01, 00, 00, 0, time.UTC)
			err = SaveLastScannedTime(ctx, newScannedTime)
			So(err, ShouldBeNil)

			res, err = GetLastScannedTime(ctx)
			So(res.LastScanned, ShouldEqual, newScannedTime)
			So(err, ShouldBeNil)
		})
	})
}
