// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

import (
	"os"
)

// TieStderr is a special token for Shell.{Run,Retval,Stdout} to indicate that
// Stderr content should be redirected to Stdout.
const TieStderr = "2>&1"

// Shell is a basic interface for interacting with the Repo.
//
// Paths
//
// All `path` arguments to the Shell are either:
//   * start with '/' and are relative to the corresponding Repo's root.
//   * OR; are relative to the Shell's current working directory.
//
// It is not permitted to access a path outside the Repo's root.
//
// A just-created Shell starts with the 'cwd' at the ConfigDir.
//
// Errors
//
// All functions of Shell will panic under error conditions. This is consistent
// with the API of the ApplyFix plugin method.
type Shell interface {
	// Cd changes the current directory.
	//
	// Absolute paths (i.e. beginning with '/') are interpreted as relative to the
	// repo root.
	//
	// Attempting to Cd out of the top of the repo is an error.
	Cd(path string)

	// ModifyFile allows trivial modification of a file.
	//
	// This will call `modify` with the contents of the old file ("" if the file
	// didn't exist), and will write the returned string back to `path`, if the
	// returned string is different. This will create missing intermediate
	// directories.
	//
	// Allows supplying FileMode. If omitted, defaults to 0666 (i.e. a+rw) for new
	// files, and otherwise will keep the mode of the existing file.
	ModifyFile(path string, modify func(oldContents string) string, mode ...os.FileMode)

	// Stat returns the FileInfo for the entity at `path`, or nil if no such
	// entity exists.
	Stat(path string) os.FileInfo

	// Run executes a command `name` with arguments `args`.
	//
	// `name` may be relative to the Shell's cwd (e.g. `./main.star`), or to $PATH
	// (e.g. `git`).
	//
	// The command is expected to return exitcode 0.
	//
	// The command runs with the cwd of the Shell.
	//
	// Stdout+Stderr are redirected to logging (with Stdout logged at 'Info' level
	// and Stderr logged at 'Error' level).
	//
	// If the very last item in `args` is the TieStderr constant, both will be
	// tied together and all output will be logged at 'Info' level.
	Run(name string, args ...string)

	// Same as `Run`, but returns the exitcode of the process (instead of
	// asserting an exit code of 0).
	Retval(name string, args ...string) int

	// Same as `Run`, but returns the stdout of the process. If TieStderr is the
	// last argument, then this captures Stderr, too.
	Stdout(name string, args ...string) string
}
