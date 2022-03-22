// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristic

import (
	"context"
	"fmt"
	gfim "infra/appengine/gofindit/model"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/common/logging"
)

// ScoringCriteria represents how we score in the heuristic analysis.
type ScoringCriteria struct {
	// The score if the suspect touched the same file in the failure log.
	TouchedSameFile int
	// The score if the suspect touched a related file to a file in the failure log.
	TouchedRelatedFile int
	// The score if the suspect touched the same file and the same line as in the failure log.
	TouchedSameLine int
}

// AnalyzeChangeLogs analyzes the changelogs based on the failure signals.
// Returns a dictionary that maps the commits and the result found.
func AnalyzeChangeLogs(c context.Context, signal *gfim.CompileFailureSignal, changelogs []*gfim.ChangeLog) (*gfim.HeuristicAnalysisResult, error) {
	result := &gfim.HeuristicAnalysisResult{}
	for _, changelog := range changelogs {
		justification, err := AnalyzeOneChangeLog(c, signal, changelog)
		commit := changelog.Commit
		if err != nil {
			logging.Errorf(c, "Error analyze change log for commit %s. Error: %w", commit, err)
			continue
		}
		reviewUrl, err := changelog.GetReviewUrl()
		if err != nil {
			logging.Errorf(c, "Error getting reviewUrl for commit: %s. Error: %w", commit, err)
			continue
		}

		// We only care about those relevant CLs
		if justification.GetScore() > 0 {
			result.AddItem(commit, reviewUrl, justification)
		}
	}
	result.Sort()
	return result, nil
}

// AnalyzeOneChangeLog analyzes one changelog(revision) and returns the
// justification of how likely that changelog is the culprit.
func AnalyzeOneChangeLog(c context.Context, signal *gfim.CompileFailureSignal, changelog *gfim.ChangeLog) (*gfim.SuspectJustification, error) {
	// TODO (crbug.com/1295566): check DEPs file as well, if the CL touches DEPs.
	// This is a nice-to-have feature, and is an edge case.
	justification := &gfim.SuspectJustification{}
	author := changelog.Author.Email
	for _, email := range getNonBlamableEmail() {
		if email == author {
			return &gfim.SuspectJustification{IsNonBlamable: true}, nil
		}
	}

	// Check files and line number extracted from output
	criteria := &ScoringCriteria{
		TouchedSameFile:    5,
		TouchedRelatedFile: 2,
		TouchedSameLine:    10,
	}
	for file, lines := range signal.Files {
		for _, diff := range changelog.ChangeLogDiffs {
			e := updateJustification(c, justification, file, lines, diff, criteria)
			if e != nil {
				return nil, e
			}
		}
	}

	// Check for dependency.
	criteria = &ScoringCriteria{
		TouchedSameFile:    3,
		TouchedRelatedFile: 1,
	}
	for _, edge := range signal.Edges {
		for _, dependency := range edge.Dependencies {
			for _, diff := range changelog.ChangeLogDiffs {
				e := updateJustification(c, justification, dependency, []int{}, diff, criteria)
				if e != nil {
					return nil, e
				}
			}
		}
	}

	justification.Sort()
	return justification, nil
}

func updateJustification(c context.Context, justification *gfim.SuspectJustification, fileInLog string, lines []int, diff gfim.ChangeLogDiff, criteria *ScoringCriteria) error {
	// TODO (crbug.com/1295566): In case of MODIFY, also query Gitiles for the
	// changed region and compared with lines. If they intersect, increase the score.
	// This may lead to a better score indicator.

	// Get the relevant file paths from CLs
	relevantFilePaths := []string{}
	switch diff.Type {
	case gfim.ChangeType_ADD, gfim.ChangeType_COPY, gfim.ChangeType_MODIFY:
		relevantFilePaths = append(relevantFilePaths, diff.NewPath)
	case gfim.ChangeType_RENAME:
		relevantFilePaths = append(relevantFilePaths, diff.NewPath, diff.OldPath)
	case gfim.ChangeType_DELETE:
		relevantFilePaths = append(relevantFilePaths, diff.OldPath)
	default:
		return fmt.Errorf("Unsupported diff type %s", diff.Type)
	}
	for _, filePath := range relevantFilePaths {
		score := 0
		reason := ""
		if IsSameFile(filePath, fileInLog) {
			score = criteria.TouchedSameFile
			reason = getReasonSameFile(filePath, diff.Type)
		} else if IsRelated(filePath, fileInLog) {
			score = criteria.TouchedRelatedFile
			reason = getReasonRelatedFile(filePath, diff.Type, fileInLog)
		}
		if score > 0 {
			justification.AddItem(score, filePath, reason)
		}
	}
	return nil
}

func getReasonSameFile(filePath string, changeType gfim.ChangeType) string {
	m := getChangeTypeActionMap()
	action := m[string(changeType)]
	return fmt.Sprintf("The file \"%s\" was %s and it was in the failure log.", filePath, action)
}

func getReasonRelatedFile(filePath string, changeType gfim.ChangeType, relatedFile string) string {
	m := getChangeTypeActionMap()
	action := m[string(changeType)]
	return fmt.Sprintf("The file \"%s\" was %s. It was related to the file %s which was in the failure log.", filePath, action, relatedFile)
}

