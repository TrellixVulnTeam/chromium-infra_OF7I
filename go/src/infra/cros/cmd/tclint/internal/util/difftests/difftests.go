// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package difftests provides utilities for writing tests that compare against
// golden output.
package difftests

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

// DiscoverTestCases discovers test cases for expectation tests.
//
// DiscoverTestCases assumes the following filesystem layout under `root`:
// - Input files, named ${root}/**/*.input, contain the input protobuf payload.
// - Golden files are named correspondingly ${root}/**/*.golden
//
// Golden files may be missing in some cases, and will be created by the
// UpdateGoldenIfRequested() method on TestCase.
func DiscoverTestCases(t *testing.T, root string) []TestCase {
	tcs := []TestCase{}
	for _, p := range discoverInputFiles(t, root) {
		dir, base := filepath.Split(p)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		tc := TestCase{
			Name:       filepath.Join(dir, name),
			inputFile:  p,
			goldenFile: filepath.Join(dir, fmt.Sprintf("%s%s", name, goldenFileExt)),
		}
		if _, err := os.Stat(tc.goldenFile); err == nil {
			tc.goldenFileFound = true
		}
		tcs = append(tcs, tc)
	}

	if len(tcs) == 0 {
		t.Fatalf("no input files found in %s", root)
	}
	return tcs
}

// TestCase provides methods load and save data for a diff test.
type TestCase struct {
	Name            string
	inputFile       string
	goldenFile      string
	goldenFileFound bool
}

// LoadInput loads the input payload for the test case into the provided
// protobuf message.
func (tc *TestCase) LoadInput(t *testing.T, outM proto.Message) {
	r, err := os.Open(tc.inputFile)
	if err != nil {
		t.Fatalf("load proto from %s: %s", tc.inputFile, err.Error())
	}
	if err := jsonpb.Unmarshal(r, outM); err != nil {
		t.Fatalf("load proto from %s: %s", tc.inputFile, err.Error())
	}
}

// LoadGolden loads the golden file for a test case.
//
// Failures to load golden file contents reported as non-fatal errors so that
// the golden file can be updated later.
//
// Callers should skip TestCases where golden file can not be loaded.
func (tc *TestCase) LoadGolden(t *testing.T) (contents []string, loaded bool) {
	data := []string{}
	var s []byte
	if !tc.goldenFileFound {
		t.Errorf("no golden file for input file %s", tc.inputFile)
		return nil, false
	}

	var err error
	if s, err = ioutil.ReadFile(tc.goldenFile); err != nil {
		t.Errorf("load golden file %s: %s", tc.goldenFile, err.Error())
		return nil, false
	}
	if err := json.Unmarshal(s, &data); err != nil {
		t.Errorf("load golden file %s: %s", tc.goldenFile, err.Error())
		return nil, false
	}
	return data, true
}

// Intentionally uses verbose flag name to avoid collision with predefined flags
// in the testing package.
var update = flag.Bool("update-lint-golden-files", false, "Update the golden files for lint diff tests")

// UpdateGoldenIfRequested updates the golden file for a test case.
//
// Golden file updates can be enabled from the command line by setting the
// -update-lint-golden-files flag.
func (tc *TestCase) UpdateGoldenIfRequested(t *testing.T, data []string) {
	if !*update {
		return
	}
	s, err := json.MarshalIndent(data, "", "")
	if err != nil {
		t.Fatalf("write golden file %s: %s", tc.goldenFile, err.Error())
	}
	if err := ioutil.WriteFile(tc.goldenFile, s, 0666); err != nil {
		t.Fatalf("write golden file %s: %s", tc.goldenFile, err.Error())
	}
	t.Logf("Updated golden file %s", tc.goldenFile)
}

func discoverInputFiles(t *testing.T, root string) []string {
	inputFiles := []string{}
	filepath.Walk(
		root,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				t.Fatalf("listInputFiles(%s) %s: %s", root, path, err)
			}
			if filepath.Ext(path) == inputFileExt {
				inputFiles = append(inputFiles, path)
			}
			return nil
		},
	)
	return inputFiles
}

const (
	inputFileExt  = ".input"
	goldenFileExt = ".golden"
)
