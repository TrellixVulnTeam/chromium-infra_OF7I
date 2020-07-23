// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

// Repo represents a checked-out git repo on disk.
//
// Use Shell to manipulate the repo in your plugin's ApplyFix method.
type Repo interface {
	// ConfigRoot returns the path to the config file 'root'.
	//
	// This would be the directory containing `main.star`, if the repo has one,
	// otherwise is identical to GeneratedConfigRoot.
	//
	// TODO: Have a less-heuristic way to find this path in the repo.
	//
	// This an 'absolute-style' path (see Shell).
	ConfigRoot() string

	// GeneratedConfigRoot returns the path to the generated config files (i.e.
	// the ones seen by the luci-config service).
	//
	// This an 'absolute-style' path (see Shell).
	GeneratedConfigRoot() string

	// Project returns the LUCI Project associated with this repo.
	//
	// Files retrieved with ConfigFiles() will be based on the checked-out data.
	Project() Project

	// Shell returns a new shell object for this repo with its current working
	// directory set to ConfigRoot().
	Shell() Shell
}
