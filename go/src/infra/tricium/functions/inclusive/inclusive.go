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
	termsRegexp = regexp.MustCompile(`\b((black|white)list|master|slave)\b`)

	nocheckRegexp   = regexp.MustCompile(`\b\s*(nocheck)$`)
	pyCommentRegexp = regexp.MustCompile(`#.*$`)
	cCommentRegexp  = regexp.MustCompile(`//.*$`)
	javaDocRegexp   = regexp.MustCompile(`^\s*\*`)
	gitPathRegexp   = regexp.MustCompile(`(\+|/)master(/|\:)`)

	replacements = map[string]string{
		"blacklist": "blocklist",
		"whitelist": "allowlist",
		"master":    "main",
		"slave":     "replica",
	}
	commentText = map[string]string{
		"blacklist": "Nit: Please avoid 'blacklist'. Suggested replacements include 'blocklist' and 'denylist'. Reach out to community@chromium.org if you have questions.",
		"whitelist": "Nit: Please avoid 'whitelist'. Suggested replacements include 'allowlist' and 'safelist'. Reach out to community@chromium.org if you have questions.",
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
		checkInclusiveLanguage(filepath.Join(*inputDir, file.Path), file.Path, results)
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
	if nocheckRegexp.MatchString(s) {
		return ret
	}
	if javaDocRegexp.MatchString(s) {
		return ret
	}
	if cCommentRegexp.MatchString(s) {
		// Ignore everything after `//'.
		s = cCommentRegexp.ReplaceAllLiteralString(s, "")
	}
	if pyCommentRegexp.MatchString(s) {
		// Ignore everything after `#'.
		s = pyCommentRegexp.ReplaceAllLiteralString(s, "")
	}
	// Ignore git branch references
	if gitPathRegexp.MatchString(s) {
		// Simply removing "master" from the middle of a line will remove the
		// false positive, but will break any true positives that occur
		// on the line after it since the reported character positions will be wrong.
		// So, blank it out with whitespace instead:
		s = string(gitPathRegexp.ReplaceAllFunc([]byte(s), func(b []byte) []byte {
			return []byte(strings.Repeat(" ", len(string(b))))
		}))
	}
	matchIdx := termsRegexp.FindAllStringIndex(s, -1)
	if matchIdx != nil {
		for _, idx := range matchIdx {
			startIdx := int32(idx[0])
			endIdx := int32(idx[1])
			ret = append(ret, match{startIdx, endIdx})
		}
	}
	return ret
}

func checkInclusiveLanguage(abspath, relpath string, results *tricium.Data_Results) {
	for _, m := range findMatches(relpath) {
		results.Comments = append(results.Comments, inclusiveLanguageComment(relpath, relpath, 0, m.start, m.end))
	}

	file, err := os.Open(abspath)
	if err != nil {
		log.Panicf("Failed to open file: %v, path: %s", err, abspath)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Panicf("Failed to close file: %v, path: %s", err, abspath)
		}
	}()

	lineNum := int32(1)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for non-inclusive terms.
		for _, m := range findMatches(line) {
			results.Comments = append(results.Comments, inclusiveLanguageComment(relpath, line, lineNum, m.start, m.end))
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
