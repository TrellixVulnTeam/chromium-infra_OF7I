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

	ID() string

	// ConfigFiles returns a mapping of path to config file.
	//
	// May do a (cached) network operation to retrieve the data.
	ConfigFiles() map[string]ConfigFile
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
