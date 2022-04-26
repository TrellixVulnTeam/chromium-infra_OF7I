// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/logging"

	kpb "infra/cmd/package_index/kythe/proto"
)

// protoImportRe is used for finding required_input for a Proto target, by finding the imports in
// its source. Import spec:
// https://developers.google.com/protocol-buffers/docs/reference/proto3-spec#import_statement
var protoImportRe = regexp.MustCompile(`(?m)^\s*import\s*(?:weak|public)?\s*"([^"]*)\s*";`)

// protoTarget contains all information needed to process a proto target.
type protoTarget struct {
	allFiles    []string
	sources     []string
	args        []string
	rootDir     string
	outDir      string
	protoPaths  []string
	metaDirs    []string
	corpus      string
	buildConfig string
	filehashes  *FileHashMap
	ctx         context.Context
}

// newProtoTarget initializes a new protoTarget struct.
func newProtoTarget(ctx context.Context, t gnTargetInfo, rootDir, outDir,
	corpus, buildConfig string, filehashes *FileHashMap) (*protoTarget, error) {
	if !strings.HasPrefix(outDir, "src") {
		return nil, errNotSupported
	}

	p := &protoTarget{
		ctx:         ctx,
		corpus:      corpus,
		buildConfig: buildConfig,
		filehashes:  filehashes,
		sources:     t.Sources,
		rootDir:     rootDir,
		outDir:      outDir,
	}
	p.args = t.Args
	for i := 0; i < len(p.args)-1; i++ {
		nextArg := p.args[i+1]
		if p.args[i] == "--proto-in-dir" {
			norm, err := p.normpath(nextArg)
			if err != nil {
				return nil, err
			}
			p.protoPaths = append(p.protoPaths, norm)
		} else if p.args[i] == "--cc-out-dir" {
			// Additional Kythe metadata files need to be included as required_file.
			norm, err := p.normpath(nextArg)
			if err != nil {
				return nil, err
			}
			p.metaDirs = append(p.metaDirs, norm)
		}
	}

	importDirPrefix := "--import-dir="
	var debugArgs []string
	for _, arg := range p.args {
		debugArgs = append(debugArgs, arg)
		if strings.HasPrefix(arg, importDirPrefix) {
			norm, err := p.normpath(arg[len(importDirPrefix):])
			if err != nil {
				return nil, err
			}
			p.protoPaths = append(p.protoPaths, norm)
		}
	}

	if len(p.protoPaths) == 0 {
		p.protoPaths = append(p.protoPaths, filepath.Join(rootDir, outDir))
	}
	return p, nil
}

// getUnit returns a compilation unit for a proto target.
func (p protoTarget) getUnit() (*kpb.CompilationUnit, error) {
	unitProto := &kpb.CompilationUnit{}
	var sourceFiles []string
	for _, source := range p.sources {
		gn, err := convertGnPath(p.ctx, source, p.outDir)
		if err != nil {
			return nil, err
		}
		sourceFiles = append(sourceFiles, convertPathToForwardSlashes(gn))
	}

	// args are for protoc_wrapper.py and not protoc. Extract only --proto-in-dir.
	for i := 0; i < len(p.args)-1; i++ {
		if p.args[i] == "--proto-in-dir" || p.args[i] == "--import-dir" {
			unitProto.Argument = append(unitProto.Argument, "--proto_path", p.args[i+1])
		}
	}

	// Use sort to make it deterministic, used for unit tests.
	sort.Strings(sourceFiles)
	for _, source := range sourceFiles {
		unitProto.SourceFile = append(unitProto.SourceFile, source)
		// Append to arguments since original source argument needs to be modified.
		unitProto.Argument = append(unitProto.Argument, source)
	}

	unitProto.VName = &kpb.VName{Corpus: p.corpus, Language: "protobuf"}
	if p.buildConfig != "" {
		injectUnitBuildDetails(p.ctx, unitProto, p.buildConfig)
	}

	// Use sort to make it deterministic, used for unit tests.
	files, err := p.getFiles()
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	for _, f := range files {
		if _, ok := p.filehashes.Filehash(f); !ok {
			// Indexer can't recover from such error so don't bother with unit creation.
			logging.Warningf(p.ctx, "Missing file %s in filehashes, skipping unit completely.\n", f)
			return nil, nil
		}

		f, err := filepath.Abs(f)
		if err != nil {
			return nil, err
		}

		vnamePath, err := filepath.Rel(p.rootDir, f)
		if err != nil {
			return nil, err
		}

		infoPath, err := filepath.Rel(filepath.Join(p.rootDir, p.outDir), f)
		if err != nil {
			return nil, err
		}
		h, ok := p.filehashes.Filehash(f)
		if !ok {
			logging.Warningf(p.ctx, "File %s was not found.", f)
		}

		vname := &kpb.VName{}
		setVnameForFile(vname, convertPathToForwardSlashes(vnamePath), p.corpus)
		requiredInput := &kpb.CompilationUnit_FileInput{
			VName: vname,
			Info: &kpb.FileInfo{
				Digest: h,
				Path:   convertPathToForwardSlashes(infoPath),
			},
		}
		unitProto.RequiredInput = append(unitProto.GetRequiredInput(), requiredInput)
	}
	return unitProto, nil
}

