// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package util contains utility functions
package util

import "strings"

/*
 	NormalizeFilePath returns the normalized the file path.
	Strips leading "/" or "\"
	Converts "\\" and "//" to "/"
	Resolves ".." and "." from the file path
	e.g.
	//BUILD.gn  -> BUILD.gn
	../a/b/c.cc -> a/b/c.cc
	a/b/./c.cc  -> a/b/c.cc
*/
func NormalizeFilePath(fp string) string {
	fp = strings.TrimLeft(fp, "\\/")
	fp = strings.ReplaceAll(fp, "\\", "/")
	fp = strings.ReplaceAll(fp, "//", "/")

	// path.Clean cannot handle the case like ../c.cc, so
	// we need to do it manually
	parts := strings.Split(fp, "/")
	filteredParts := []string{}
	for _, part := range parts {
		if part == "." {
			continue
		} else if part == ".." {
			if len(filteredParts) > 0 {
				filteredParts = filteredParts[:len(filteredParts)-1]
			}
		} else {
			filteredParts = append(filteredParts, part)
		}
	}
	return strings.Join(filteredParts, "/")
}
