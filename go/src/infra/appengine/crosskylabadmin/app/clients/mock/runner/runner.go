// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This program's only mission in life is to invoke the command 'mockgen'
// and to produce the file swarming.go in the parent directory.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("runner: current working directory does not exist: %s\n", err)
	}
	if _, err := os.Stat(filepath.Join("runner", "runner.go")); err != nil {
		log.Fatalf("runner: runner.go must be run from its parent directory\n")
	}
	if err := os.Chdir(".."); err != nil {
		log.Fatalf("runner: failed to chdir to parent directory: %s\n", err)
	}
	parent, err := os.Getwd()
	if err != nil {
		log.Fatalf("runner: successfully chdir'd to parent dir but then failed to getcwd: %s\n", err)
	}
	if err := os.Chdir(cwd); err != nil {
		log.Fatalf("runner: failed to return to original directory %s\n", err)
	}
	sourceFile := filepath.Join(parent, "swarming.go")
	if _, err := os.Stat(sourceFile); err != nil {
		log.Fatalf("runner: source program %s does not exist: %s\n", sourceFile, err)
	}
	err = exec.Command("mockgen", "-source", sourceFile, "-destination", "swarming.go", "-package", "mock", "SwarmingClient").Run()
	if err != nil {
		log.Fatalf("runner: subprocess failed unexpectedly: %s\n", err)
	}
	// alter the second line of the preamble so as not to include absolute paths in
	// generated file swarming.go
	bytes, readFileErr := ioutil.ReadFile("swarming.go")
	// the file will still contain an absolute path, therefore delete it unconditionally
	os.Remove("swarming.go") // ignore errors
	if readFileErr != nil {
		log.Fatalf("failed to read contents of file")
	}
	contents := string(bytes)
	lines := strings.Split(contents, "\n")
	if len(lines) < 2 {
		log.Fatalf("swarming.go has too few lines")
	}
	lines[1] = "// Source: swarming.go (invoked by runner.go)"

	newContents := fmt.Sprintf("%s\n", strings.Join(lines, "\n"))

	err = ioutil.WriteFile("swarming.go", []byte(newContents), 0o644)
	if err != nil {
		log.Fatalf("failed to write file")
	}
}
