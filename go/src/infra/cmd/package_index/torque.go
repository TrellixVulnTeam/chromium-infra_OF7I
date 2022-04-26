// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/common/data/stringset"

	kpb "infra/cmd/package_index/kythe/proto"
)

type torqueTarget struct {
	imports     []string
	args        []string
	rootDir     string
	outDir      string
	corpus      string
	buildConfig string
	targetName  string
	target      gnTargetInfo
	hashMap     *FileHashMap
	ctx         context.Context
}

func newTorqueTarget(ctx context.Context, gnTargetDict map[string]gnTargetInfo,
	targetName string, hashMap *FileHashMap, rootDir, outDir, corpus,
	buildConfig string) (*torqueTarget, error) {
	m := &torqueTarget{
		ctx:         ctx,
		targetName:  targetName,
		rootDir:     rootDir,
		outDir:      outDir,
		corpus:      corpus,
		buildConfig: buildConfig,
		hashMap:     hashMap,
	}
	m.target = gnTargetDict[m.targetName]
	m.args = gnTargetDict[m.targetName].Args
	imp, err := m.findTorqueImports()
	if err != nil {
		return nil, err
	}
	m.imports = imp
	return m, nil
}

func (m *torqueTarget) findTorqueImports() ([]string, error) {
	imports := stringset.New(0)
	// TODO(nicohartmann@, v8:12261): Need to extract included C++ files here
	// when adding support for xrefs to generated files.
	return imports.ToSlice(), nil
}

func (m *torqueTarget) getUnit() (*kpb.CompilationUnit, error) {
	unitProto := &kpb.CompilationUnit{}
	var sourceFiles []string

	for _, src := range m.target.Sources {
		gn, err := convertGnPath(m.ctx, src, m.outDir)
		if err != nil {
			return nil, err
		}
		sourceFiles = append(sourceFiles, convertPathToForwardSlashes(gn))
	}

	unitProto.SourceFile = sourceFiles

	// TODO(nicohartmann@, v8:12261): Might have to capture some arguments here
	// when supporting generated files.
	unitProto.VName = &kpb.VName{Corpus: m.corpus, Language: "torque"}
	// TODO(nicohartmann@, v8:12261): Might have to capture some build details
	// here.

	var importedFiles []string
	// TODO(nicohartmann@, v8:12261): Might have to add imported C++ files here
	// when supporting xrefs to generated C++ files.
	for _, requiredFile := range append(sourceFiles, importedFiles...) {
		p, err := filepath.Abs(filepath.Join(m.rootDir, m.outDir, requiredFile))
		if err != nil {
			return nil, err
		}

		h, _ := m.hashMap.Filehash(p)

		vname := &kpb.VName{}
		setVnameForFile(vname, convertPathToForwardSlashes(
			normalizePath(m.outDir, requiredFile)), m.corpus)
		requiredInput := &kpb.CompilationUnit_FileInput{
			VName: vname,
			Info: &kpb.FileInfo{
				Digest: h,
				Path:   convertPathToForwardSlashes(requiredFile),
			},
		}
		unitProto.RequiredInput = append(unitProto.GetRequiredInput(),
			requiredInput)
	}

	return unitProto, nil
}

func (m *torqueTarget) getFiles() ([]string, error) {
	var dataFiles []string
	for _, src := range m.target.Sources {
		gn, err := convertGnPath(m.ctx, src, m.outDir)
		if err != nil {
			return nil, err
		}
		dataFiles = append(dataFiles, filepath.Join(m.rootDir, m.outDir, gn))
	}

	return dataFiles, nil
}

func torqueTargetProcessor(ctx context.Context, rootPath, outDir, corpus,
	buildConfig string, hashMaps *FileHashMap, t *gnTarget) (
	GnTargetInterface, error) {

	if !isTorqueTarget(t) {
		return nil, errNotSupported
	}

	return newTorqueTarget(ctx, gnTargetsMap, t.targetName, hashMaps,
		rootPath, outDir, corpus, buildConfig)
}

func isTorqueTarget(t *gnTarget) bool {
	return strings.HasSuffix(t.targetName, "run_torque")
}
