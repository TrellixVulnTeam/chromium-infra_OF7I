// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/logging"

	kpb "infra/cmd/package_index/kythe/proto"
)

// clangUnit contains all the JSON information for a given clang target.
type clangUnit struct {
	Directory string `json:"directory"`
	Command   string `json:"command"`
	File      string `json:"file"`
}

// clangUnitInfo contains a clangUnit struct and filepath filename
// for a clang target.
type clangUnitInfo struct {
	unit        clangUnit
	filepathsFn string
}

// ClangTargets contains info for Clang target processing.
type ClangTargets struct {
	once       sync.Once
	filePath   string
	entries    []clangUnit
	targetChan chan clangUnit
	targetsLen int

	// Keeps track of the '*.filepaths' files already processed.
	filepathsSet *ConcurrentSet

	// WaitGroup for closing targetDataOut.
	DataWg sync.WaitGroup

	// WaitGroup for closing targetUnitOut.
	UnitWg sync.WaitGroup

	// WaitGroup for deferring processing of compilation units until after data files
	// have been processed into kzip entries.
	KzipDataWg sync.WaitGroup
}

// NewClangTargets initializes a new ClangTargets struct.
func NewClangTargets(clangTargetsPath string) *ClangTargets {
	return &ClangTargets{
		filePath: clangTargetsPath,
	}
}

// populateChannel parses the compdb file from filePath and fills targetChan with
// the parsed JSON information.
func (clangTargets *ClangTargets) populateChannel() {
	if clangTargets.entries == nil {
		// Parse JSON
		dat, err := ioutil.ReadFile(clangTargets.filePath)
		if err != nil {
			panic(err)
		}
		json.Unmarshal(dat, &clangTargets.entries)
		clangTargets.targetsLen = len(clangTargets.entries)
		clangTargets.filepathsSet = NewConcurrentSet(clangTargets.targetsLen)
		clangTargets.targetChan = make(chan clangUnit, clangTargets.targetsLen)
	}
	for _, target := range clangTargets.entries {
		clangTargets.targetChan <- target
	}
	close(clangTargets.targetChan)
}

// ProcessClangTargets processes clang targets from a given compdb file in clangTargets' filePath.
func (clangTargets *ClangTargets) ProcessClangTargets(ctx context.Context, rootPath, outDir, corpus, buildConfig string,
	hashMaps *FileHashMap, targetDataOut chan<- string, targetUnitOut chan<- *kpb.CompilationUnit) error {
	// Parse compdb once.
	clangTargets.once.Do(clangTargets.populateChannel)

	// Channel for deferred compilation unit processing.
	clangTargetChan := make(chan *clangUnitInfo, clangTargets.targetsLen)

	// Process data files.
	for target := range clangTargets.targetChan {
		files, err := getClangFiles(ctx, clangTargets.filepathsSet, target, clangTargetChan)
		if err != nil {
			return err
		}
		if files == nil {
			continue
		}
		for _, f := range files {
			targetDataOut <- f
		}
	}
	close(clangTargetChan)
	clangTargets.DataWg.Done()

	// Wait for data files to be processed for writing to kzip.
	clangTargets.KzipDataWg.Wait()

	// Process compilation units.
	for clangInfo := range clangTargetChan {
		unit, err := getClangUnit(ctx, clangInfo, rootPath, outDir, corpus, buildConfig, hashMaps)
		if err != nil {
			return err
		}
		targetUnitOut <- unit
	}
	clangTargets.UnitWg.Done()
	return nil
}

