// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

import (
	"github.com/golang/protobuf/proto"
)

// Reportable is an interface for reporting problems about an object.
//
// Implemented by Project and ConfigFile.
type Reportable interface {
	// Report logs a problem about this object.
	//
	// `tag` should be a CAPS_STRING which makes sense to your particular
	// application.
	//
	// `description` should be a human readable explanation of the problem.
	//
	// Example:
	//    tag: MISSING_BLARF
	//    description: "blarf.cfg file is missing"
	Report(tag, description string, opts ...ReportOption)
}

// Project encapsulates the pertinent details of a single LUCI Project's
// configuration files.
type Project interface {
	Reportable

	// ID is the project identifier, as it appears in projects.cfg.
	ID() string

	// ConfigFiles returns a mapping of path to config file.
	//
	// May do a (cached) network operation to retrieve the data.
	ConfigFiles() map[string]ConfigFile
}

// LocalProject is a checked out project that can be modified.
type LocalProject interface {
	Project

	// ConfigRoot returns the path to the config file 'root' within the repo.
	//
	// This would be the directory with the root of the lucicfg Starlark code
	// tree, if the repo has one, otherwise it is the root of the repository.
	//
	// This an 'absolute-style' path (see Shell).
	ConfigRoot() string

	// GeneratedConfigRoot returns the path to the generated config files (i.e.
	// the ones seen by the luci-config service).
	//
	// This an 'absolute-style' path (see Shell).
	GeneratedConfigRoot() string

	// Repo is the repository that contains this project.
	Repo() Repo

	// Shell returns a new shell object that can be used to modify configs.
	//
	// Its cwd is set to the ConfigRoot.
	Shell() Shell

	// RegenerateConfigs runs lucicfg to regenerate project configs.
	RegenerateConfigs()
}

// ConfigFile encapsulates as single configuration file from a Project.
type ConfigFile interface {
	Reportable

	// Path returns the path relative to the repo's generated configuration
	// directory.
	Path() string

	// RawData returns the data of this configuration file as a string.
	//
	// May do a (cached) network operation to retrieve the raw data.
	RawData() string

	// TextPb parses the raw data as a text proto with "heredoc" processing into
	// `out`.
	//
	// May do a (cached) network operation to retrieve the raw data.
	//
	// If there are errors during parsing, this panics.
	TextPb(out proto.Message)
}
