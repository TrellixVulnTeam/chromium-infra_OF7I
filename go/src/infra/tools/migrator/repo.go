// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

// Repo represents a checked-out git repo on disk.
//
// It contains configs of one or more LUCI projects.
type Repo interface {
	// Empty for now.
}
