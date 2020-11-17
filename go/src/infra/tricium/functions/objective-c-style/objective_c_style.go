// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tricium "infra/tricium/api/v1"
)

const (
	category = "ObjectiveCStyle"
)

var (
	fileAllowlist = []string{".m", ".mm"}

	// Captures return type and selector.
	methodImplementationRegexp = regexp.MustCompile(`-\s\((.+)\)(get.*)\s\{`)
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
	log.Printf("Read FILES data.")

	// Create RESULTS data.
	output := &tricium.Data_Results{}
	for _, file := range input.Files {
		if file.IsBinary {
			log.Printf("Skipping binary file %q.", file.Path)
			continue
		}
		if !isAllowed(file.Path) {
			log.Printf("Skipping file: %q.", file.Path)
			continue
		}
		if comments := checkGetPrefix(filepath.Join(*inputDir, file.Path)); comments != nil {
			for _, comment := range comments {
				log.Printf("%s: %s", file.Path, comment.Category)
				output.Comments = append(output.Comments, comment)
			}
		}
	}

	// Write Tricium RESULTS data.
	path, err := tricium.WriteDataType(*outputDir, output)
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

func checkGetPrefix(path string) []*tricium.Data_Comment {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panicf("Failed to read file: %v", err)
	}

	contentString := string(content)
	matches := methodImplementationRegexp.FindAllStringSubmatchIndex(contentString, -1)

	var comments []*tricium.Data_Comment = nil
	converter := newCharIndexToLineConverter(contentString)
	for _, match := range matches {
		functionStartIndex := match[0]
		functionEndIndex := match[1]
		returnType := contentString[match[2]:match[3]]
		selector := contentString[match[4]:match[5]]

		if returnType != "void" || !strings.Contains(selector, ":") {
			functionStartLine := converter.getStartLine(functionStartIndex)
			comment := foundGetPrefix(path,
				functionStartLine,
				converter.getEndLine(functionEndIndex),
				converter.getEndChar(functionStartLine))
			comments = append(comments, comment)
		}
	}
	return comments
}

func foundGetPrefix(path string, startLine int, endLine int, functionEndChar int) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category: fmt.Sprintf("ObjectiveCStyle/Get"),
		Message: "The use of \"get\" is unnecessary, unless one or more values " +
			"are returned indirectly. See: " +
			"https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/CodingGuidelines/Articles/NamingMethods.html#:~:text=The%20use%20of%20%22get%22%20is%20unnecessary,%20unless%20one%20or%20more%20values%20are%20returned%20indirectly.",
		Path:      path,
		StartLine: int32(startLine),
		EndLine:   int32(endLine),
		StartChar: 0, // always zero
		EndChar:   int32(functionEndChar),
	}
}

// CharIndexToLineConverter converts character index obtained from regex to line
// number required by tricium.
type CharIndexToLineConverter interface {
	getStartLine(startIndex int) int
	getEndLine(endIndex int) int
	getEndChar(startLine int) int
}

func newCharIndexToLineConverter(content string) CharIndexToLineConverter {
	return charIndexToLineConverter{content, nil, nil}
}

func (converter charIndexToLineConverter) getStartLine(startIndex int) int {
	return sort.SearchInts(getLookupTable(converter), startIndex) + 1
}

func (converter charIndexToLineConverter) getEndLine(endIndex int) int {
	return sort.SearchInts(getLookupTable(converter), endIndex)
}

func (converter charIndexToLineConverter) getEndChar(startLine int) int {
	return len(getLines(converter)[startLine])
}

type charIndexToLineConverter struct {
	// Full content of the source file.
	content string
	// |content| string split by "\n". Lazily created.
	lines []string
	// Dynamic programming lookup table of the same length as |lines|. Each value
	// in the table represents number of characters in current and all previous
	// lines. Lazily created.
	lookupTable []int
}

func getLines(converter charIndexToLineConverter) []string {
	if converter.lines == nil {
		converter.lines = strings.Split(converter.content, "\n")
	}
	return converter.lines
}

func getLookupTable(converter charIndexToLineConverter) []int {
	if converter.lookupTable == nil {
		previousLinesLen := 0
		for _, line := range getLines(converter) {
			currentLineLen := len(line) + len("\n")
			converter.lookupTable = append(converter.lookupTable, previousLinesLen+currentLineLen)
			previousLinesLen += currentLineLen
		}
	}
	return converter.lookupTable
}
