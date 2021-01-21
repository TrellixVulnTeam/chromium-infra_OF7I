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

	// Matches methods with get prefix. Captures return type and selector.
	methodImplementationRegexp = regexp.MustCompile(`[-+]\s\((.+)\)(get.*)\s\{`)

	// Matches delegate properties. Captures property specifiers.
	delegatePropertyRegexp = regexp.MustCompile(`@property\((.*)\)\s*id<.*>\s.*[dD]elegate;`)

	// Matches all pointer properties. Captures property specifiers.
	pointerPropertyRegexp = regexp.MustCompile(`@property(.*)(\*|id<.*>).*;`)
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
		if comments := checkSourceFile(*inputDir, file.Path); comments != nil {
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

// checkSourceFile performs various regex based checks and return comments for
// found violations. Current checks include:
// - flagging methods where "get" prefix was misused
// - flagging strong delegates
//
// dirPath is the path to the repo directory base path, and filePath
// is the relative path from the base to the file.
func checkSourceFile(base string, path string) []*tricium.Data_Comment {
	content, err := ioutil.ReadFile(filepath.Join(base, path))
	if err != nil {
		log.Panicf("Failed to read file: %v", err)
	}

	contentString := string(content)
	matches := methodImplementationRegexp.FindAllStringSubmatchIndex(contentString, -1)

	// Look for methods where "get" prefix was misused.
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

	// Look for strong delegate properties. In Objective-C the normal convention
	// is to have "weak" delegate properties. "strong" delegate properties can
	// cause retain cycles, which is quite error prone, because "strong" is
	// default.
	matches = delegatePropertyRegexp.FindAllStringSubmatchIndex(contentString, -1)
	for _, match := range matches {
		propertyStartIndex := match[0]
		propertyEndIndex := match[1]
		specifiers := contentString[match[2]:match[3]]
		if !strings.Contains(specifiers, "weak") {
			propertyStartLine := converter.getStartLine(propertyStartIndex)
			comment := foundStrongDelegate(path,
				propertyStartLine,
				converter.getEndLine(propertyEndIndex),
				converter.getEndChar(propertyStartLine))
			comments = append(comments, comment)
		}
	}

	// Look for properties that do not have any ownership specifier. By default,
	// properties have "strong" ownership, which can lead to dependency cycles.
	matches = pointerPropertyRegexp.FindAllStringSubmatchIndex(contentString, -1)
	for _, match := range matches {
		propertyStartIndex := match[0]
		propertyEndIndex := match[1]
		specifiers := contentString[match[2]:match[3]]
		if !strings.Contains(specifiers, "weak") &&
			!strings.Contains(specifiers, "strong") &&
			!strings.Contains(specifiers, "copy") &&
			!strings.Contains(specifiers, "assign") {
			propertyStartLine := converter.getStartLine(propertyStartIndex)
			comment := foundPropertyWithNoOwnershipSpecifier(path,
				propertyStartLine,
				converter.getEndLine(propertyEndIndex),
				converter.getEndChar(propertyStartLine))
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

func foundStrongDelegate(path string, startLine int, endLine int, functionEndChar int) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  fmt.Sprintf("ObjectiveCStyle/StrongDelegate"),
		Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
		Path:      path,
		StartLine: int32(startLine),
		EndLine:   int32(endLine),
		StartChar: 0, // always zero
		EndChar:   int32(functionEndChar),
	}
}

func foundPropertyWithNoOwnershipSpecifier(path string, startLine, endLine, functionEndChar int) *tricium.Data_Comment {
	return &tricium.Data_Comment{
		Category:  "ObjectiveCStyle/ExplicitOwnership",
		Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
		Path:      path,
		StartLine: int32(startLine),
		EndLine:   int32(endLine),
		StartChar: 0, // always zero
		EndChar:   int32(functionEndChar),
	}
}

// CharIndexToLineConverter converts character index obtained from regex to line
// number required by tricium. Line numbers for Tricium comments are 1-based
// because that's how it is in the Gerrit API, where line 0 represents a
// file-level comment, and line numbers starting with 1 are displayed next to
// actual lines.
type CharIndexToLineConverter interface {
	getStartLine(startIndex int) int
	getEndLine(endIndex int) int
	getEndChar(startLine int) int
}

func newCharIndexToLineConverter(content string) CharIndexToLineConverter {
	return charIndexToLineConverter{content, nil, nil}
}

func (converter charIndexToLineConverter) getStartLine(startIndex int) int {
	return sort.SearchInts(getLookupTable(converter), startIndex) + 2
}

func (converter charIndexToLineConverter) getEndLine(endIndex int) int {
	return sort.SearchInts(getLookupTable(converter), endIndex) + 1
}

func (converter charIndexToLineConverter) getEndChar(startLine int) int {
	return len(getLines(converter)[startLine-1])
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