// getClangFiles returns a list of filepaths needed for a given clang target.
func getClangFiles(ctx context.Context, filepathsSet *ConcurrentSet,
	target clangUnit, clangTargetChan chan<- *clangUnitInfo) ([]string, error) {
	var files []string
	filepathsFn := filepath.Join(target.Directory, target.File+".filepaths")
	// We don't want to fail if one of the filepaths doesn't exist. However we
	// keep track of it.
	if _, err := os.Stat(filepathsFn); os.IsNotExist(err) {
		return nil, nil
	}

	// For some reason, the compilation database contains the same targets more
	// than once. However we have just one file containing the file paths of
	// the involved files. So we can skip this target if we already processed
	// it.
	if !filepathsSet.Add(filepathsFn) {
		return nil, nil
	}

	// Send to clangTargetChan for deferred unit processing.
	clangTargetChan <- &clangUnitInfo{target, filepathsFn}

	// All file paths given in the *.filepaths file are either absolute paths
	// or relative to the directory target in the compilation database.
	file, err := os.Open(filepathsFn)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Each line in the '*.filepaths' file references the path to a source
		// file involved in the compilation.
		fname := strings.ReplaceAll(strings.TrimSpace(scanner.Text()), "//", "/")
		if !filepath.IsAbs(fname) {
			fname = filepath.Join(target.Directory, fname)
		}

		// We should not package builtin clang header files, see
		// crbug.com/513826
		if strings.Contains(fname, "third_party/llvm-build") {
			continue
		}
		files = append(files, fname)

		// .pb.h.meta file is required for CC/PB cross references
		if _, err := os.Stat(filepath.Join(fname + ".meta")); err == nil && strings.HasSuffix(fname, ".pb.h") {
			files = append(files, fname+".meta")
		}
	}

	if err = file.Close(); err != nil {
		return nil, err
	}
	return files, nil
}

// getClangUnit returns the compilation unit for a given clang target.
func getClangUnit(ctx context.Context, clangInfo *clangUnitInfo, rootPath, outDir, corpus, buildConfig string,
	hashMaps *FileHashMap) (*kpb.CompilationUnit, error) {
	unitProto := &kpb.CompilationUnit{}
	commandList, err := shellSplit(clangInfo.unit.Command)
	if err != nil {
		return nil, err
	}
	logging.Debugf(ctx, "Generating Translation Unit data for %s\nCompile command: %s",
		clangInfo.unit.File, clangInfo.unit.Command)

	// On some platforms, the |command_list| starts with the goma executable,
	// followed by the path to the clang executable (either clang++ or
	// clang-cl.exe). We want the clang executable to be the first parameter.
	for i, cmd := range commandList {
		if strings.Contains(cmd, "clang") {
			// Shorten the list of commands such that it starts with the path to
			// the clang executable.
			commandList = commandList[i:]
			break
		}
	}

	// Extract the output file argument.
	var outputFile string
	for i, cmd := range commandList {
		if cmd == "-o" && i+1 < len(commandList) {
			outputFile = commandList[i+1]
			break
		} else if strings.HasPrefix(cmd, "/Fo") {
			// Handle the Windows case.
			outputFile = cmd[len("/Fo"):]
			break
		}
	}

	if outputFile == "" {
		logging.Warningf(ctx, "No output file path found for %s\n", clangInfo.unit.File)
	}

	if strings.Contains(commandList[0], "clang-cl") {
		// Convert any args starting with -imsvc to use forward slashes, since
		// this is what Kythe expects.
		for i, cmd := range commandList {
			if strings.HasPrefix(cmd, "-imsvc") {
				commandList[i] = strings.ReplaceAll(cmd, "\\", "/")
			}
		}

		// HACK ALERT: Here we define header guards to prevent Kythe from using
		// the CUDA wrapper headers, which cause indexing errors.
		// The standard Kythe extractor dumps header search state to help the
		// indexer find the right headers, but we don't do that in this script.
		// The below lines work around it by excluding the CUDA headers entirely.
		commandList = append(commandList,
			"-D__CLANG_CUDA_WRAPPERS_NEW",
			"-D__CLANG_CUDA_WRAPPERS_COMPLEX",
			"-D__CLANG_CUDA_WRAPPERS_ALGORITHM",
		)

		// Remove any args that may cause errors with the Kythe indexer.
		ln := 0
		for _, arg := range commandList {
			if isUnwantedWinArg(arg) {
				continue
			}
			commandList[ln] = arg
			ln++
		}
		commandList = commandList[:ln]
	}

	// This macro is used to guard Kythe-specific pragmas, so we must define it
	// for Kythe to see them. In particular the kythe_inline_metadata pragma we
	// insert into mojom generated files.
	commandList = append(commandList, "-DKYTHE_IS_RUNNING=1")

	file, err := os.Open(clangInfo.filepathsFn)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fname := strings.TrimSpace(scanner.Text())
		// We should not package builtin clang header files, see
		// crbug.com/513826
		if strings.Contains(fname, "third_party/llvm-build") {
			continue
		}

		// The clang tool uses '//' to separate the system path where system
		// headers can be found from the relative path used in the #include
		// statement.
		if strings.Contains(fname, "//") {
			fname = strings.ReplaceAll(fname, "//", "/")
		}
		fnameFullpath, err := addClangUnitInput(ctx,
			fname, clangInfo.unit.Directory, outDir, corpus, hashMaps, unitProto)
		if err != nil {
			return nil, err
		}

		// .pb.h.meta file is required for CC/PB cross references
		if _, err := os.Stat(fnameFullpath + ".meta"); err == nil && strings.HasSuffix(fname, ".pb.h") {
			_, err = addClangUnitInput(ctx,
				fname+".meta", clangInfo.unit.Directory, outDir, corpus, hashMaps, unitProto)
			if err != nil {
				return nil, err
			}
		}
	}

	if err = file.Close(); err != nil {
		return nil, err
	}

	unitProto.SourceFile = append(unitProto.SourceFile, clangInfo.unit.File)
	unitProto.WorkingDirectory = convertPathToForwardSlashes(clangInfo.unit.Directory)
	unitProto.OutputKey = outputFile
	unitProto.VName = &kpb.VName{
		Corpus:   corpus,
		Language: "c++",
	}

	// Add the build config if specified.
	if buildConfig != "" {
		details := &kpb.BuildDetails{
			BuildConfig: buildConfig,
		}
		any, err := ptypes.MarshalAny(details)
		if err != nil {
			return nil, err
		}
		any.TypeUrl = "kythe.io/proto/kythe.proto.BuildDetails"
		unitProto.Details = append(unitProto.Details, any)
	}

	// Disable all warnings with -w so that the indexer can run successfully.
	// The job of the indexer is to index the code, not to verify it. Warnings
	// we actually care about should show up in the compile step.
	unitProto.Argument = append(unitProto.Argument, commandList...)
	unitProto.Argument = append(unitProto.Argument, "-w")
	return unitProto, nil
}

