// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

type shell struct {
	repo *repo

	cwd string // relative to repo.root
}

func (s *shell) Cd(path string) {
	newRel := s.computeRepoRelative(path)

	abs := filepath.Join(s.repo.root, newRel)
	st, err := os.Stat(abs)
	if err != nil {
		panic(errors.Annotate(err, "Cd(%q) -> /%s", path, newRel).Err())
	}
	if !st.IsDir() {
		panic(errors.Reason("Cd(%q) -> /%s: not a directory", path, newRel).Err())
	}

	s.cwd = newRel
}

func (s *shell) computeRepoRelative(path string) string {
	if len(path) == 0 {
		return s.cwd
	}
	var newRel string
	if strings.HasPrefix(path, "/") {
		newRel = filepath.Clean(path)
	} else {
		newRel = filepath.Join(s.cwd, path)
	}
	if strings.HasPrefix(newRel, ".."+string(filepath.Separator)) {
		panic(errors.Reason("computeRepoRelative(%q) would leave the repo", path).Err())
	}
	return newRel
}

func (s *shell) ModifyFile(path string, modify func(oldContents string) string, mode ...os.FileMode) {
	abspath := filepath.Join(s.repo.root, s.computeRepoRelative(path))

	newMode := os.FileMode(0666)

	oldContent, err := ioutil.ReadFile(abspath)
	if err == nil {
		if len(mode) == 0 {
			st, err := os.Stat(abspath)
			if err != nil {
				panic(errors.Annotate(err, "statting %q", abspath).Err())
			}
			newMode = st.Mode()
		}
	} else if !os.IsNotExist(err) {
		panic(errors.Annotate(err, "while attempting to read/modify/write %q", abspath).Err())
	}

	newContent := []byte(modify(string(oldContent)))
	os.MkdirAll(filepath.Dir(abspath), 0777)
	err = ioutil.WriteFile(abspath, newContent, newMode)
	if err != nil {
		panic(errors.Annotate(err, "writing file %q", abspath).Err())
	}
}

func (s *shell) Stat(path string) os.FileInfo {
	relpath := s.computeRepoRelative(path)
	abspath := filepath.Join(s.repo.root, relpath)
	st, err := os.Stat(abspath)
	if os.IsNotExist(err) {
		return nil
	}
	if err == nil {
		return st
	}
	panic(errors.Annotate(err, "Stat(%q)", relpath).Err())
}

func (s *shell) Run(name string, args ...string) {
	cmd := exec.CommandContext(s.repo.ctx, name, args...)
	cmd.Dir = filepath.Join(s.repo.root, s.cwd)
	logging.Infof(s.repo.ctx, "%q: Run(%q %q)", cmd.Dir, name, args)
	if err := redirectIOAndWait(cmd, defaultLogger(s.repo.ctx)); err != nil {
		panic(errors.Annotate(err, "Run(%q, %q)", name, args).Err())
	}
}

func (s *shell) Retval(name string, args ...string) int {
	cmd := exec.CommandContext(s.repo.ctx, name, args...)
	cmd.Dir = filepath.Join(s.repo.root, s.cwd)
	logging.Infof(s.repo.ctx, "%q: Retval(%q %q)", cmd.Dir, name, args)
	if err := redirectIOAndWait(cmd, defaultLogger(s.repo.ctx)); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		panic(errors.Annotate(err, "Retval(%q, %q)", name, args).Err())
	}
	return 0
}

func (s *shell) Stdout(name string, args ...string) string {
	cmd := exec.CommandContext(s.repo.ctx, name, args...)
	cmd.Dir = filepath.Join(s.repo.root, s.cwd)
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	logging.Infof(s.repo.ctx, "%q: Stdout(%q %q)", cmd.Dir, name, args)
	if err := redirectIOAndWait(cmd, defaultLogger(s.repo.ctx)); err != nil {
		panic(errors.Annotate(err, "Stdout(%q, %q)", name, args).Err())
	}
	return buffer.String()
}
