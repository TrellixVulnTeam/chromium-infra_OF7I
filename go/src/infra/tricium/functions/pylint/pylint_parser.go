// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	tricium "infra/tricium/api/v1"
)

// Paths to the required resources relative to the executable directory.
const (
	pythonPath        = "python/bin/python3"
	pylintPath        = "pylint/bin/pylint"
	pylintPackagePath = "pylint/"
)

// pylintResult corresponds to each element of pylint's JSON output.
type pylintResult struct {
	Line   int32  `json:"line"`
	Column int32  `json:"column"`
	Path   string `json:"path"`
	// E.g. "warning" or "convention".
	Type string `json:"type"`
	// E.g. "unused-argument".
	Symbol  string `json:"symbol"`
	Message string `json:"message"`
}

func main() {
	if err := mainImpl(); err != nil {
		fmt.Fprintf(os.Stderr, "pylint_parser: %s\n", err)
		os.Exit(1)
	}
}

func mainImpl() error {
	inputDir := flag.String("input", "", "Path to root of Tricium input")
	outputDir := flag.String("output", "", "Path to root of Tricium output")
	disable := flag.String("disable", "", "Comma-separated list of checks "+
		"or categories of checks to disable.")
	enable := flag.String("enable", "", "Comma-separated checks "+
		"or categories of checks to enable. "+
		"The enable list overrides the disable list.")
	flag.Parse()
	if flag.NArg() != 0 {
		return errors.New("unexpected argument")
	}

	// Retrieve the path name for the executable that started the current process.
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	exPath := filepath.Dir(ex)
	log.Printf("Using executable path %q.", exPath)

	// Read Tricium input FILES data.
	input := &tricium.Data_Files{}
	if err = tricium.ReadDataType(*inputDir, input); err != nil {
		return fmt.Errorf("failed to read FILES data: %w", err)
	}
	log.Printf("Read FILES data.")

	// Filter the files to include only .py files.
	files, err := tricium.FilterFiles(input.Files, "*.py")
	if err != nil {
		return fmt.Errorf("failed to filter files: %w", err)
	}

	// Construct the command args and invoke Pylint on the given paths.
	cmdName := filepath.Join(exPath, pythonPath)
	absPylintPath := filepath.Join(exPath, pylintPath)
	absPylintPackagePath := filepath.Join(exPath, pylintPackagePath)
	if _, err := os.Stat(absPylintPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("pylint executable does not exist at %s", absPylintPath)
		}
		return fmt.Errorf("failed to check if pylint executable exists: %w", err)
	}
	cmdArgs := []string{
		absPylintPath,
		"--rcfile", filepath.Join(exPath, "pylintrc"),
		"--output-format", "json",
	}
	// With Pylint, the order of the disable and enable command line flags is
	// important; the later flags override previous flags. But for this
	// executable, the order is unimportant, the "enable" flag is always put
	// after "disable", so it always takes precedence.
	if *disable != "" {
		cmdArgs = append(cmdArgs, "--disable", *disable)
	}
	if *enable != "" {
		cmdArgs = append(cmdArgs, "--enable", *enable)
	}
	// In the output, we want relative paths from the repository root, which
	// will be the same as relative paths from the input directory root.
	for _, file := range files {
		cmdArgs = append(cmdArgs, file.Path)
	}
	cmd := exec.Command(cmdName, cmdArgs...)
	log.Printf("Command: %s", cmd.Args)

	stdout := &bytes.Buffer{}

	// Set PYTHONPATH for the command to run so that the bundled version of
	// pylint and its dependencies are used.
	env := os.Environ()
	env = append(env, fmt.Sprintf("PYTHONPATH=%s", absPylintPackagePath))
	cmd.Env = env
	cmd.Dir = *inputDir
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	// A non-zero exit status for Pylint doesn't mean that an error occurred,
	// it just means that warnings were found, so we can ignore the error as
	// long as it's an ExitError.
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			log.Printf("pylint produced non-zero exit code %d", exitErr.ExitCode())
		} else {
			return fmt.Errorf("error running pylint: %w", err)
		}
	}

	log.Printf("pylint output: %s", stdout.String())
	comments, err := parsePylintOutput(stdout.Bytes())
	if err != nil {
		return err
	}
	output := &tricium.Data_Results{}
	output.Comments = comments

	// Write Tricium RESULTS data.
	path, err := tricium.WriteDataType(*outputDir, output)
	if err != nil {
		return fmt.Errorf("failed to write RESULTS data: %w", err)
	}
	log.Printf("Wrote RESULTS data to path %q.", path)
	return nil
}

// parsePylintOutput reads populates results from pylint JSON output.
func parsePylintOutput(stdout []byte) ([]*tricium.Data_Comment, error) {
	var results []pylintResult
	if err := json.Unmarshal(stdout, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pylint output: %w", err)
	}
	var comments []*tricium.Data_Comment
	for _, r := range results {
		msg := r.Message
		if r.Symbol == "undefined-variable" {
			msg = (msg + ".\n" +
				"This check could give false positives when there are wildcard imports\n" +
				"(from module import *). It is recommended to avoid wildcard imports; see\n" +
				"https://www.python.org/dev/peps/pep-0008/#imports")
		}
		comments = append(comments, &tricium.Data_Comment{
			Path: r.Path,
			Message: fmt.Sprintf(
				"%s.\nTo disable, add: # pylint: disable=%s", msg, r.Symbol),
			Category:  fmt.Sprintf("Pylint/%s/%s", r.Type, r.Symbol),
			StartLine: r.Line,
			StartChar: r.Column,
		})
	}
	return comments, nil
}
