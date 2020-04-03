// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package builder

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"infra/cmd/cloudbuildhelper/fileset"
	"infra/cmd/cloudbuildhelper/manifest"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBuilder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("With temp dir", t, func() {
		tmpDir, err := ioutil.TempDir("", "builder_test")
		So(err, ShouldBeNil)
		Reset(func() { os.RemoveAll(tmpDir) })

		b, err := New()
		So(err, ShouldBeNil)
		defer b.Close()

		put := func(path, body string) {
			fp := filepath.Join(tmpDir, filepath.FromSlash(path))
			So(os.MkdirAll(filepath.Dir(fp), 0777), ShouldBeNil)
			So(ioutil.WriteFile(fp, []byte(body), 0666), ShouldBeNil)
		}

		build := func(manifestBody string) (*fileset.Set, error) {
			manifestPath := filepath.Join(tmpDir, "manifest.yaml")
			So(ioutil.WriteFile(manifestPath, []byte(manifestBody), 0600), ShouldBeNil)
			loaded, err := manifest.Load(manifestPath)
			So(err, ShouldBeNil)
			So(loaded.RenderSteps(), ShouldBeNil)
			return b.Build(ctx, loaded)
		}

		Convey("ContextDir only", func() {
			put("ctx/f1", "file 1")
			put("ctx/f2", "file 2")

			out, err := build(`{
				"name": "test",
				"contextdir": "ctx"
			}`)
			So(err, ShouldBeNil)
			So(out.Files(), ShouldHaveLength, 2)

			So(b.Close(), ShouldBeNil)
			So(b.Close(), ShouldBeNil) // idempotent
		})

		Convey("A bunch of steps", func() {
			put("ctx/f1", "file 1")
			put("ctx/f2", "file 2")

			put("copy/f1", "overridden")
			put("copy/dir/f", "f")

			out, err := build(`{
				"name": "test",
				"contextdir": "ctx",
				"build": [
					{
						"copy": "${manifestdir}/copy",
						"dest": "${contextdir}"
					},
					{
						"go_binary": "infra/cmd/cloudbuildhelper/builder/testing/helloworld",
						"dest": "${contextdir}/gocmd"
					},
					{
						"run": [
							"go",
							"run",
							"infra/cmd/cloudbuildhelper/builder/testing/helloworld",
							"${contextdir}/say_hi"
						],
						"outputs": ["${contextdir}/say_hi"]
					}
				]
			}`)
			So(err, ShouldBeNil)

			names := make([]string, out.Len())
			byName := make(map[string]fileset.File, out.Len())
			for i, f := range out.Files() {
				names[i] = f.Path
				byName[f.Path] = f
			}
			So(names, ShouldResemble, []string{
				"dir", "dir/f", "f1", "f2", "gocmd", "say_hi",
			})

			r, err := byName["f1"].Body()
			So(err, ShouldBeNil)
			blob, err := ioutil.ReadAll(r)
			So(err, ShouldBeNil)
			So(string(blob), ShouldEqual, "overridden")
		})

		Convey("Go GAE bundling", func() {
			m, err := manifest.Load(filepath.FromSlash("testing/gaebundle.yaml"))
			So(err, ShouldBeNil)
			m.ContextDir = tmpDir
			So(m.RenderSteps(), ShouldBeNil)

			out, err := b.Build(ctx, m)
			So(err, ShouldBeNil)

			files := make([]string, 0, out.Len())
			byName := make(map[string]fileset.File, out.Len())
			for _, f := range out.Files() {
				if !f.Directory {
					files = append(files, f.Path)
					byName[f.Path] = f
				}
			}
			So(files, ShouldResemble, []string{
				"_gopath/env",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/anotherpkg.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/buildflags_amd64.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/buildflags_linux.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/curgo.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/fake-app.yaml",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/main.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/static.txt",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/static/static.txt",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld/vendor.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/pkg1/pkg1.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/pkg2/pkg2.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/vendor/example.com/another/another.go",
				"_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/vendor/example.com/pkg/pkg.go",
				"helloworld",
			})

			So(byName["helloworld"], ShouldResemble, fileset.File{
				Path:          "helloworld",
				SymlinkTarget: "_gopath/src/infra/cmd/cloudbuildhelper/builder/testing/helloworld",
			})
		})
	})
}
