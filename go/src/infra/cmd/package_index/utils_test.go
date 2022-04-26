package main

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/golang/protobuf/ptypes"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/proto"

	kpb "infra/cmd/package_index/kythe/proto"
)

func TestRemoveFilepathsFiles(t *testing.T) {
	t.Parallel()
	Convey("Remove filepaths files", t, func() {
		ctx := context.Background()
		tmpdir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		Convey("Remove filepaths with files", func() {
			// File setup
			f, err := os.Create(filepath.Join(tmpdir, "foo.filepaths"))
			if err != nil {
				t.Fatal(err)
			}
			f.Close()
			fpath, err := filepath.Abs(f.Name())
			if err != nil {
				t.Fatal(err)
			}

			nested, err := ioutil.TempDir(tmpdir, "")
			if err != nil {
				t.Fatal(err)
			}

			b, err := os.Create(filepath.Join(nested, "bar.filepaths"))
			if err != nil {
				t.Fatal(err)
			}
			b.Close()
			bpath, err := filepath.Abs(b.Name())
			if err != nil {
				t.Fatal(err)
			}

			removeFilepathsFiles(ctx, tmpdir)

			Convey("tmpdir should be empty while the nested dir should remain unchanged", func() {
				_, err = os.Stat(fpath)
				So(os.IsNotExist(err), ShouldEqual, true)

				_, err = os.Stat(bpath)
				So(err, ShouldEqual, nil)
			})
		})

		Convey("Remove filepaths without filepaths files", func() {
			h, err := os.Create(filepath.Join(tmpdir, "hello.txt"))
			if err != nil {
				t.Fatal(err)
			}
			h.Close()
			hpath, err := filepath.Abs(h.Name())
			if err != nil {
				t.Fatal(err)
			}

			removeFilepathsFiles(ctx, tmpdir)

			Convey("Nothing should have changed", func() {
				_, err = os.Stat(hpath)
				So(err, ShouldEqual, nil)
			})
		})

	})
}

func TestConvertGnPath(t *testing.T) {
	t.Parallel()
	Convey("Convert GN Path", t, func() {
		ctx := context.Background()
		gnPath := "//path/tests/thing.txt"

		Convey("Bad outDir", func() {
			So(func() {
				convertGnPath(ctx, gnPath, "/bad/dir/")
			}, ShouldPanic)
		})

		Convey("GN path", func() {
			Convey("GN path should be relative to outDir", func() {
				r, err := convertGnPath(ctx, gnPath, "src/path/to/out")
				if err != nil {
					t.Fatal(err)
				}
				So(r, ShouldEqual, filepath.Join("..", "..", "tests", "thing.txt"))
			})
		})
	})
}

func TestNormalizePath(t *testing.T) {
	t.Parallel()
	Convey("Normalize path", t, func() {
		dir := "test/dir/"
		p := "/path/to/file.txt"

		Convey("Joining path", func() {
			Convey("The path should be cleaned and joined", func() {
				So(normalizePath(dir, p), ShouldEqual, filepath.Join("test", "dir", "path", "to", "file.txt"))
			})
		})
	})
}

func TestInjectDetails(t *testing.T) {
	t.Parallel()
	Convey("Inject unit details", t, func() {
		ctx := context.Background()
		buildConfig := "testconfig"

		Convey("BuildDetails not found in unitProto", func() {
			unitProto := &kpb.CompilationUnit{}
			injectUnitBuildDetails(ctx, unitProto, buildConfig)

			Convey("New details should be added to unitProto", func() {
				So(len(unitProto.GetDetails()), ShouldEqual, 1)

				any := unitProto.GetDetails()[0]
				So(any.GetTypeUrl(), ShouldEqual, "kythe.io/proto/kythe.proto.BuildDetails")

				buildDetails := &kpb.BuildDetails{}
				proto.Unmarshal(any.GetValue(), buildDetails)
				So(buildDetails.GetBuildConfig(), ShouldEqual, buildConfig)
			})
		})

		Convey("BuildDetails found in unitProto", func() {
			// unitProto details setup
			unitProto := &kpb.CompilationUnit{}
			details := &kpb.BuildDetails{}
			details.BuildConfig = ""
			anyDetails, err := ptypes.MarshalAny(details)
			if err != nil {
				t.Fatal(err)
			}

			anyDetails.TypeUrl = "kythe.io/proto/kythe.proto.BuildDetails"
			unitProto.Details = append(unitProto.Details, anyDetails)
			injectUnitBuildDetails(ctx, unitProto, buildConfig)

			Convey("Any details should modified in unitProto", func() {
				So(len(unitProto.GetDetails()), ShouldEqual, 1)

				any := unitProto.GetDetails()[0]
				So(any.GetTypeUrl(), ShouldEqual, "kythe.io/proto/kythe.proto.BuildDetails")

				buildDetails := &kpb.BuildDetails{}
				proto.Unmarshal(any.GetValue(), buildDetails)
				So(buildDetails.GetBuildConfig(), ShouldEqual, buildConfig)
			})
		})
	})
}