func getChangeTypeActionMap() map[string]string {
	return map[string]string{
		gfim.ChangeType_ADD:    "added",
		gfim.ChangeType_COPY:   "copied",
		gfim.ChangeType_RENAME: "renamed",
		gfim.ChangeType_MODIFY: "modified",
		gfim.ChangeType_DELETE: "deleted",
	}
}

// IsSameFile makes the best effort in guessing if the file in the failure log
// is the same as the file in the changelog or not.
// Args:
// fullFilePath: Full path of a file committed to git repo.
// fileInLog: File path appearing in a failure log. It may or may not be a full path.
// Example:
// ("chrome/test/base/chrome_process_util.h", "base/chrome_process_util.h") -> True
// ("a/b/x.cc", "a/b/x.cc") -> True
// ("c/x.cc", "a/b/c/x.cc") -> False
func IsSameFile(fullFilePath string, fileInLog string) bool {
	// In some cases, fileInLog is prepended with "src/", we want a relative path to src/
	fileInLog = strings.TrimPrefix(fileInLog, "src/")
	if fileInLog == fullFilePath {
		return true
	}
	return strings.HasSuffix(fullFilePath, fmt.Sprintf("/%s", fileInLog))
}

// IsRelated checks if 2 files are related.
// Example:
// file.h <-> file_impl.cc
// x.h <-> x.cc
func IsRelated(fullFilePath string, fileInLog string) bool {
	filePathExt := strings.TrimPrefix(filepath.Ext(fullFilePath), ".")
	fileInLogExt := strings.TrimPrefix(filepath.Ext(fileInLog), ".")
	if !AreRelelatedExtensions(filePathExt, fileInLogExt) {
		return false
	}

	if strings.HasSuffix(fileInLog, ".o") || strings.HasSuffix(fileInLog, ".obj") {
		fileInLog = NormalizeObjectFilePath(fileInLog)
	}

	if IsSameFile(StripExtensionAndCommonSuffix(fullFilePath), StripExtensionAndCommonSuffix(fileInLog)) {
		return true
	}

	return false
}

// Strips extension and common suffixes from file name to guess relation.
// Examples:
//file_impl.cc, file_unittest.cc, file_impl_mac.h -> file
func StripExtensionAndCommonSuffix(filePath string) string {
	dir := filepath.Dir(filePath)
	name := filepath.Base(filePath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	commonSuffixes := []string{
		"impl",
		"browser_tests", // Those test suffixes are here for completeness, in compile analysis we will not use them
		"browser_test",
		"browsertest",
		"browsertests",
		"unittests",
		"unittest",
		"tests",
		"test",
		"gcc",
		"msvc",
		"arm",
		"arm64",
		"mips",
		"portable",
		"x86",
		"android",
		"ios",
		"linux",
		"mac",
		"ozone",
		"posix",
		"win",
		"aura",
		"x",
		"x11",
	}
	for true {
		found := false
		for _, suffix := range commonSuffixes {
			suffix = "_" + suffix
			if strings.HasSuffix(name, suffix) {
				found = true
				name = strings.TrimSuffix(name, suffix)
			}
		}
		if !found {
			break
		}
	}
	if dir == "." {
		return name
	}
	return fmt.Sprintf("%s/%s", dir, name)
}

// NormalizeObjectFilePath normalizes the file path to an c/c++ object file.
// During compile, a/b/c/file.cc in TARGET will be compiled into object file
// obj/a/b/c/TARGET.file.o, thus 'obj/' and TARGET need to be removed from path.
func NormalizeObjectFilePath(filePath string) string {
	if !(strings.HasSuffix(filePath, ".o") || strings.HasSuffix(filePath, ".obj")) {
		return filePath
	}
	filePath = strings.TrimPrefix(filePath, "obj/")
	dir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	parts := strings.Split(fileName, ".")
	if len(parts) == 3 {
		// Special cases for file.cc.obj and similar cases
		if parts[1] != "c" && parts[1] != "cc" && parts[1] != "cpp" && parts[1] != "m" && parts[1] != "mm" {
			fileName = fmt.Sprintf("%s.%s", parts[1], parts[2])
		}
	} else if len(parts) > 3 {
		fileName = strings.Join(parts[1:], ".")
	}
	if dir == "." {
		return fileName
	}
	return fmt.Sprintf("%s/%s", dir, fileName)
}

// AreRelelatedExtensions checks if 2 extensions are related
func AreRelelatedExtensions(ext1 string, ext2 string) bool {
	relations := [][]string{
		{"h", "hh", "c", "cc", "cpp", "m", "mm", "o", "obj"},
		{"py", "pyc"},
		{"gyp", "gypi"},
	}
	for _, group := range relations {
		found1 := false
		found2 := false
		for _, ext := range group {
			if ext == ext1 {
				found1 = true
			}
			if ext == ext2 {
				found2 = true
			}
		}
		if found1 && found2 {
			return true
		}
	}
	return false
}

// getNonBlamableEmail returns emails whose changes should never be flagged as culprits.
func getNonBlamableEmail() []string {
	return []string{"chrome-release-bot@chromium.org"}
}
