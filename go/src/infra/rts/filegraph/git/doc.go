// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package git implements derivation of a file graph from git log and optionally
// from the file structure.
//
// Change-log-based distance
//
// This distance is derived from the probability that two files appeared in the
// same commit. The core idea is that relevant files tend to be modified
// together.
//
//  Distance(x, y) = -log(P(x is relevant to y))
//  P(x is relevant to y) := sum(1/(len(c.files)-1) for c in y.commits if x in c.files) / len(y.commits)
//
// or in English,
//  - the distance from x to y is -log of the probability that x is relevant to y
//  - x is relevant to y if x is likely to appear in a commit that touches y,
//    where the commit is chosen randomly and independently.
//
// There are more nuances to this formula, e.g. file removals are not counted
// towards len(commit.files), and commits with len(files) = 1 or
// len(files) > limit are ignored. File renames are also taken care of.
//
// Note that this formula penalizes large commits. The more files are modified
// in a commit, the weaker is the strength of its signal.
//
// This graph defines distance only between files, and not directories.
//
// File-structure-based distance
//
// This distance is derived from the file structure. It is the number of edges
// between two files in the *file tree*, i.e. the number of hops one has to make
// to navigate from one file to another in the file tree. For example, given the
// following file stucture:
//
//   //
//   ├── foo/
//   │   ├── bar.cc
//   │   └── baz.cc
//   └── qux.cc
//
// The distance between //foo/bar.cc and //foo/baz.cc is 2 because they have a
// common parent, and the distance between //foo/bar.cc and and //qux.cc is 3
// because the path goes through the root.
//
// The file-structured-based distance can compensate for the weakness of the
// change-log-based distance.
//
// This distance formula is disabled by default, and can be enabled in
// EdgeReader.
package git
