// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

// Compile Failure Signal represents signal extracted from compile failure log.
type CompileFailureSignal struct {
	Nodes []string
	Edges []*CompileFailureEdge
	// A map of {<file_path>:[lines]} represents failure positions in source file
	Files map[string][]int
}

// CompileFailureEdge represents a failed edge in ninja failure log
type CompileFailureEdge struct {
	Rule         string // Rule is like CXX, CC...
	OutputNodes  []string
	Dependencies []string
}

func (c *CompileFailureSignal) AddLine(filePath string, line int) {
	c.AddFilePath(filePath)
	for _, l := range c.Files[filePath] {
		if l == line {
			return
		}
	}
	c.Files[filePath] = append(c.Files[filePath], line)
}

func (c *CompileFailureSignal) AddFilePath(filePath string) {
	if c.Files == nil {
		c.Files = map[string][]int{}
	}
	_, exist := c.Files[filePath]
	if !exist {
		c.Files[filePath] = []int{}
	}
}
