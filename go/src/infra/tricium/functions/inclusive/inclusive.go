// Copyright 2021The Chromium Authors. All rights reserved.
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
	category = "InclusiveLanguageCheck"
)

var (
	inclusiveRegexp = regexp.MustCompile(`\b((black|white)list|master|slave)\b`)
	replacements    = map[string]string{
		"blacklist": "blocklist",
		"whitelist": "allowlist",
		"master":    "main",
		"slave":     "replica",
	}
	commentText = map[string]string{
		"blacklist": "Nit: Please avoid 'blacklist'. Suggested replacements include 'blocklist' and 'denylilst'. Reach out to community@chromium.org if you have questions.",
		"whitelist": "Nit: Please avoid 'whitelist'. Suggested replacements include 'allowlist' and 'safelilst'. Reach out to community@chromium.org if you have questions.",
		"master":    "Nit: Please avoid 'master'. Suggested replacements include 'main' and 'primary' and 'producer'. Reach out to community@chromium.org if you have questions.",
		"slave":     "Nit: Please avoid 'slave'. Suggested replacements include 'replica' and 'secondary' and 'consumer'. Reach out to community@chromium.org if you have questions.",
	}
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
		checkInclusiveLanguage(filepath.Join(*inputDir, file.Path), results)
	}

	// Write Tricium RESULTS data.
	path, err := tricium.WriteDataType(*outputDir, results)
	if err != nil {
		log.Panicf("Failed to write RESULTS data: %v", err)
	}
	log.Printf("Wrote RESULTS data to %q.", path)
}

type match struct {
	start, end int32
}

func findMatches(s string) (ret []match) {
	inclusiveIdx := inclusiveRegexp.FindAllStringIndex(s, -1)
	if inclusiveIdx != nil {
		for _, idx := range inclusiveIdx {
			startIdx := int32(idx[0])
			endIdx := int32(idx[1])
			ret = append(ret, match{startIdx, endIdx})
		}
	}
	return ret
}

func checkInclusiveLanguage(path string, results *tricium.Data_Results) {
	for _, m := range findMatches(path) {
		results.Comments = append(results.Comments, inclusiveLanguageComment(path, path, 0, m.start, m.end))
	}

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

		// Check for non-inclusive terms.
		for _, m := range findMatches(line) {
			results.Comments = append(results.Comments, inclusiveLanguageComment(path, line, lineNum, m.start, m.end))
		}
		lineNum++
	}
}

func inclusiveLanguageReplacement(path string, line string, lineNum int32, startIdx int32, endIdx int32) *tricium.Data_Replacement {
	term := strings.ToLower(line[startIdx:endIdx])

	replacement, ok := replacements[term]
	if !ok {
		log.Printf("No replacement found for %q", term)
	}
	return &tricium.Data_Replacement{
		Path:        path,
		Replacement: replacement,
		StartLine:   lineNum,
		EndLine:     lineNum,
		StartChar:   startIdx,
		EndChar:     endIdx,
	}
}

func inclusiveLanguageComment(path string, line string, lineNum int32, startIdx int32, endIdx int32) *tricium.Data_Comment {
	term := strings.ToLower(line[startIdx:endIdx])

	msg, ok := commentText[term]
	if !ok {
		log.Printf("No comment text found for %q", term)
	}

	ret := &tricium.Data_Comment{
		Category:  fmt.Sprintf("%s/%s", category, "Warning"),
		Message:   msg,
		Path:      path,
		StartLine: lineNum,
		EndLine:   lineNum,
		StartChar: startIdx,
		EndChar:   endIdx,
	}
	ret.Suggestions = append(ret.Suggestions, &tricium.Data_Suggestion{
		Description:  msg,
		Replacements: []*tricium.Data_Replacement{inclusiveLanguageReplacement(path, line, lineNum, startIdx, endIdx)},
	})
	return ret
}
