// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package osutils

import (
	"os"
	"path/filepath"
)

func Abs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	return abs
}

// Verify that the given relative path exists.
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Verify that a directory exists at the given relative path.
func DirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// Look for a relative path, ascending through parent directories.
//
// Args:
//  pathToFind: The relative path to look for.
//  startPath: The path to start the search from.  If |startPath| is a
//    directory, it will be included in the directories that are searched.
//  endPath: The path to stop searching.
//  testFunc: The function to use to verify the relative path.
func FindInPathParents(
	pathToFind string,
	startPath string,
	endPath string,
	testFunc func(string) bool) string {

	// Default parameter values.
	if endPath == "" {
		endPath = "/"
	}
	if testFunc == nil {
		testFunc = PathExists
	}
	currentPath := startPath
	for {
		// Test to see if path exists in this directory
		targetPath := filepath.Join(Abs(currentPath), pathToFind)
		if testFunc(targetPath) {
			return Abs(targetPath)
		}
		rel, _ := filepath.Rel(endPath, currentPath)
		if rel == "." || rel == "" {
			// Reached endPath.
			return ""
		}

		// Go up one directory.
		currentPath += "/.."
	}
}
