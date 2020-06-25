// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tricium "infra/tricium/api/v1"
)

const (
	category = "HttpsCheck"
)

var (
	fileAllowlist = []string{".md"}
	httpRegexp    = regexp.MustCompile(`http:\/\/[^\s]*`)
)

func main() {
	inputDir := flag.String("input", "", "Path to root of Tricium input")
	outputDir := flag.String("output", "", "Path to root of Tricium output")
	flag.Parse()
	if flag.NArg() != 0 {
		log.Panicf("Unexpected argument.")
	}

	// Read Tricium input FILES data.
	input := &tricium.Data_Files{}
	if err := tricium.ReadDataType(*inputDir, input); err != nil {
		log.Panicf("Failed to read FILES data: %v", err)
	}

	// Create RESULTS data.
	results := &tricium.Data_Results{}
	for _, file := range input.Files {
		if file.IsBinary {
			log.Printf("Skipping binary file %q.", file.Path)
			continue
		}
		if !isAllowed(file.Path) {
			log.Printf("Skipping file: %q.", file.Path)
			continue
		}
		checkHTTPS(filepath.Join(*inputDir, file.Path), results)
	}

	// Write Tricium RESULTS data.
	path, err := tricium.WriteDataType(*outputDir, results)
	if err != nil {
		log.Panicf("Failed to write RESULTS data: %v", err)
	}
	log.Printf("Wrote RESULTS data to %q.", path)
}

func isAllowed(path string) bool {
	for _, ext := range fileAllowlist {
		if ext == filepath.Ext(path) || ext == filepath.Base(path) {
			return true
		}
	}
	return false
}

func checkHTTPS(path string, results *tricium.Data_Results) {
	file, err := os.Open(path)
	if err != nil {
		log.Panicf("Failed to open file: %v, path: %s", err, path)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Panicf("Failed to close file: %v, path: %s", err, path)
		}
	}()

	lineNum := int32(1)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for http:// URLs.
		httpIdx := httpRegexp.FindAllStringIndex(line, -1)

		if httpIdx != nil {
			for _, idx := range httpIdx {
				startIdx := int32(idx[0])
				endIdx := int32(idx[1])
				url := line[startIdx:endIdx]
				// Ensure each match isn't a go/ or g/ link.
				if !strings.HasPrefix(url, "http://go/") && !strings.HasPrefix(url, "http://g/") {
					results.Comments = append(results.Comments, httpURLComment(path, lineNum, startIdx, endIdx))
				}
			}
		}
		lineNum++
	}
}

func httpURLComment(path string, lineNum int32, startIdx int32, endIdx int32) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  fmt.Sprintf("%s/%s", category, "Warning"),
		Message:   "Nit: Replace http:// URLs with https://",
		Path:      path,
		StartLine: lineNum,
		EndLine:   lineNum,
		StartChar: startIdx,
		EndChar:   endIdx,
	}
}
