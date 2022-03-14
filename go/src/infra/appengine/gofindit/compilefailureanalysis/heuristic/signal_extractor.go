// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristic

import (
	"context"
	"fmt"
	"infra/appengine/gofindit/model"
	gfim "infra/appengine/gofindit/model"
	"infra/appengine/gofindit/util"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	// Patterns for Python stack trace frames.
	PYTHON_STACK_TRACE_FRAME_PATTERN_1 = `File "(?P<file>.+\.py)", line (?P<line>[0-9]+), in (?P<function>.+)`
	PYTHON_STACK_TRACE_FRAME_PATTERN_2 = `(?P<function>[^\s]+) at (?P<file>.+\.py):(?P<line>[0-9]+)`
	// Match file path separator: "/", "//", "\", "\\".
	PATH_SEPARATOR_PATTERN = `(?:/{1,2}|\\{1,2})`

	// Match drive root directory on Windows, like "C:/" or "C:\\".
	WINDOWS_ROOT_PATTERN = `[a-zA-Z]:` + PATH_SEPARATOR_PATTERN

	// Match system root directory on Linux/Mac.
	UNIX_ROOT_PATTERN = `/+`

	// Match system/drive root on Linux/Mac/Windows.
	ROOT_DIR_PATTERN = "(?:" + WINDOWS_ROOT_PATTERN + "|" + UNIX_ROOT_PATTERN + ")"

	// Match file/directory names and also match ., ..
	FILE_NAME_PATTERN = `[\w\.-]+`
)

// ExtractSignals extracts necessary signals for heuristic analysis from logs
func ExtractSignals(c context.Context, compileLogs *gfim.CompileLogs) (*model.CompileFailureSignal, error) {
	if compileLogs.NinjaLog == nil && compileLogs.StdOutLog == "" {
		return nil, fmt.Errorf("Unable to extract signals from empty logs.")
	}
	// Prioritise extracting signals from ninja logs instead of stdout logs
	if compileLogs.NinjaLog != nil {
		return ExtractSignalsFromNinjaLog(c, compileLogs.NinjaLog)
	}
	return ExtractSignalsFromStdoutLog(c, compileLogs.StdOutLog)
}

// ExtractSignalsFromNinjaLog extracts necessary signals for heuristic analysis from ninja log
func ExtractSignalsFromNinjaLog(c context.Context, ninjaLog *model.NinjaLog) (*model.CompileFailureSignal, error) {
	signal := &model.CompileFailureSignal{}
	for _, failure := range ninjaLog.Failures {
		edge := &model.CompileFailureEdge{
			Rule:         failure.Rule,
			OutputNodes:  failure.OutputNodes,
			Dependencies: normalizeDependencies(failure.Dependencies),
		}
		signal.Edges = append(signal.Edges, edge)
		signal.Nodes = append(signal.Nodes, failure.OutputNodes...)
		e := extractFiles(signal, failure.Output)
		if e != nil {
			return nil, e
		}
	}
	return signal, nil
}

func extractFiles(signal *model.CompileFailureSignal, output string) error {
	pythonPatterns := []*regexp.Regexp{
		regexp.MustCompile(PYTHON_STACK_TRACE_FRAME_PATTERN_1),
		regexp.MustCompile(PYTHON_STACK_TRACE_FRAME_PATTERN_2),
	}
	filePathLinePattern := regexp.MustCompile(getFileLinePathPattern())

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// Do not extract the first line
		if i == 0 {
			continue
		}
		// Check if the line matches python pattern
		matchedPython := false
		for _, pythonPattern := range pythonPatterns {
			matches, err := util.MatchedNamedGroup(pythonPattern, line)
			if err == nil {
				pyLine, e := strconv.Atoi(matches["line"])
				if e != nil {
					return e
				}
				signal.AddLine(util.NormalizeFilePath(matches["file"]), pyLine)
				matchedPython = true
				continue
			}
		}
		if matchedPython {
			continue
		}
		// Non-python cases
		matches := filePathLinePattern.FindAllStringSubmatch(line, -1)
		if matches != nil {
			for _, match := range matches {
				if len(match) != 3 {
					return fmt.Errorf("Invalid line: %s", line)
				}
				// match[1] is file, match[2] is line number
				if match[2] == "" {
					signal.AddFilePath(util.NormalizeFilePath(match[1]))
				} else {
					lineInt, e := strconv.Atoi(match[2])
					if e != nil {
						return e
					}
					signal.AddLine(util.NormalizeFilePath(match[1]), lineInt)
				}
			}
		}
	}
	return nil
}

func normalizeDependencies(dependencies []string) []string {
	result := []string{}
	for _, dependency := range dependencies {
		result = append(result, util.NormalizeFilePath(dependency))
	}
	return result
}

// ExtractSignalsFromStdoutLog extracts necessary signals for heuristic analysis from stdout log
func ExtractSignalsFromStdoutLog(c context.Context, log string) (*model.CompileFailureSignal, error) {
	// TODO: Implement this
	return nil, nil
}

/*
	Match a full file path and line number.
	It could match files with or without line numbers like below:
		c:\\a\\b.txt:12
		c:\a\b.txt(123)
		c:\a\b.txt:[line 123]
		D:/a/b.txt
		/a/../b/./c.txt
		a/b/c.txt
		//BUILD.gn:246
*/
func getFileLinePathPattern() string {
	pattern := `(`
	pattern += ROOT_DIR_PATTERN + "?"                                    // System/Drive root directory.
	pattern += `(?:` + FILE_NAME_PATTERN + PATH_SEPARATOR_PATTERN + `)*` // Directories.
	pattern += FILE_NAME_PATTERN + `\.` + getFileExtensionPattern()
	pattern += `)`                           // File name and extension.
	pattern += `(?:(?:[\(:]|\[line )(\d+))?` // Line number might not be available.
	return pattern
}

// getFileExtensionPattern matches supported file extensions.
// Sort extension list to avoid non-full match like 'c' matching 'c' in 'cpp'.
func getFileExtensionPattern() string {
	extensions := getSupportedFileExtension()
	sort.Sort(sort.Reverse(sort.StringSlice(extensions)))
	return fmt.Sprintf("(?:%s)", strings.Join(extensions, "|"))
}

// getSupportedFileExtension get gile extensions to filter out files from log.
func getSupportedFileExtension() []string {
	return []string{
		"c",
		"cc",
		"cpp",
		"css",
		"exe",
		"gn",
		"gni",
		"gyp",
		"gypi",
		"h",
		"hh",
		"html",
		"idl",
		"isolate",
		"java",
		"js",
		"json",
		"m",
		"mm",
		"mojom",
		"nexe",
		"o",
		"obj",
		"py",
		"pyc",
		"rc",
		"sh",
		"sha1",
		"txt",
	}
}
