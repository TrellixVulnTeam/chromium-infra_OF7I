// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.chromium.org/luci/common/logging"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	kpb "infra/cmd/package_index/kythe/proto"
)

// The kzip directories to write data files and compilation units into.
var filesDir = "kzip/files/"
var unitsDir = "kzip/pbunits/"

// kzipEntry stores the path to a file to be written in the kzip and its contents.
type kzipEntry struct {
	path    string
	content []byte
}

// mergeExistingKzips iterates through files inside existingJavaKzipsPath
// and sends existing kzip files to kzipChannel to be processed.
//
// There might be more than one kzip file for the same target. This happens
// when arguments to javac change and ninja can't remove dynamically generated
// kzip file.
//
// Sort by modified time, and discard older targets in processExistingKzips
// that have the same output key.
func (ip *indexPack) mergeExistingKzips(existingKzipChannel chan<- string) error {
	f, err := os.Open(ip.existingJavaKzipsPath)
	if err != nil {
		return err
	}
	files, err := f.Readdir(-1)
	defer f.Close()
	if err != nil {
		return err
	}

	var kzips []os.FileInfo
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".kzip" {
			kzips = append(kzips, file)
		}
	}

	// Sort by modified time, reversed.
	sort.Slice(kzips, func(i, j int) bool {
		return kzips[j].ModTime().Before(kzips[i].ModTime())
	})
	for _, file := range kzips {
		existingKzipChannel <- filepath.Join(ip.existingJavaKzipsPath, file.Name())
	}

	close(existingKzipChannel)
	return nil
}

// processExistingKzips reads existing kzips then extracts and sends compilation
// units to be written to the final kzip output.
//
// outputSet is used to keep track of units already processed based on their output keys.
// Since files from existingKzipChannel are sorted by mod time, this must be run
// by a single goroutine.
func (ip *indexPack) processExistingKzips(ctx context.Context, existingKzipChannel <-chan string,
	kzipEntryChannel chan<- kzipEntry, outputSet *ConcurrentSet) error {
	for kzip := range existingKzipChannel {
		err := ip.processExistingKzip(ctx, kzip, kzipEntryChannel, outputSet)
		if err != nil {
			return err
		}
	}

	return nil
}

// processExistingKzip processes a single kzipEntry. Called by processExistingKzips.
// kzip should contain following structure:
//
// foo/
// foo/files
// foo/files/bar
// foo/units
// foo/units/bar
//
// We only care for foo/files/* and foo/units/* and we expect only one file
// in foo/units/ directory.
func (ip *indexPack) processExistingKzip(ctx context.Context, kzip string, kzipEntryChannel chan<- kzipEntry,
	outputSet *ConcurrentSet) error {
	var unit *zip.File
	files := make(map[string]*zip.File)

	r, err := zip.OpenReader(kzip)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, zipInfo := range r.File {
		fName := zipInfo.Name
		segments := strings.Split(fName, "/")

		if len(segments) != 3 || segments[len(segments)-1] == "" {
			continue
		}

		if segments[1] == "units" {
			if unit != nil {
				logging.Warningf(ctx, "Ignoring kzip file as more than one unit in kzip file %s.", fName)
				return nil
			}
			unit = zipInfo
		} else if segments[1] == "files" {
			files[segments[2]] = zipInfo
		} else {
			logging.Warningf(ctx, "Unexpected file %s in kzip %s.", fName, kzip)
		}
	}

	if unit == nil {
		logging.Warningf(ctx, "Ignoring kzip file %s as unit file is not found.", kzip)
	}

	// Add unit file.
	rc, err := unit.Open()
	if err != nil {
		logging.Warningf(ctx, "Error opening generated zip file unit %s: %s", kzip, err)
		return err
	}
	defer rc.Close()

	content := make([]byte, unit.UncompressedSize64)
	_, err = io.ReadFull(rc, content)
	if err != nil {
		logging.Warningf(ctx, "Error reading generated zip file unit %s: %s", kzip, err)
		return err
	}

	// Units in Java zip archive are json encoded.
	// Convert JSON to protobuf.
	indexedCompilationProto := &kpb.IndexedCompilation{}
	err = protojson.Unmarshal(content, indexedCompilationProto)
	if err != nil {
		logging.Warningf(ctx, "Error parsing JSON")
		return err
	}
	protoUnit := indexedCompilationProto.GetUnit()
	outputKey := protoUnit.GetOutputKey()

	// Check if outputKey has already been processed and add if not.
	if !outputSet.Add(outputKey) {
		logging.Infof(ctx, "Duplicated unit \"%s\" (filename: %s)", outputKey, kzip)
		return nil
	}

	if protoUnit != nil && ip.buildConfig != "" {
		injectUnitBuildDetails(ip.ctx, protoUnit, ip.buildConfig)
	}

	entry, err := indexedCompilationToKzipEntry(indexedCompilationProto)
	if err != nil {
		return err
	}
	kzipEntryChannel <- entry
	logging.Debugf(ctx, "Added %s from kzip", entry.path)

	for _, file := range files {
		rc, err = file.Open()
		if err != nil {
			return err
		}

		content = make([]byte, file.UncompressedSize64)
		_, err = io.ReadFull(rc, content)
		if err != nil {
			return err
		}

		if !ip.hashMaps.Add(file.Name, file.Name) {
			// File already added.
			continue
		}

		kzipEntryChannel <- kzipEntry{filesDir + filepath.Base(file.Name), content}
		logging.Debugf(ctx, "Added %s from kzip", file.Name)

		rc.Close()
	}

	return nil
}

