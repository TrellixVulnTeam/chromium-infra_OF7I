// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import "regexp"

// chunkRe matches validly formed chunk IDs.
var ChunkRe = regexp.MustCompile(`^[0-9a-f]{1,32}$`)

// algorithmRe matches validly formed clustering algorithm names.
var AlgorithmRe = regexp.MustCompile(`^[0-9a-z\-.]{1,32}$`)