func TestFindImports(t *testing.T) {
	t.Parallel()
	Convey("Find imports", t, func() {
		ctx := context.Background()
		re := regexp.MustCompile(`(?m)^\s*import\s*(?:weak|public)?\s*"([^"]*)\s*";`)

		// File setup
		tmpdir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		importPaths := []string{tmpdir}

		f, err := os.Create(tmpdir + "/test.proto")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		f.WriteString("import \"foo.proto\";\nimport weak \"bar.proto\";\n")
		fpath, err := filepath.Abs(f.Name())
		if err != nil {
			t.Fatal(err)
		}

		Convey("File doesn't exist", func() {
			p := "/path/to/no/file"
			Convey("Imports should be empty", func() {
				So(len(findImports(ctx, re, p, importPaths)), ShouldEqual, 0)
			})
		})

		Convey("Import file doesn't exist", func() {
			Convey("Imports should be empty", func() {
				So(len(findImports(ctx, re, fpath, importPaths)), ShouldEqual, 0)
			})
		})

		Convey("Import file exists", func() {
			// Create import files
			fooPath := filepath.Join(tmpdir, "foo.proto")
			foo, err := os.Create(fooPath)
			if err != nil {
				t.Fatal(err)
			}
			defer foo.Close()

			barPath := filepath.Join(tmpdir, "bar.proto")
			bar, err := os.Create(barPath)
			if err != nil {
				t.Fatal(err)
			}
			defer bar.Close()

			Convey("Should return two imports", func() {
				r := findImports(ctx, re, fpath, importPaths)
				So(len(r), ShouldEqual, 2)
				So(r, ShouldContain, fooPath)
				So(r, ShouldContain, barPath)
			})
		})
	})
}

func TestSetVname(t *testing.T) {
	t.Parallel()
	Convey("Inject unit details", t, func() {
		var vnameProto kpb.VName
		vnameProtoRoot := "root"
		vnameProto.Root = vnameProtoRoot

		defaultCorpus := "corpus"

		Convey("Bad filepath", func() {
			p := "\\bad\\path"

			So(func() {
				setVnameForFile(&vnameProto, p, defaultCorpus)
			}, ShouldPanic)
		})

		Convey("Filepath has special corpus", func() {
			p := "third_party/depot_tools/win_toolchain/rest/of/path"
			setVnameForFile(&vnameProto, p, defaultCorpus)

			Convey("Should modify vnameProto with special/external settings", func() {
				So(vnameProto.Path, ShouldEqual, "rest/of/path")
				So(vnameProto.Root, ShouldEqual, "third_party/depot_tools/win_toolchain")
			})
		})

		Convey("Filepath has no special corpus", func() {
			p := "src/build/rest/of/path"
			setVnameForFile(&vnameProto, p, defaultCorpus)

			Convey("Should vnameProto with default settings", func() {
				So(vnameProto.Path, ShouldEqual, "build/rest/of/path")
				So(vnameProto.Root, ShouldEqual, vnameProtoRoot)
				So(vnameProto.Corpus, ShouldEqual, defaultCorpus)
			})
		})
	})
}

func TestUnwantedWinArg(t *testing.T) {
	t.Parallel()
	Convey("Win args", t, func() {

		Convey("Wanted arg", func() {
			arg := "-O"

			Convey("Arg should be included", func() {
				So(isUnwantedWinArg(arg), ShouldEqual, false)
			})
		})

		Convey("Unwanted arg", func() {
			arg := "-DSK_USER_CONFIG_HEADER"

			Convey("Arg shouldn't be included", func() {
				So(isUnwantedWinArg(arg), ShouldEqual, true)
			})
		})
	})
}
