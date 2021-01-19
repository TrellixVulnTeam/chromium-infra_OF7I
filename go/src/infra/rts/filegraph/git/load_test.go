// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package git

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGraphCache(t *testing.T) {
	t.Parallel()

	Convey(`GraphCache`, t, func() {
		ctx := context.Background()
		ctx = memlogger.Use(ctx)

		Convey(`empty file is cache-miss`, func() {
			tmpd, err := ioutil.TempDir("", "filegraph_git")
			So(err, ShouldBeNil)
			defer os.RemoveAll(tmpd)

			var cache graphCache
			cache.File, err = os.Create(filepath.Join(tmpd, "empty"))
			So(err, ShouldBeNil)
			defer cache.Close()

			_, err = cache.tryReading(ctx)
			So(err, ShouldBeNil)

			log := logging.Get(ctx).(*memlogger.MemLogger)
			So(log, memlogger.ShouldHaveLog, logging.Info, "populating cache")
		})
	})
}
