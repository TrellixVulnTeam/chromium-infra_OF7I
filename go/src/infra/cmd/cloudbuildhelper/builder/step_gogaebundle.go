// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package builder

import (
	"context"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v2"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/cloudbuildhelper/fileset"
	"infra/cmd/cloudbuildhelper/gitignore"
)

// runRunBuildStep executes manifest.RunBuildStep.
func runGoGAEBundleBuildStep(ctx context.Context, inv *stepRunnerInv) error {
	logging.Infof(ctx, "Bundling %q", inv.BuildStep.GoGAEBundle)

	yamlPath, err := filepath.Abs(inv.BuildStep.GoGAEBundle)
	if err != nil {
		return errors.Annotate(err, "failed to convert the path %q to absolute", inv.BuildStep.GoGAEBundle).Err()
	}

	// Read go runtime version from the YAML to know what Go build flags to use.
	//
	// It is either e.g. "go113" for GAE Standard or "go1.13" or just "go" for
	// GAE Flex.
	runtime, err := readRuntime(yamlPath)
	if err != nil {
		return err
	}
	logging.Infof(ctx, "Runtime is %q", runtime)
	if runtime != "go" && !strings.HasPrefix(runtime, "go1") {
		return errors.Reason("%q is not a supported go runtime", runtime).Err()
	}
	var goMinorVer int64
	if strings.HasPrefix(runtime, "go1") {
		runtime = strings.ReplaceAll(runtime, ".", "")
		if goMinorVer, err = strconv.ParseInt(runtime[3:], 10, 32); err != nil {
			return errors.Annotate(err, "can't parse %q", runtime).Err()
		}
	}

	// The directory with `main` package.
	mainDir := filepath.Dir(yamlPath)

	// Get a build.Context as if we are building for linux amd64. We primarily use
	// it to call its MatchFile method to check build tags.
	bc := buildContext(mainDir, int(goMinorVer))

	// Load the main package and all its transitive dependencies.
	mainPkg, err := loadPackageTree(ctx, bc)
	if err != nil {
		return err
	}

	// We'll copy `mainPkg` directly into its final location in _gopath, but make
	// the original intended destination point to it as a symlink.
	goPathDest := filepath.Join(inv.Manifest.ContextDir, "_gopath", "src", mainPkg.PkgPath)
	linkName, err := relPath(inv.Manifest.ContextDir, inv.BuildStep.Dest)
	if err != nil {
		return err
	}
	linkTarget, err := relPath(filepath.Dir(inv.BuildStep.Dest), goPathDest)
	if err != nil {
		return err
	}
	if err := inv.Output.AddSymlink(linkName, linkTarget); err != nil {
		return errors.Annotate(err, "failed to setup a symlink to location in _gopath").Err()
	}

	// Respect .gcloudignore files.
	excludedByIgnoreFile, err := gitignore.NewExcluder(mainDir, ".gcloudignore")
	if err != nil {
		return errors.Annotate(err, "when loading .gcloudignore files").Err()
	}

	// Copy all files that make up "main" package (they can be only at the root
	// of `mainDir`), and copy all non-go files recursively (they can potentially
	// be referenced by static_files in app.yaml). We'll deal with Go dependencies
	// separately.
	err = inv.addFilesToOutput(ctx, mainDir, goPathDest, func(absPath string, isDir bool) bool {
		switch {
		case excludedByIgnoreFile(absPath, isDir):
			return true // respect .gcloudignore exclusions
		case isDir:
			return false // do not exclude directories, may have contain static files
		}
		rel, err := relPath(mainDir, absPath)
		if err != nil {
			panic(fmt.Sprintf("impossible: %s", err))
		}
		switch {
		// Do not exclude non-code files regardless of where they are.
		case !isGoSourceFile(rel):
			return false
		// Exclude code files not in the mainDir. If they are needed, they'll be
		// discovered by the next step.
		case rel != filepath.Base(rel):
			return true
		// For code files in the mainDir, pick up only ones matching the build
		// context (linux amd64).
		default:
			matches, err := bc.MatchFile(mainDir, rel)
			if err != nil {
				logging.Warningf(ctx, "Failed to check whether %q matches the build context, skipping it: %s", absPath, err)
				return true
			}
			return !matches
		}
	})
	if err != nil {
		return err
	}

	// Packages for different go versions may have different files in them due to
	// filtering based on build tags. For each Go runtime we keep a separate map
	// of visited packages in this runtime. In practice it means if the GAE app
	// uses more than one runtime, all packages will be visited more than once.
	// Each separate visit may add more files to the output (or just revisit
	// already added ones, which is a noop).
	goDeps := inv.State.goDeps(runtime)

	errs := 0    // number of errors in packages.Visit below
	visited := 0 // number of packages actually visited
	copied := 0  // number of files copied

	reportErr := func(format string, args ...interface{}) {
		logging.Errorf(ctx, format, args...)
		errs++
	}

	// Copy all transitive dependencies into _gopath/src/<pkg>.
	logging.Infof(ctx, "Copying transitive dependencies...")
	packages.Visit([]*packages.Package{mainPkg}, nil, func(pkg *packages.Package) {
		switch {
		case errs != 0:
			return // failing already
		case !goDeps.Add(pkg.ID):
			return // added it already in some previous build step
		case isStdlib(bc, pkg):
			return // we are not bundling stdlib packages
		default:
			visited++
		}

		// List of file names to copy into the output. They all must be in the same
		// directory.
		filesToAdd := make([]string, 0, len(pkg.GoFiles)+len(pkg.IgnoredFiles)+len(pkg.OtherFiles))

		// All non-go source files (like *.c) go into the tarball as is.
		filesToAdd = append(filesToAdd, pkg.OtherFiles...)

		// We visit GoFiles and IgnoredFiles because we want to recheck the build
		// tags using bc.MatchFile: packages.Load *always* uses the current Go
		// version tags, but we want to apply bc.ReleaseTags instead. It means we
		// may need to pick up some files rejected by packages.Load (they end up in
		// IgnoredFiles list), or reject some files from GoFiles.
		addGoFiles := func(paths []string) {
			for _, p := range paths {
				switch match, err := bc.MatchFile(filepath.Split(p)); {
				case err != nil:
					reportErr("Failed to check build tags of %q: %s", p, err)
				case match:
					filesToAdd = append(filesToAdd, p)
				}
			}
		}
		addGoFiles(pkg.GoFiles)
		addGoFiles(pkg.IgnoredFiles)

		if len(filesToAdd) == 0 || errs != 0 {
			return
		}

		// Verify all files come from the same directory (since we are placing them
		// into the same directory in the tarball).
		srcDir := filepath.Dir(filesToAdd[0])
		for _, path := range filesToAdd {
			if filepath.Dir(path) != srcDir {
				reportErr("Expected %q to be under %q", path, srcDir)
			}
		}
		if errs != 0 {
			return
		}

		// Add them all to the tarball if not already there.
		dstDir := filepath.Join("_gopath", "src", pkg.PkgPath)
		for _, path := range filesToAdd {
			name := filepath.Base(path)
			err := inv.Output.AddFromDisk(path, filepath.Join(dstDir, name), nil)
			if err != nil {
				reportErr("Failed to copy %q to the tarball: %s", path, err)
			} else {
				copied++
			}
		}
	})
	if errs != 0 {
		return errors.Reason("failed to add Go files to the tarball, see the log").Err()
	}
	logging.Infof(ctx, "Visited %d packages and copied %d files", visited, copied)

	// Drop a script that can be used to test sanity of this tarball.
	envPath := filepath.Join("_gopath", "env")
	return inv.Output.AddFromMemory(envPath, []byte(envScript), &fileset.File{
		Executable: true,
	})
}

