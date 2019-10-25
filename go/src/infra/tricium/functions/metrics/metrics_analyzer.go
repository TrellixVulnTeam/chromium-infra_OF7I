// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	tricium "infra/tricium/api/v1"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/waigani/diffparser"
	"go.chromium.org/luci/common/data/stringset"
)

// enum contains all the data about a particular enum.
type enum struct {
	Name     string `xml:"name,attr"`
	Elements []struct {
		Value string `xml:"value,attr"`
		Label string `xml:"label,attr"`
	} `xml:"int"`
}

// enumFile contains all the data in an enums file.
type enumFile struct {
	Enums struct {
		EnumList []enum `xml:"enum"`
	} `xml:"enums"`
}

type diffsPerFile struct {
	addedLines   map[string][]int
	removedLines map[string][]int
}

func main() {
	inputDir := flag.String("input", "", "Path to root of Tricium input")
	outputDir := flag.String("output", "", "Path to root of Tricium output")
	prevDir := flag.String("previous", "", "Path to directory with previous versions of changed files")
	patchPath := flag.String("patch", "", "Path to patch of changed files")
	enumsPath := flag.String("enums", "src/tools/metrics/histograms/enums.xml", "Path to enums file")
	flag.Parse()
	// This is a temporary way for us to get recipes to work without breaking the current analyzer.
	var filePaths []string
	var filesChanged *diffsPerFile
	var err error
	if *prevDir != "" {
		filePaths = flag.Args()
		filesChanged, err = getDiffsPerFile(filePaths, *patchPath)
		if err != nil {
			log.Panicf("Failed to get diffs per file: %v", err)
		}
	} else {
		// Read Tricium input FILES data.
		input := &tricium.Data_Files{}
		if err := tricium.ReadDataType(*inputDir, input); err != nil {
			log.Panicf("Failed to read FILES data: %v. Did you specify a Tricium-compatible input directory with -input?", err)
		}
		log.Printf("Read FILES data.")
		// Only add histogram.xml to filePaths.
		for _, file := range input.Files {
			if !file.IsBinary && filepath.Base(file.Path) == "histograms.xml" {
				filePaths = append(filePaths, file.Path)
			}
		}
		// We need this outside the if statement since it will be used in getDiffsPerFile later.
		*patchPath = input.Patch
		// Only get previous files if .xml files were modified.
		if len(filePaths) != 0 {
			// Set up the temporary directory where we'll put previous files and apply the patch on them.
			// The temporary directory should be cleaned up before exiting.
			tempDir, err := ioutil.TempDir(*inputDir, "get-previous-file")
			if err != nil {
				log.Panicf("Failed to setup temporary directory: %v", err)
			}
			defer func() {
				if err = os.RemoveAll(tempDir); err != nil {
					log.Panicf("Failed to clean up temporary directory %q: %v", tempDir, err)
				}
			}()
			log.Printf("Created temporary directory %q.", tempDir)
			*prevDir = filepath.Join(tempDir, *inputDir)
			// Previous files will be put into prevDir.
			getPreviousFiles(filePaths, *inputDir, *prevDir, *patchPath)
		}
		filesChanged, err = getDiffsPerFile(filePaths, filepath.Join(*inputDir, *patchPath))
		if err != nil {
			log.Panicf("Failed to get diffs per file: %v", err)
		}
	}
	singletonEnums := getSingleElementEnums(filepath.Join(*inputDir, *enumsPath))

	results := &tricium.Data_Results{}
	for _, filePath := range filePaths {
		inputPath := filepath.Join(*inputDir, filePath)
		f := openFileOrDie(inputPath)
		defer closeFileOrDie(f)
		if filepath.Ext(filePath) == ".xml" {
			results.Comments = append(results.Comments, analyzeHistogramFile(f, filePath, *inputDir, *prevDir, filesChanged, singletonEnums)...)
		} else if filepath.Ext(filePath) == ".json" {
			results.Comments = append(results.Comments, analyzeFieldTrialTestingConfig(f, filePath)...)
		}
	}

	// Write Tricium RESULTS data.
	path, err := tricium.WriteDataType(*outputDir, results)
	if err != nil {
		log.Panicf("Failed to write RESULTS data: %v. Did you specify an output directory with -output?", err)
	}
	log.Printf("Wrote RESULTS data to path %q.", path)
}

