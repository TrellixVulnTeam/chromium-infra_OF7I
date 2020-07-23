// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package migrator provides interfaces and tooling for migrating LUCI
// configuration files across all known LUCI projects.
//
// The tool revolves around the user providing a plugin to perform bulk analysis
// and optional fixes across all LUCI projects' configuration folders, from the
// point of view of the 'luci-config' service.
//
// The plugin currently has two main points of extension:
//   * FindProblems - Analyze the LUCI project and config file contents to see
//     if the project has migrated. This can inspect the project name, as well
//     as the presence and contents of any config files.
//   * ApplyFix - If FindProblems revealed issues, ApplyFix will be run in the
//     context of a local sparse+shallow checkout containing the configuration
//     files. This has the ability to run programs in the checkout, as well as
//     stat/read/modify files.
//
// This package contains the interface definitions for the migrator plugin.
package migrator
