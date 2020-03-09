// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package meta

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab/internal/site"
)

// skylabLatest is a fragment of a cipd manifest that is used to install the latest version of the skylab
// command line tool.
const skylabLatest = "chromiumos/infra/skylab/${platform} latest"

// Update subcommand: Update skylab tool.
var Update = &subcommands.Command{
	UsageLine: "update",
	ShortDesc: "update skylab tool",
	LongDesc: `Update skylab tool.

If you installed the skylab tool as a part of lab tools, you should
use update_lab_tools instead of this.

This is just a thin wrapper around CIPD.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type updateRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *updateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *updateRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	d, err := executableDir()
	if err != nil {
		return err
	}
	root, err := findCIPDRootDir(d)
	if err != nil {
		return err
	}

	if err := cipdEnsureLatest(a, root); err != nil {
		return err
	}
	fmt.Fprintf(a.GetErr(), "%s: You may need to run skylab login again after the update\n", a.GetName())
	fmt.Fprintf(a.GetErr(), "%s: Run skylab whoami to check login status\n", a.GetName())
	return nil
}

// executableDir returns the directory the current executable came
// from.
func executableDir() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", errors.Annotate(err, "get executable directory").Err()
	}
	return filepath.Dir(p), nil
}

func findCIPDRootDir(dir string) (string, error) {
	a, err := filepath.Abs(dir)
	if err != nil {
		return "", errors.Annotate(err, "find CIPD root dir").Err()
	}
	for d := a; d != "/"; d = filepath.Dir(d) {
		if isCIPDRootDir(d) {
			return d, nil
		}
	}
	return "", errors.Reason("find CIPD root dir: no CIPD root above %s", dir).Err()
}

func isCIPDRootDir(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, ".cipd"))
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}

// cipdEnsureLatest takes an application and a directory and runs a command with
// arguments that will read a cipd manifest from stdin and then run "ensure".
//
// Without this function, you need to run `sudo env PATH="$PATH" skylab update` in order to update
// skylab if skylab was installed as root.
//
// cipdEnsureLatest assumes that the directory exists and that the [[dir]]/.cipd directory
// exists.
func cipdEnsureLatest(a subcommands.Application, dir string) error {
	// We create two runnable command objects that update the cipd directory.
	// One runs as the current user and the other always runs as root.
	// If the command that runs as the current user fails, then we try the second command.
	asSelf := exec.Command("cipd", "ensure", "-root", dir, "-ensure-file", "-")
	asSelf.Stdin = strings.NewReader(skylabLatest)
	asSelf.Stdout = a.GetOut()
	asSelf.Stderr = a.GetErr()
	// Windows does not support sudo
	pathvar := fmt.Sprintf("PATH=%s", os.Getenv("PATH"))
	asRootUnix := exec.Command("sudo", "/usr/bin/env", pathvar, "cipd", "ensure", "-root", dir, "-ensure-file", "-")
	asRootUnix.Stdin = strings.NewReader(skylabLatest)
	asRootUnix.Stdout = a.GetOut()
	asRootUnix.Stderr = a.GetErr()

	if err := asSelf.Run(); err == nil {
		return nil
	}

	// We unconditionally run `sudo` on all OS's, however, we expect it to fail on Windows.
	fmt.Fprintf(a.GetErr(), "Retrying as root. Updating skylab through cipd.\n")
	if err := asRootUnix.Run(); err != nil {
		return fmt.Errorf("updating cipd as root: %s", err)
	}
	return nil
}
