// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package main implements the Git File Isolator analyzer.
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"infra/tricium/api/v1"
)

func main() {
	inputDir := flag.String("input", "", "Path to root of Tricium input")
	outputDir := flag.String("output", "", "Path to root of Tricium output")
	flag.Parse()

	// Read Tricium input GIT_FILE_DETAILS data.
	input := &tricium.Data_GitFileDetails{}
	if err := tricium.ReadDataType(*inputDir, input); err != nil {
		log.Fatalf("Failed to read GIT_FILE_DETAILS data: %v", err)
	}
	log.Printf("Read GIT_FILE_DETAILS data: %+v", input)

	// Set up tmp dir.
	dir, err := ioutil.TempDir("", "git-file-isolator")
	if err != nil {
		log.Fatalf("Failed to setup temporary directory: %v", err)
	}

	// Clean up.
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Fatalf("Failed to clean up temporary directory, dir: %s, %v", dir, err)
		}
	}()
	log.Printf("Created temporary directory: %s", dir)

	// Check out files from the given git ref.
	cmds := []*exec.Cmd{
		exec.Command("git", "init"),
		exec.Command("git", "fetch", "--depth=1", "--no-tags",
			"--no-recurse-submodules", input.Repository, input.Ref),
		exec.Command("git", "checkout", "FETCH_HEAD", "--"),
	}

	// Explicitly add the list of files to the command line to checkout
	// to speed things up.
	// NB! The max length for a command line supported by the OS may be
	// exceeded; the max length for command line on POSIX can be inspected
	// with `getconf ARG_MAX`.
	for _, file := range input.Files {
		cmds[2].Args = append(cmds[2].Args, file.Path)
	}
	for _, c := range cmds {
		c.Dir = dir
		log.Printf("Running cmd: %s", c.Args)
		if err := c.Run(); err != nil {
			log.Fatalf("Failed to run command: %v, cmd: %s", err, c.Args)
		}
	}

	// Copy files to output directory for isolation.
	// Skip over any files which couldn't be copied and don't
	// include them in the output.
	output := &tricium.Data_Files{}
	for _, file := range input.Files {
		log.Printf("Preparing to copy file %q.", file.Path)
		src := filepath.Join(dir, file.Path)
		if fileInfo, err := os.Lstat(src); err != nil {
			log.Fatalf("Failed to stat file: %v", err)
		} else if !fileInfo.Mode().IsRegular() {
			log.Printf("Skipping file %q with mode %s.", src, fileInfo.Mode())
			continue
		}

		dest := filepath.Join(*outputDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
			log.Fatalf("Failed to create dirs for file: %v", err)
		}
		cmd := exec.Command("cp", src, dest)

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalf("Failed to read stderr: %v", err)
		}
		log.Printf("Running cmd: %s", cmd.Args)
		if err := cmd.Start(); err != nil {
			log.Fatalf("Failed to invoke command: %v", err)
		}
		slurp, _ := ioutil.ReadAll(stderr)
		if err := cmd.Wait(); err != nil {
			log.Fatalf("Command failed: %v, stderr: %s", err, slurp)
		}
		output.Files = append(output.Files, file)
	}

	// Write Tricium output FILES data.
	path, err := tricium.WriteDataType(*outputDir, output)
	if err != nil {
		log.Fatalf("Failed to write FILES data: %v", err)
	}
	log.Printf("Wrote FILES data, path: %q, value: %+v\n", path, output)
}
