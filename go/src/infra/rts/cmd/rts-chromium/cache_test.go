// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"go.chromium.org/luci/common/data/caching/lru"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCacheFile(t *testing.T) {
	t.Parallel()
	Convey(`CacheFile`, t, func() {
		ctx := context.Background()
		f := cacheFile(filepath.Join(t.TempDir(), "cache"))

		ctx = memlogger.Use(ctx)
		log := logging.Get(ctx).(*memlogger.MemLogger)

		Convey(`E2E`, func() {
			var dest []string

			So(f.Read(&dest), ShouldNotBeNil)
			So(f.TryRead(ctx, &dest), ShouldBeFalse)

			f.TryWrite(ctx, []string{"hello", "test"})
			So(f.TryRead(ctx, &dest), ShouldBeTrue)
			So(dest, ShouldResemble, []string{"hello", "test"})
		})

		Convey(`Corrupted`, func() {
			So(f.Write(ctx, "not an array"), ShouldBeNil)

			var dest []string
			So(f.TryRead(ctx, &dest), ShouldBeFalse)
			So(log, memlogger.ShouldHaveLog, logging.Warning, "failed to read cache")
		})
	})
}

func TestCache(t *testing.T) {
	t.Parallel()
	Convey(`CacheFile`, t, func() {
		ctx := context.Background()

		type entry struct {
			Value int
		}
		c := cache{
			dir:       t.TempDir(),
			memory:    lru.New(256),
			valueType: reflect.TypeOf(entry{}),
		}

		Convey(`E2E`, func() {
			called := 0
			maker := func() (interface{}, error) {
				called++
				return &entry{Value: 1}, nil
			}

			// Cache miss.
			actual, err := c.GetOrCreate(ctx, "key", maker)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &entry{Value: 1})
			So(called, ShouldEqual, 1)

			// Cache hit.
			actual, err = c.GetOrCreate(ctx, "key", maker)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &entry{Value: 1})
			So(called, ShouldEqual, 1)

			// Clear RAM and assert it loads from disk.
			c.memory.Reset()
			actual, err = c.GetOrCreate(ctx, "key", maker)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &entry{Value: 1})
			So(called, ShouldEqual, 1)
		})

		Convey(`Put`, func() {
			c.Put(ctx, "key", &entry{Value: 54})
			actual, err := c.GetOrCreate(ctx, "key", func() (interface{}, error) {
				return &entry{Value: 1}, nil
			})
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, &entry{Value: 54})
		})
	})
}
