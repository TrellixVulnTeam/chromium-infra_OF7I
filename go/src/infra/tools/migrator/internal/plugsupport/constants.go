// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"go.chromium.org/luci/common/errors"
)

// ProjectDir is an absolute path to a migrator project directory.
type ProjectDir string

// PluginDir returns the absolute path of the migrator project's plugin code
// directory.
func (p ProjectDir) PluginDir() string {
	return filepath.Join(string(p), "_plugin")
}

// ConfigDir returns the absolute path of the migrator project's config
// directory.
func (p ProjectDir) ConfigDir() string {
	return filepath.Join(string(p), ".migration")
}

// ConfigFile returns the absolute path of the migrator project's main config
// file.
//
// The existance of this file is used to determine if a folder is a migrator
// project.
func (p ProjectDir) ConfigFile() string {
	return filepath.Join(p.ConfigDir(), "config")
}

// TrashDir returns the absolute path of the migrator project's trash
// directory.
//
// The trash directory is used to compile the plugin; New runs of migrator will
// make best-effort attempts to clean up this directory using CleanTrash().
func (p ProjectDir) TrashDir() string {
	return filepath.Join(string(p), ".trash")
}

// ReportPath returns the absolute path of the migrator project's CSV scan
// report file.
func (p ProjectDir) ReportPath() string {
	return filepath.Join(string(p), "scan.csv")
}

// ProjectLog returns the absolute path of the scan log for a given LUCI
// project within this migrator project.
func (p ProjectDir) ProjectLog(projectID string) string {
	return filepath.Join(string(p), projectID+".scan.log")
}

// ProjectRepoTemp returns a LUCI project temporary checkout directory.
//
// During repo creation, the initial git repo is cloned here and then moved to
// its ProjectRepo() path on success.
func (p ProjectDir) ProjectRepoTemp(projectID string) string {
	return filepath.Join(p.TrashDir(), projectID)
}

// ProjectRepo returns the path for a specific LUCI project's git checkout.
func (p ProjectDir) ProjectRepo(projectID string) string {
	return filepath.Join(string(p), projectID)
}

// MkTempDir generates a new temporary directory within TrashDir().
func (p ProjectDir) MkTempDir() (string, error) {
	if err := os.Mkdir(p.TrashDir(), 0777); err != nil {
		return "", err
	}
	return ioutil.TempDir(p.TrashDir(), "")
}

// CleanTrash removes TrashDir().
func (p ProjectDir) CleanTrash() error {
	return os.RemoveAll(p.TrashDir())
}

// FindProjectRoot finds a migrator ProjectDir starting from `abspath` and
// working up towards the filesystem root.
func FindProjectRoot(abspath string) (ProjectDir, error) {
	curPath := ProjectDir(abspath)
	for {
		if st, err := os.Stat(curPath.ConfigFile()); err == nil {
			if st.Mode().IsRegular() {
				return curPath, nil
			}
		}
		newPath := ProjectDir(filepath.Dir(string(curPath)))
		if newPath == curPath {
			break
		}
		curPath = newPath
	}
	return "", errors.Reason("not in a migrator project: %q", abspath).Err()
}