// readRuntime reads `runtime` field in the YAML file.
func readRuntime(path string) (string, error) {
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Annotate(err, "failed to read %q", path).Err()
	}

	var appYaml struct {
		Runtime string `yaml:"runtime"`
	}
	if err := yaml.Unmarshal(blob, &appYaml); err != nil {
		return "", errors.Annotate(err, "file %q is not a valid YAML", path).Err()
	}

	return appYaml.Runtime, nil
}

// buildContext returns a build.Context targeting linux-amd64.
//
// If goMinorVer is not 0, sets ReleaseTags to pick the specific go release.
func buildContext(mainDir string, goMinorVer int) *build.Context {
	bc := build.Default
	bc.GOARCH = "amd64"
	bc.GOOS = "linux"
	bc.Dir = mainDir
	if goMinorVer != 0 {
		bc.ReleaseTags = nil
		for i := 1; i <= goMinorVer; i++ {
			bc.ReleaseTags = append(bc.ReleaseTags, fmt.Sprintf("go1.%d", i))
		}
	}
	return &bc
}

// loadPackageTree loads the main package with its dependencies.
func loadPackageTree(ctx context.Context, bc *build.Context) (*packages.Package, error) {
	logging.Infof(ctx, "Loading the package tree...")

	// Note: this can actually download files into the modules cache when running
	// in module mode and thus can be quite slow.
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule,
		Context: ctx,
		Logf:    func(format string, args ...interface{}) { logging.Debugf(ctx, format, args...) },
		Dir:     bc.Dir,
		Env:     append(os.Environ(), "GOOS="+bc.GOOS, "GOARCH="+bc.GOARCH),
	}, ".")
	if err != nil {
		return nil, errors.Annotate(err, "failed to load the main package").Err()
	}

	// `packages.Load` records some errors inside packages.Package.
	errs := 0
	visited := 0
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		visited++
		for _, err := range pkg.Errors {
			logging.Errorf(ctx, "Error loading package %q: %s", pkg.ID, err)
			errs++
		}
	})
	if errs != 0 {
		return nil, errors.Reason("failed to load the package tree").Err()
	}

	// We expect only one package to match our load query.
	if len(pkgs) != 1 {
		return nil, errors.Reason("expected to load 1 package, but got %d", len(pkgs)).Err()
	}

	// Make sure it is indeed `main` and log its path in the package tree.
	mainPkg := pkgs[0]
	if mainPkg.PkgPath == "" {
		return nil, errors.Reason("could not figure out import path of the main package").Err()
	}
	logging.Infof(ctx, "Import path is %q", mainPkg.PkgPath)
	if mainPkg.Name != "main" {
		return nil, errors.Annotate(err, "only \"main\" package can be bundled, got %q", mainPkg.Name).Err()
	}
	if mainPkg.Module != nil {
		logging.Infof(ctx, "Module is %q at %q", mainPkg.Module.Path, mainPkg.Module.Dir)
	}

	logging.Infof(ctx, "Transitively depends on %d packages (including stdlib)", visited-1)
	return mainPkg, nil
}

