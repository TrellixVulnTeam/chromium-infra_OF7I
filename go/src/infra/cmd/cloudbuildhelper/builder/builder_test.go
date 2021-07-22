// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package builder

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"go.chromium.org/luci/common/logging/gologger"

	"infra/cmd/cloudbuildhelper/fileset"
	"infra/cmd/cloudbuildhelper/manifest"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	if os.Getenv("GO111MODULE") == "off" {
		srcDir, err := filepath.Abs("testdata")
		if err != nil {
			panic(err)
		}
		os.Setenv("GOPATH", srcDir)
	} else {
		// Our test module has "vendor" directory and has go >=1.14 in go.mod, so
		// "-mod=vendor" is the mode that Go should be picking. But on CI builder
		// GOFLAGS may override it to "-mod=readonly" which breaks the test. Set it
		// explicitly.
		os.Setenv("GOFLAGS", "-mod=vendor")
	}
}

func TestBuilder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = gologger.StdConfig.Use(ctx)

	Convey("With temp dir", t, func() {
		srcDir, err := filepath.Abs("testdata")
		So(err, ShouldBeNil)

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

			out, err := build(fmt.Sprintf(`{
				"name": "test",
				"contextdir": "ctx",
				"inputsdir": %q,
				"build": [
					{
						"copy": "${manifestdir}/copy",
						"dest": "${contextdir}"
					},
					{
						"go_binary": "testpkg/helloworld",
						"dest": "${contextdir}/gocmd",
					},
					{
						"run": [
							"go",
							"run",
							"testpkg/helloworld",
							"${contextdir}/say_hi"
						],
						"outputs": ["${contextdir}/say_hi"]
					}
				]
			}`, filepath.Join(srcDir, "src", "testpkg")))
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
			// To test .gitignore handling, create a gitignored file manually, since
			// we can't check it in.
			err := ioutil.WriteFile(filepath.FromSlash("testdata/src/testpkg/helloworld/static/ignored"), nil, 0600)
			So(err, ShouldBeNil)

			m, err := manifest.Load(filepath.FromSlash("testdata/src/testpkg/gaebundle.yaml"))
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

			expected := []string{
				"_gopath/env",
				"_gopath/src/testpkg/helloworld/anotherpkg.go",
				"_gopath/src/testpkg/helloworld/buildflags_amd64.go",
				"_gopath/src/testpkg/helloworld/buildflags_linux.go",
				"_gopath/src/testpkg/helloworld/curgo.go",
				"_gopath/src/testpkg/helloworld/fake-app.yaml",
				"_gopath/src/testpkg/helloworld/main.go",
				"_gopath/src/testpkg/helloworld/static.txt",
				"_gopath/src/testpkg/helloworld/static/static.txt",
				"_gopath/src/testpkg/helloworld/vendor.go",
				"_gopath/src/testpkg/pkg1/pkg1.go",
				"_gopath/src/testpkg/pkg1/vendor.go",
				"_gopath/src/testpkg/pkg2/pkg2.go",
				"helloworld",
			}

			// In Go Modules mode, "vendor" can appear only at the module's root
			// directory (all other "vendor" directories are ignored). In GOPATH mode,
			// "vendor" directories can appear anywhere and they are used by the
			// corresponding packages.
			//
			// See https://golang.org/ref/mod#vendoring.
			if os.Getenv("GO111MODULE") != "off" {
				expected = append(expected,
					"_gopath/src/example.com/another/another_a.go",
					"_gopath/src/example.com/pkg/pkg_a.go",
				)
			} else {
				expected = append(expected,
					"_gopath/src/testpkg/pkg1/vendor/example.com/another/another_b.go",
					"_gopath/src/testpkg/pkg1/vendor/example.com/pkg/pkg_b.go",
					"_gopath/src/testpkg/vendor/example.com/another/another_a.go",
					"_gopath/src/testpkg/vendor/example.com/pkg/pkg_a.go",
				)
			}

			sort.Strings(expected)
			So(files, ShouldResemble, expected)

			So(byName["helloworld"], ShouldResemble, fileset.File{
				Path:          "helloworld",
				SymlinkTarget: "_gopath/src/testpkg/helloworld",
			})
		})
	})
}
