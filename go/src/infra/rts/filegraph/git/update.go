// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package git

import (
	"context"

	"go.chromium.org/luci/common/errors"
)

// UpdateOptions are options for Graph.Update().
type UpdateOptions struct {
	// Callback, if not nil, is called each time after each commit is processed
	// and Graph.Commit is updated.
	Callback func() error

	// MaxCommitSize is the maximum number of files touched by a commit.
	// Commits that exceed this limit are ignored.
	// The rationale is that large commits provide a weak signal of file
	// relatedness and are expensive to process, O(N^2).
	MaxCommitSize int
}

// Update updates the graph based on changes in a git repository.
// This is the only way to mutate the Graph.
// Applies all changes reachable from rev, but not from g.Commit, and updates
// g.Commit.
//
// If returns an error which wasn't returned by the callback, then it is
// possible that the graph is corrupted.
func (g *Graph) Update(ctx context.Context, repoDir, rev string, opt UpdateOptions) error {
	g.ensureInitialized()
	if rev == "" {
		return errors.New("rev is empty")
	}

	return readLog(ctx, repoDir, g.Commit, rev, func(c commit) error {
		if err := g.apply(c.Files, opt.MaxCommitSize); err != nil {
			return errors.Annotate(err, "failed to apply commit %s", c.Hash).Err()
		}

		// TODO(nodir): do not call the callback if we are in the middle of
		// processing a second parent, because it is not a safe stopping point,
		// because the graph already incorporated commits that are not reachable
		// by c.Hash. The graph must not be saved in this state.
		g.Commit = c.Hash
		if opt.Callback != nil {
			return opt.Callback()
		}
		return nil
	})
}

// apply applies the file changes to the graph.
func (g *Graph) apply(fileChanges []fileChange, maxFileCount int) error {
	files := make([]*node, 0, len(fileChanges))
	for _, fc := range fileChanges {
		switch {
		case fc.Status == 'R':
			// The file was renamed.
			oldFile := g.ensureNode("//" + fc.Path)
			newFile := g.ensureNode("//" + fc.Path2)
			oldFile.ensureAlias(newFile)
			newFile.ensureAlias(oldFile)
			files = append(files, newFile)

		case fc.Status == 'D':
			// Ignore this file.
			// If this file is re-added later, it is likely to be a revert, where we'd
			// record the relation.
			// And if the file is never coming back, then its relations do not matter.

		case fc.Path2 != "":
			return errors.Reason("unexpected non-empty path2 %q for file status %c", fc.Path2, fc.Status).Err()

		default:
			files = append(files, g.ensureNode("//"+fc.Path))
		}
	}

	// Create edges between each file pair.
	// This is O(FILES * (FILES + EDGES_PER_FILE)),
	// so skip the commit if there are too many files.
	switch {
	case len(files) <= 1:
		// Skip this commit. It provides no signal about file relatedness.
		return nil

	case maxFileCount != 0 && len(files) > maxFileCount:
		// Skip this commit - too large.

		// NOTE: it is important to exit *after* we've processed renames above.
		// Large commits are often file moves.
		// If we didn't process them, the filegraph would be missing large sets
		// of files.
		return nil
	}

	// For any file in |files|, compute the probability of picking
	// any other file.
	p := probability(probOne / int64(len(files)-1))

	fileSet := make(map[*node]struct{}, len(files))
	for _, f := range files {
		fileSet[f] = struct{}{}
	}
	for _, file := range files {
		file.probSumDenominator++

		updated := make(map[*node]struct{}, len(files)-1)
		// Increment the commit count in file's edges that point to other files.
		for i, e := range file.edges {
			if _, ok := fileSet[e.to]; ok {
				updated[e.to] = struct{}{}

				// Add the probability of this file being selected from this commit,
				// unless it is an alias edge.
				if e.probSum != 0 {
					file.edges[i].probSum += p
				}
			}
		}

		// Add the missing edges.
		for _, to := range files {
			if to != file {
				if _, ok := updated[to]; !ok {
					file.prepareToAppendEdges()
					file.edges = append(file.edges, edge{to: to, probSum: p})
				}
			}
		}
	}

	return nil
}