// addClangUnitInput adds required input to unitProto and returns full path to file
// that was added. Used as a helper function in getClangUnit.
func addClangUnitInput(ctx context.Context, fname, dir, outDir, corpus string, hashMaps *FileHashMap,
	unitProto *kpb.CompilationUnit) (string, error) {
	// Clean up fname and set to absolute path for use in hashMaps.
	fname = filepath.Clean(fname)
	fnameFullpath := fname

	// Paths in *.filepaths files are either absolute or relative to dir.
	// Format and clean fnameFullpath to make it consistent with entries in hashMaps.
	if !filepath.IsAbs(fnameFullpath) {
		fnameFullpathAbs, err := filepath.Abs(filepath.Join(dir, fname))
		if err != nil {
			return "", err
		}
		fnameFullpath = fnameFullpathAbs
	}
	fnameFullpath = filepath.Clean(fnameFullpath)
	hash, ok := hashMaps.Filehash(fnameFullpath)
	if !ok {
		logging.Warningf(ctx, "No information about required input file %s\n", fnameFullpath)
		return "", nil
	}

	// Handle absolute paths - when normalizing we assume paths are
	// relative to the output directory (e.g. src/out/Debug).
	if filepath.IsAbs(fname) {
		fnameRel, err := filepath.Rel(dir, fname)
		if err != nil {
			return "", err
		}
		fname = fnameRel
	}

	vname := &kpb.VName{}
	setVnameForFile(vname, convertPathToForwardSlashes(normalizePath(outDir, fname)), corpus)
	requiredInput := &kpb.CompilationUnit_FileInput{
		VName: vname,
		Info: &kpb.FileInfo{
			Path:   convertPathToForwardSlashes(fname),
			Digest: hash,
		},
	}

	unitProto.RequiredInput = append(unitProto.GetRequiredInput(), requiredInput)
	return fnameFullpath, nil
}
