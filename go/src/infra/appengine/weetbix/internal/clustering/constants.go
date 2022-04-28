// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import "regexp"

// ChunkRe matches validly formed chunk IDs.
var ChunkRe = regexp.MustCompile(`^[0-9a-f]{1,32}$`)

// AlgorithmRePattern is the regular expression pattern matching
// validly formed clustering algorithm names.
// The overarching requirement is [0-9a-z\-]{1,32}, which we
// sudivide into an algorithm name of up to 26 characters
// and an algorithm version number.
const AlgorithmRePattern = `[0-9a-z\-]{1,26}-v[1-9][0-9]{0,3}`

// AlgorithmRe matches validly formed clustering algorithm names.
var AlgorithmRe = regexp.MustCompile(`^` + AlgorithmRePattern + `$`)