// getFiles retrieves a list of all files that are required for compilation of a target.
// Returns a list of all included files, in their absolute paths.
func (p protoTarget) getFiles() ([]string, error) {
	if len(p.allFiles) == 0 {
		var paths []string

		// Use absolute paths as it makes things easier. gn_target sources start
		// with // so that needs to be stripped.
		for _, source := range p.sources {
			gn, err := convertGnPath(p.ctx, source, p.outDir)
			if err != nil {
				return nil, err
			}

			p, err := filepath.Abs(filepath.Join(p.rootDir, p.outDir, gn))
			if err != nil {
				return nil, err
			}
			paths = append(paths, p)
		}

		allFiles := stringset.New(0)
		for len(paths) > 0 {
			pth := paths[0]
			paths = paths[1:]
			if allFiles.Has(pth) {
				// Already processed.
				continue
			}
			allFiles.Add(pth)
			for _, imp := range findImports(p.ctx, protoImportRe, pth, p.protoPaths) {
				paths = append(paths, imp)
			}
		}

		// Include metadata files.
		for _, metaDir := range p.metaDirs {
			if _, err := os.Stat(metaDir); os.IsNotExist(err) {
				logging.Warningf(p.ctx, "Protobuf meta directory not found: %s\n", metaDir)
				continue
			}

			f, err := os.Open(metaDir)
			if err != nil {
				return nil, err
			}
			files, err := f.Readdirnames(-1)
			defer f.Close()
			for _, fname := range files {
				if strings.HasSuffix(fname, ".meta") {
					allFiles.Add(filepath.Join(metaDir, fname))
				}
			}
		}
		p.allFiles = allFiles.ToSlice()
	}
	return p.allFiles, nil
}

// protoTargetProcessor takes in a target and either returns an error if the target
// isn't a proto target or returns a processedTarget struct.
//
// If files is true, process the target files. Otherwise, process the target unit.
func protoTargetProcessor(ctx context.Context, rootPath, outDir, corpus, buildConfig string,
	hashMaps *FileHashMap, t *gnTarget) (GnTargetInterface, error) {
	if !t.IsProtoTarget() {
		return nil, errNotSupported
	}

	p, err := newProtoTarget(ctx, t.targetInfo, rootPath, outDir, corpus, buildConfig, hashMaps)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// IsProtoTarget returns true if t is a proto target.
func (t *gnTarget) IsProtoTarget() bool {
	if t.targetInfo.Script == "" {
		return false
	}
	script := t.targetInfo.Script
	if strings.HasSuffix(script, "/python2_action.py") && len(t.targetInfo.Args) > 0 {
		script = t.targetInfo.Args[0]
	}
	return strings.HasSuffix(script, "/protoc_wrapper.py")
}

// normpath is a helper function that returns a clean path of argPath relative to the protoTarget.
func (p protoTarget) normpath(argPath string) (string, error) {
	r, err := filepath.Abs(filepath.Join(p.rootDir, p.outDir, argPath))
	if err != nil {
		return "", err
	}
	return r, nil
}