// relPath calls filepath.Rel and annotates the error.
func relPath(base, path string) (string, error) {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return "", errors.Annotate(err, "failed to calculate rel(%q, %q)", base, path).Err()
	}
	return rel, nil
}

// isGoSourceFile returns true if rel may be read by Go compiler.
//
// See https://golang.org/src/go/build/build.go.
func isGoSourceFile(rel string) bool {
	switch filepath.Ext(rel) {
	case ".go", ".c", ".cc", ".cxx", ".cpp", ".m", ".s", ".h", ".hh", ".hpp", ".hxx", ".f", ".F", ".f90", ".S", ".sx", ".swig", ".swigcxx":
		return true
	default:
		return false
	}
}

// isStdlib returns true if the package has its *.go files under GOROOT.
func isStdlib(bc *build.Context, pkg *packages.Package) bool {
	switch {
	case pkg.Name == "unsafe":
		return true // this package is a magical indicator and has no Go files
	case len(pkg.GoFiles) == 0:
		return false // assume other stdlib packages have Go files
	default:
		root := filepath.Clean(bc.GOROOT) + string(filepath.Separator)
		return strings.HasPrefix(pkg.GoFiles[0], root)
	}
}

// envScript spits out a script that modifies Go env vars to point to files
// in the tarball. Can be used to manually test the tarball's soundness.
const envScript = `#!/usr/bin/env bash
cd $(dirname "${BASH_SOURCE[0]}")

echo "export GOARCH=amd64"
echo "export GOOS=linux"
echo "export GO111MODULE=off"
echo "export GOPATH=$(pwd)"
`
