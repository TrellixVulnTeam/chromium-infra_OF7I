// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"fmt"
	"strings"
)

// multierror is a simple error implementation that represents multiple errors.
type multierror struct {
	errs []error
}

// Error reports all of the represented errors.
func (m multierror) Error() string {
	messages := make([]string, 1+len(m.errs))
	if len(m.errs) == 1 {
		messages[0] = "1 error occurred:\n"
	} else {
		messages[0] = fmt.Sprintf("%d errors occurred:\n", len(m.errs))
	}
	for i, err := range m.errs {
		messages[i+1] = fmt.Sprintf("  * %s\n", err.Error())
	}
	return strings.Join(messages, "")
}

type validatable interface {
	validate(*validator)
}

// validate performs type-specific validation of x.
//
// x.validate will be called with a validator object. If any errors are
// recorded on the validator, then an error that reports all of the
// recorded errors will be returned, otherwise nil will be returned.
func validate(x validatable, name string) error {
	v := &validator{location: name}
	x.validate(v)
	if len(v.errs) == 0 {
		return nil
	}
	return multierror{v.errs}
}

// validator provides the means for recording errors in an object.
type validator struct {
	// location is the name of the element being validated. This is used in
	// errorf for creating messages that include the full chain of member
	// references.
	location string
	// errs is the errors that have been recorded.
	errs []error
}

// validate validates a nested object.
//
// x.validate will be called with a validator object with an updated location
// taking the form of {v.location}.{name}. On completion, if any errors are
// recorded on the validator, they will be recorded on v.
func (v *validator) validate(x validatable, name string) {
	err := validate(x, fmt.Sprint(v.location, ".", name))
	if err != nil {
		v.errs = append(v.errs, err.(multierror).errs...)
	}
}

// errorf records a validation error.
//
// The arguments to errorf are interpreted the same as for fmt.Errorf, with the
// addition that all occurrences of the substring "${}" in the format string
// will be replaced by the value of v.location before formatting takes place.
func (v *validator) errorf(format string, a ...interface{}) {
	format = strings.ReplaceAll(format, "${}", v.location)
	v.errs = append(v.errs, fmt.Errorf(format, a...))
}

func (x *GitilesRepo) validate(v *validator) {
	if x.Host == "" {
		v.errorf("${}.host is not set")
	}
	if x.Project == "" {
		v.errorf("${}.project is not set")
	}
}

func (x *BootstrapProperties) validate(v *validator) {
	switch config := x.ConfigProject.(type) {
	case *BootstrapProperties_TopLevelProject_:
		v.validate(config, "top_level_project")

	case *BootstrapProperties_DependencyProject_:
		v.validate(config, "dependency_project")

	case nil:
		v.errorf("none of the config_project fields in ${} is set")

	default:
		v.errorf("unexpected type for ${}.config_project: %T", config)
	}
	if x.PropertiesFile == "" {
		v.errorf("${}.properties_file is not set")
	}
	if x.Exe == nil {
		v.errorf("${}.exe is not set")
	} else {
		if x.Exe.CipdPackage == "" {
			v.errorf("${}.exe.cipd_package is not set")
		}
		if x.Exe.CipdVersion == "" {
			v.errorf("${}.exe.cipd_version is not set")
		}
		if len(x.Exe.Cmd) == 0 {
			v.errorf("${}.exe.cmd is not set")
		}
	}
}

func (x *BootstrapProperties_TopLevelProject_) validate(v *validator) {
	t := x.TopLevelProject
	if t.Repo == nil {
		v.errorf("${}.repo is not set")
	} else {
		v.validate(t.Repo, "repo")
	}
	if t.Ref == "" {
		v.errorf("${}.ref is not set")
	}
}

func (x *BootstrapProperties_DependencyProject_) validate(v *validator) {
	d := x.DependencyProject
	if d.TopLevelRepo == nil {
		v.errorf("${}.top_level_repo is not set")
	} else {
		v.validate(d.TopLevelRepo, "top_level_repo")
	}
	if d.TopLevelRef == "" {
		v.errorf("${}.top_level_ref is not set")
	}
	if d.ConfigRepo == nil {
		v.errorf("${}.config_repo is not set")
	} else {
		v.validate(d.ConfigRepo, "config_repo")
	}
	if d.ConfigRepoPath == "" {
		v.errorf("${}.config_repo_path is not set")
	}
}
