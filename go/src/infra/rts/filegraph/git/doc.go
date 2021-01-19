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
//  Distance(x, y) = -log(P(y is relevant to x))
//  P(y is relevant to x) := sum(1/(len(c.files)-1) for c in x.commits if y in c.files) / len(x.commits)
//
// or in English, distance from x to y is -log of the probability that y
// appears in a commit that touches x and is chosen randomly and independently.
// There are more nuances to this formula, e.g. file removals are not counted
// towards len(commit.files), and commits with len(files) = 1 or
// len(files) > limit are ignored.
//
// Note that this formula penalizes large commits. The more files are modified
// in a commit, the weaker is the strength of its signal.
//
// This graph defines distance only between files, and not directories.
package git