// getDiffsPerFile gets the added and removed line numbers for a particular file.
func getDiffsPerFile(filePaths []string, patchPath string) (*diffsPerFile, error) {
	patch, err := ioutil.ReadFile(patchPath)
	if err != nil {
		return &diffsPerFile{}, err
	}
	diff, err := diffparser.Parse(string(patch))
	if err != nil {
		return &diffsPerFile{}, err
	}
	diffInfo := &diffsPerFile{
		addedLines:   map[string][]int{},
		removedLines: map[string][]int{},
	}
	fileSet := stringset.NewFromSlice(filePaths...)
	for _, diffFile := range diff.Files {
		if diffFile.Mode == diffparser.DELETED || !fileSet.Has(diffFile.NewName) {
			continue
		}
		for _, hunk := range diffFile.Hunks {
			for _, line := range hunk.WholeRange.Lines {
				if line.Mode == diffparser.ADDED {
					diffInfo.addedLines[diffFile.NewName] = append(diffInfo.addedLines[diffFile.NewName], line.Number)
				} else if line.Mode == diffparser.REMOVED {
					diffInfo.removedLines[diffFile.NewName] = append(diffInfo.removedLines[diffFile.NewName], line.Number)
				}
			}
		}
	}
	return diffInfo, nil
}

// getPreviousFiles unapplies a patch in copied files.
// It copies filePaths files from inputDir into prevDir,
// then applies the patch file at patchPath reversed on the copied files.
// patchPath must be relative to inputDir.
func getPreviousFiles(filePaths []string, inputDir, prevDir, patchPath string) {
	filesToCopy := append(filePaths, patchPath)
	for _, filePath := range filesToCopy {
		tempPath := filepath.Join(prevDir, filePath)
		// Note: Must use filepath.Dir rather than path.Dir to be compatible with Windows.
		if err := os.MkdirAll(filepath.Dir(tempPath), os.ModePerm); err != nil {
			log.Panicf("Failed to create dirs for file: %v", err)
		}
		copyFile(filepath.Join(inputDir, filePath), tempPath)
	}
	// Only apply patch if patch is not empty.
	fi, err := os.Stat(filepath.Join(inputDir, patchPath))
	if err != nil {
		log.Panicf("Failed to get file info for patch %s: %v", patchPath, err)
	}
	if fi.Size() > 0 {
		cmds := []*exec.Cmd{exec.Command("git", "init")}
		for _, filePath := range filePaths {
			cmds = append(cmds, exec.Command("git", "apply", "-p1", "--reverse", "--include="+filePath, patchPath))
		}
		for _, c := range cmds {
			var stderr bytes.Buffer
			c.Dir = prevDir
			c.Stderr = &stderr
			log.Printf("Running cmd: %s", c.Args)
			if err := c.Run(); err != nil {
				log.Panicf("Failed to run command %s\n%v\nStderr: %s", c.Args, err, stderr.String())
			}
		}
	}
}

func copyFile(sourceFile, destFile string) {
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		log.Panicf("Failed to read file %s while copying file %s to %s: %v", sourceFile, sourceFile, destFile, err)
	}
	if err = ioutil.WriteFile(destFile, input, 0644); err != nil {
		log.Panicf("Failed to write file %s while copying file %s to %s: %v", destFile, sourceFile, destFile, err)
	}
}

func getSingleElementEnums(inputPath string) stringset.Set {
	singletonEnums := make(stringset.Set)
	f := openFileOrDie(inputPath)
	defer closeFileOrDie(f)
	enumBytes, err := ioutil.ReadAll(f)
	if err != nil {
		log.Panicf("Failed to read enums into buffer: %v. Did you specify the enums file correctly with -enums?", err)
	}
	var allEnums enumFile
	if err := xml.Unmarshal(enumBytes, &allEnums); err != nil {
		log.Panicf("Failed to unmarshal enums")
	}
	for _, enum := range allEnums.Enums.EnumList {
		if len(enum.Elements) == 1 {
			singletonEnums.Add(enum.Name)
		}
	}
	return singletonEnums
}

func openFileOrDie(path string) *os.File {
	f, err := os.Open(path)
	if err != nil {
		log.Panicf("Failed to open file: %v, path: %s", err, path)
	}
	return f
}

func closeFileOrDie(f *os.File) {
	if err := f.Close(); err != nil {
		log.Panicf("Failed to close file: %v", err)
	}
}
