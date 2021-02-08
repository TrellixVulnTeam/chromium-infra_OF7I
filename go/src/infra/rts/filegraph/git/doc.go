// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package git implements derivation of a file graph from git log.
//
// Distance
//
// The distance is derived from the probability that two files appeared in the
// same commit. The core idea is that relevant files tend to be
// modified together.
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
package git
