// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

import (
	"context"
)

// API is the implementation of a plugin, as returned by the plugin's
// InstantiateAPI function.
//
// One API instance will be created per LUCI project during 'scan', and
// `FindProblems` will be invoked before `ApplyFix`.
type API interface {
	// FindProblems allows you to report problems about a Project, or about
	// certain configuration files within the project.
	//
	// If the method finds issues which warrant followup, it should use
	// proj.Report and/or proj.ConfigFiles()["filename"].Report. Reporting one or
	// more problems will cause the migrator tool to set up a checkout for this
	// project.
	//
	// Logging is set up for this context, and will be diverted to a per-project
	// logfile.
	//
	// `proj` is implemented with LUCI Config API calls; it reflects the state of
	// the project as currently known by the luci-config service.
	//
	// This function should panic on error.
	FindProblems(ctx context.Context, proj Project)

	// ApplyFix allows you to attempt to automatically fix problems within a repo.
	//
	// Note that for real implementations you may want to keep details on the
	// `impl` struct; this will let you carry over information from
	// FindProblems.
	//
	// Logging is set up for this context, and will be diverted to a per-project
	// logfile.
	//
	// `repo.Project()` is implemented with local filesystem interactions on the
	// checked-out repo. This may differ from the current state of the luci-config
	// service.
	//
	// This function should panic on error.
	ApplyFix(ctx context.Context, repo Repo)
}

// InstantiateAPI is the symbol that plugins must export.
//
// It should return a new instance of API.
//
// If this returns nil, it has the effect of a plugin which:
//    FindProblems reports a generic problem "FindProblems not defined".
//    ApplyFix does nothing.
type InstantiateAPI func() API