// validateUnit checks that the unit's source file is present in required inputs.
func validateUnit(unit *kpb.CompilationUnit) bool {
	for _, requiredInput := range unit.GetRequiredInput() {
		if requiredInput.GetInfo().GetPath() == unit.GetSourceFile()[0] {
			return true
		}
	}
	return false
}

// unitFileToKzipEntry takes compilationUnits from unitProtoChannel and
// creates a kzipEntry to be written to the kzip archive.
func (ip *indexPack) unitFileToKzipEntry(ctx context.Context,
	unitProtoChannel <-chan *kpb.CompilationUnit, kzipEntryChannel chan<- kzipEntry) error {
	for unitProto := range unitProtoChannel {
		// Some units may not have their source file as a required input.
		if !validateUnit(unitProto) {
			// If invalid, drop these units and continue processing.
			logging.Errorf(ctx,
				"Skipping unit since source file %s was not in required inputs.", unitProto.GetSourceFile())
			continue
		}

		logging.Debugf(ctx, "Unit argument: %s", unitProto.GetArgument())
		indexedCompilationProto := &kpb.IndexedCompilation{
			Unit: unitProto,
		}

		// Dump the unit in proto wire format.
		entry, err := indexedCompilationToKzipEntry(indexedCompilationProto)
		if err != nil {
			return err
		}
		kzipEntryChannel <- entry
		logging.Debugf(ctx, "Writing compilation unit file %s", entry.path)
	}

	return nil
}

// dataFileToKzipEntry takes files from dataFileChannel and
// creates a kzipEntry to be written to the kzip archive.
func (ip *indexPack) dataFileToKzipEntry(ctx context.Context,
	dataFileChannel <-chan string, kzipEntryChannel chan<- kzipEntry) error {
	for fname := range dataFileChannel {
		fname, err := filepath.Abs(fname)
		if err != nil {
			return err
		}

		if _, ok := ip.hashMaps.Filehash(fname); !ok {
			if _, err := os.Stat(fname); os.IsNotExist(err) {
				logging.Warningf(ctx, "File %s does not exist: %s", fname, err)
				continue
			}

			content, err := ioutil.ReadFile(fname)
			if err != nil {
				return err
			}
			hashByte := sha256.Sum256(content)
			hash := hex.EncodeToString(hashByte[:])

			if !ip.hashMaps.Add(fname, hash) {
				warning := fmt.Sprintf("Not including source file: %s ", fname)
				otherFname, ok := ip.hashMaps.Filename(hash)
				if ok {
					warning += fmt.Sprintf("because it has the same hash as: %s", otherFname)
				}
				logging.Warningf(ctx, warning)
				continue
			}

			hashFname := filesDir + hash
			logging.Debugf(ctx, "Including source file %s as %s for compilation", fname, hashFname)

			kzipEntryChannel <- kzipEntry{hashFname, content}
		}
	}

	return nil
}

// indexedCompilationToKzipEntry converts a IndexedCompilation struct to a kzipEntry.
// If there is an error marshaling the contents of the given proto, this method
// returns an empty kzipEntry and the error.
func indexedCompilationToKzipEntry(indexedCompilationProto *kpb.IndexedCompilation) (kzipEntry, error) {
	content, err := proto.Marshal(indexedCompilationProto)
	if err != nil {
		return kzipEntry{}, err
	}
	hash := sha256.Sum256(content)
	path := unitsDir + hex.EncodeToString(hash[:])
	return kzipEntry{path, content}, nil
}

// writeToKzip first creates the kzip file with the appropriate directory structure
// and writes kzipEntries to the kzip archive.
func (ip *indexPack) writeToKzip(kzipEntryChannel <-chan kzipEntry) error {
	kzip, err := os.Create(ip.outputFile)
	if err != nil {
		return err
	}
	defer kzip.Close()

	// Create the directories inside kzip.
	w := zip.NewWriter(kzip)
	_, err = w.Create("kzip/")
	if err != nil {
		return err
	}
	_, err = w.Create(filesDir)
	if err != nil {
		return err
	}
	_, err = w.Create(unitsDir)
	if err != nil {
		return err
	}
	defer w.Close()

	// Write kzip entries into kzip.
	for entry := range kzipEntryChannel {
		f, err := w.Create(entry.path)
		if err != nil {
			return err
		}
		_, err = f.Write(entry.content)
		if err != nil {
			return err
		}
	}

	return nil
}
