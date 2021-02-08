// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package filegraph implements a directed weighted graph of files, where the
// weight of edge (x, y), called distance, represents how much y is affected
// by changes in x. Such graph provides distance between any two files, and can
// order files by distance from/to a given file.
package filegraph
