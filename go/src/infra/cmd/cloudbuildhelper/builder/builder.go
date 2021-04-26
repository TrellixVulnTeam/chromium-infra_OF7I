// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package builder implement local build process.
package builder

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/cloudbuildhelper/fileset"
	"infra/cmd/cloudbuildhelper/gitignore"
	"infra/cmd/cloudbuildhelper/manifest"
)

// Builder executes build steps specified via "build" field in the manifest.
type Builder struct {
	tmpDir string
}

// New initializes a builder, allocating a temp directory for it.
func New() (*Builder, error) {
	tmpDir, err := ioutil.TempDir("", "cloudbuildhelper")
	if err != nil {
		return nil, errors.Annotate(err, "failed to allocate a temporary directory").Err()
	}
	return &Builder{tmpDir: tmpDir}, nil
}

// Close removes all temporary files held by this Builder.
//
// Closing the builder invalidates all outputs it ever produced, thus this
// should be done only after outputs are processed.
//
// Idempotent.
func (b *Builder) Close() error {
	if b.tmpDir != "" {
		if err := os.RemoveAll(b.tmpDir); err != nil {
			return errors.Annotate(err, "failed to remove builder temp dir").Err()
		}
		b.tmpDir = ""
	}
	return nil
}

// Build executes all local builds steps specified in the manifest.
//
// The result of this process is a fileset.Set with m.ContextDir and outputs of
// all build steps. Note that Builder is oblivious of Dockerfile or any other
// docker specifics. It just executes local steps specified via "build" field in
// the manifest.
//
// The returned fileset should not outlive Builder, since it may reference
// temporary files owned by Builder.
func (b *Builder) Build(ctx context.Context, m *manifest.Manifest) (*fileset.Set, error) {
	logging.Debugf(ctx, "Starting the local build using temp dir %q", b.tmpDir)

	out := &fileset.Set{}

	if m.ContextDir != "" {
		logging.Debugf(ctx, "Adding %q to the output set...", m.ContextDir)
		excluder, err := gitignore.NewExcluder(m.ContextDir)
		if err != nil {
			return nil, errors.Annotate(err, "when loading .gitignore files").Err()
		}
		if err := out.AddFromDisk(m.ContextDir, ".", excluder); err != nil {
			return nil, errors.Annotate(err, "failed to add contextdir %q to output set", m.ContextDir).Err()
		}
	}

	state := builderState{}

	for idx, bs := range m.Build {
		concrete := bs.Concrete()
		logging.Debugf(ctx, "Executing local build step #%d - %s", idx+1, concrete)

		var runner stepRunner
		switch concrete.(type) {
		case *manifest.CopyBuildStep:
			runner = runCopyBuildStep
		case *manifest.GoBuildStep:
			runner = runGoBuildStep
		case *manifest.RunBuildStep:
			runner = runRunBuildStep
		case *manifest.GoGAEBundleBuildStep:
			runner = runGoGAEBundleBuildStep
		default:
			panic("impossible, did you forget to implement a step?")
		}

		err := runner(ctx, &stepRunnerInv{
			State:      &state,
			Manifest:   m,
			BuildStep:  bs,
			Output:     out,
			TempDir:    b.tmpDir,
			TempSuffix: fmt.Sprintf("_%d", idx),
		})
		if err != nil {
			return nil, errors.Annotate(err, "local build step #%d (%s) failed", idx+1, concrete).Err()
		}
	}

	return out, nil
}

// stepRunner executes on local build step 'bs', writing result to 'out'.
type stepRunner func(ctx context.Context, inv *stepRunnerInv) error

// stepRunnerInv is a bundle of parameters for a stepRunner invocation.
type stepRunnerInv struct {
	State      *builderState       // a state is carried over between steps
	Manifest   *manifest.Manifest  // fully populated target manifest
	BuildStep  *manifest.BuildStep // build step we are executing
	Output     *fileset.Set        // where to put output files
	TempDir    string              // can be used to drop arbitrary files into
	TempSuffix string              // unique per-step suffix, to name temp files
}

// builderState is carried over between steps.
//
// They can mutate it if they want.
type builderState struct {
	goStdlibPerVersion map[string]stringset.Set // names of stdlib packaged discovered in GOROOT
	goDepsPerVersion   map[string]stringset.Set // import paths of packages copied to the _gopath already
}

// goStdlib is a set of stdlib packaged in GOROOT.
func (s *builderState) goStdlib(goVer string) stringset.Set {
	if s.goStdlibPerVersion == nil {
		s.goStdlibPerVersion = make(map[string]stringset.Set, 1)
	}
	ss := s.goStdlibPerVersion[goVer]
	if ss == nil {
		ss = stringset.New(0)
		s.goStdlibPerVersion[goVer] = ss
	}
	return ss
}

// goDeps is a set of imported packages copied to the _gopath already.
func (s *builderState) goDeps(goVer string) stringset.Set {
	if s.goDepsPerVersion == nil {
		s.goDepsPerVersion = make(map[string]stringset.Set, 1)
	}
	ss := s.goDepsPerVersion[goVer]
	if ss == nil {
		ss = stringset.New(0)
		s.goDepsPerVersion[goVer] = ss
	}
	return ss
}

// addFilesToOutput adds `src` (which is an existing file or directory on disk)
// to the output set as filepath.Rel(contextDir, dst), failing if the result is
// outside of the context dir.
func (inv *stepRunnerInv) addFilesToOutput(ctx context.Context, src, dst string, exclude fileset.Excluder) error {
	rel, err := filepath.Rel(inv.Manifest.ContextDir, dst)
	if err != nil {
		return err
	}
	logging.Infof(ctx, "Copying %s => ${contextdir}/%s", filepath.Base(src), rel)
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.Reason("the destination should be under the context directory, got %q", rel).Err()
	}
	return inv.Output.AddFromDisk(src, rel, exclude)
}

// addBlobToOutput adds a non-executable regular file to the output set as
// filepath.Rel(contextDir, dst), failing if the result is outside of the
// context dir.
func (inv *stepRunnerInv) addBlobToOutput(ctx context.Context, dst string, blob []byte) error {
	rel, err := filepath.Rel(inv.Manifest.ContextDir, dst)
	if err != nil {
		return err
	}
	logging.Infof(ctx, "Writing ${contextdir}/%s", rel)
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.Reason("the destination should be under the context directory, got %q", rel).Err()
	}
	return inv.Output.AddFromMemory(rel, blob, nil)
}
