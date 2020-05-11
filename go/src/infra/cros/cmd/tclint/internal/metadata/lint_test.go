// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package metadata provides functions to lint Chrome OS integration test
// metadata.
package metadata_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"infra/cros/cmd/tclint/internal/metadata"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"
	metadataPB "go.chromium.org/chromiumos/config/go/api/test/metadata/v1"
)

// Intentionally uses verbose flag name to avoid collision with predefined flags
// in the testing package.
var update = flag.Bool("update-lint-golden-files", false, "Update the golden files for lint diff tests")

// Tests returned diagnostic messages by comparing against golden expectation
// files.
//
// Returned diagnostics are the public API for tclint tool. This test prevents
// unintended regressions in the messages. To avoid spurious failures due to
// changes in logic unrelated to the message creation, each test case should
// minimize the number of errors detected.
func TestErrorMessages(t *testing.T) {
	for _, tc := range discoverTestCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			var spec metadataPB.Specification
			loadSpecification(t, tc.inputFile, &spec)
			r := metadata.Lint(&spec)
			got := r.Display()
			want := loadGoldenFile(t, tc)
			if diff := pretty.Compare(want, got); diff != "" {
				t.Errorf("lint errors expectations mismatch, -want +got: \n%s", diff)

			}
			if *update {
				writeGoldenFile(t, tc.goldenFile, got)
			}
		})
	}
}

type testCase struct {
	name            string
	inputFile       string
	goldenFile      string
	goldenFileFound bool
}

const (
	testDataDir   = "testdata"
	inputFileExt  = ".input"
	goldenFileExt = ".golden"
)

func discoverTestCases(t *testing.T) []testCase {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("load test data: %s", err.Error())
	}
	dataDir := filepath.Join(wd, testDataDir)
	fs, err := ioutil.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("load test data from %s: %s", dataDir, err.Error())
	}

	tcs := []testCase{}
	for _, f := range fs {
		if filepath.Ext(f.Name()) != inputFileExt {
			continue
		}
		name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		tc := testCase{
			name:       name,
			inputFile:  filepath.Join(testDataDir, f.Name()),
			goldenFile: filepath.Join(testDataDir, fmt.Sprintf("%s%s", name, goldenFileExt)),
		}
		if _, err := os.Stat(tc.goldenFile); err == nil {
			tc.goldenFileFound = true
		} else {
			// Failures to find golden file is not considered fatal so that the
			// golden file can be added later.
			t.Errorf("no golden file for input file %s", tc.inputFile)
		}
		tcs = append(tcs, tc)
	}

	if len(tcs) == 0 {
		t.Fatalf("no input files found in %s", dataDir)
	}
	return tcs
}

func loadSpecification(t *testing.T, path string, m proto.Message) {
	r, err := os.Open(path)
	if err != nil {
		t.Fatalf("load proto from %s: %s", path, err.Error())
	}
	if err := jsonpb.Unmarshal(r, m); err != nil {
		t.Fatalf("load proto from %s: %s", path, err.Error())
	}
}

// Returns the loaded data, or empty list on errors.
//
// Failures to load golden file contents are not considered fatal so that the
// golden file can be updated later.
func loadGoldenFile(t *testing.T, tc testCase) []string {

	data := []string{}
	var s []byte
	if tc.goldenFileFound {
		var err error
		if s, err = ioutil.ReadFile(tc.goldenFile); err != nil {
			t.Errorf("load golden file %s: %s", tc.goldenFile, err.Error())
			return nil
		}
		if err := json.Unmarshal(s, &data); err != nil {
			t.Errorf("load golden file %s: %s", tc.goldenFile, err.Error())
			return nil
		}
	}
	return data
}

func writeGoldenFile(t *testing.T, f string, data []string) {
	s, err := json.MarshalIndent(data, "", "")
	if err != nil {
		t.Fatalf("write golden file %s: %s", f, err.Error())
	}
	if err := ioutil.WriteFile(f, s, 0666); err != nil {
		t.Fatalf("write golden file %s: %s", f, err.Error())
	}
	t.Logf("Updated golden file %s", f)
}
