// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"go.chromium.org/luci/common/clock/testclock"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestCache(t *testing.T) {
	t.Parallel()

	Convey("With temp dir", t, func() {
		tmp, err := ioutil.TempDir("", "gaedeploy_test")
		So(err, ShouldBeNil)
		Reset(func() { os.RemoveAll(tmp) })

		testTime := testclock.TestRecentTimeLocal.Round(time.Second)
		ctx, tc := testclock.UseTime(context.Background(), testTime)

		cache := Cache{Root: filepath.Join(tmp, "cache")}

		scan := func() []string {
			files, err := ioutil.ReadDir(cache.Root)
			So(err, ShouldBeNil)
			names := make([]string, len(files))
			for i, f := range files {
				names[i] = f.Name()
			}
			return names
		}

		Convey("WithTarball happy path", func() {
			src := testSrc{
				data: map[string]string{
					"dir/":     "",
					"dir/file": "hi",
				},
			}

			callback := func(path string) error {
				blob, err := ioutil.ReadFile(filepath.Join(path, "dir", "file"))
				So(err, ShouldBeNil)
				So(string(blob), ShouldResemble, "hi")
				return nil
			}

			err := cache.WithTarball(ctx, &src, callback)
			So(err, ShouldBeNil)
			So(src.calls, ShouldEqual, 1)

			// Updated the metadata
			entryDir := filepath.Join(cache.Root, hex.EncodeToString(src.SHA256()))
			m, err := readMetadata(ctx, entryDir)
			So(err, ShouldBeNil)
			So(m, ShouldResemble, cacheMetadata{
				Created: testTime,
				Touched: testTime,
			})

			tc.Add(time.Minute)

			err = cache.WithTarball(ctx, &src, callback)
			So(err, ShouldBeNil)
			So(src.calls, ShouldEqual, 1) // didn't touch the source

			// Updated the metadata
			m, err = readMetadata(ctx, entryDir)
			So(err, ShouldBeNil)
			So(m, ShouldResemble, cacheMetadata{
				Created: testTime,
				Touched: testTime.Add(time.Minute),
			})
		})

		Convey("WithTarball wrong hash", func() {
			src := testSrc{
				data: map[string]string{
					"dir/":     "",
					"dir/file": "hi",
				},
				sha256: bytes.Repeat([]byte{0}, 32),
			}
			err := cache.WithTarball(ctx, &src, func(path string) error {
				panic("must not be called")
			})
			So(err, ShouldErrLike, "tarball hash mismatch")
		})

		Convey("Trim works", func() {
			var created []string // oldest to newest
			for i := 0; i < 3; i++ {
				src := testSrc{
					data: map[string]string{"file": fmt.Sprintf("file %d", i)},
				}
				created = append(created, hex.EncodeToString(src.SHA256()))
				err := cache.WithTarball(ctx, &src, func(path string) error { return nil })
				So(err, ShouldBeNil)
				tc.Add(time.Minute)
			}

			So(scan(), ShouldHaveLength, len(created))

			// Kick two oldest ones (keep only one newest).
			So(cache.Trim(ctx, 1), ShouldBeNil)

			// Worked!
			So(scan(), ShouldResemble, []string{created[2]})
		})
	})
}

type testSrc struct {
	data  map[string]string
	calls int

	blob   []byte
	sha256 []byte
}

func (src *testSrc) asTarGz() []byte {
	if src.blob != nil {
		return src.blob
	}

	keys := make([]string, 0, len(src.data))
	for k := range src.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := bytes.Buffer{}
	gz := gzip.NewWriter(&buf)
	tr := tar.NewWriter(gz)

	for _, key := range keys {
		body := src.data[key]
		hdr := &tar.Header{
			Name: key,
			Mode: 0600,
			Size: int64(len(body)),
		}
		if err := tr.WriteHeader(hdr); err != nil {
			panic(err)
		}
		if _, err := tr.Write([]byte(body)); err != nil {
			panic(err)
		}
	}

	if err := tr.Close(); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func (src *testSrc) SHA256() []byte {
	if src.sha256 != nil {
		return src.sha256
	}
	h := sha256.New()
	h.Write(src.asTarGz())
	src.sha256 = h.Sum(nil)
	return src.sha256
}

func (src *testSrc) Open(ctx context.Context, tmp string) (io.ReadCloser, error) {
	src.calls++
	return ioutil.NopCloser(bytes.NewReader(src.asTarGz())), nil
}
