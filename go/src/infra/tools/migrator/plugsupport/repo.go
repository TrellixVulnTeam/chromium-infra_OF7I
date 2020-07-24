// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	configpb "go.chromium.org/luci/common/proto/config"

	"infra/tools/migrator"
)

type repo struct {
	root string

	relConfigRoot          string
	relGeneratedConfigRoot string

	projPB *configpb.Project

	ctx context.Context
}

func (r *repo) Project() migrator.Project {
	panic("IMPLEMENT PROJECT()")
}

func (r *repo) ConfigRoot() string          { return "/" + r.relConfigRoot }
func (r *repo) GeneratedConfigRoot() string { return "/" + r.relGeneratedConfigRoot }

const magicUpstreamRef = "refs/CONFIG_UPSTREAM"

// CreateRepo generates a new repo, checking it out if it's unavailable.
//
// Returns `true` if this did a new checkout.
func CreateRepo(ctx context.Context, project ProjectDir, projPB *configpb.Project) (ret migrator.Repo, newCheckout bool, err error) {
	realPath := project.ProjectRepo(projPB.Id)
	gitLoc := projPB.GetGitilesLocation()

	if _, err = os.Stat(realPath); err != nil && !os.IsNotExist(err) {
		err = errors.Annotate(err, "statting checkout").Err()
		return
	} else if os.IsNotExist(err) {
		newCheckout = true

		tempPath := project.ProjectRepoTemp(projPB.Id)

		if err = os.Mkdir(tempPath, 0777); err != nil {
			err = errors.Annotate(err, "creating repo checkout").Err()
			return
		}

		git := gitRunner{root: tempPath, ctx: ctx}
		git.run("init")
		git.run("config", "extensions.PartialClone", "origin")
		git.run("config", "depot-tools.upstream", magicUpstreamRef)
		git.run("sparse-checkout", "init")
		git.run("remote", "add", "origin", gitLoc.Repo)
		git.run("config", "remote.origin.fetch", "+"+gitLoc.Ref+":"+magicUpstreamRef)
		git.run("config", "remote.origin.partialclonefilter", "blob:none")
		git.run("fetch", "--depth", "1", "origin")
		git.run("sparse-checkout", "add", gitLoc.Path)
		git.run("new-branch", "fix_config")
		if err = git.err; err != nil {
			return
		}

		if err = os.Rename(tempPath, realPath); err != nil {
			return
		}
	}

	realRet := &repo{
		root: realPath,

		relGeneratedConfigRoot: gitLoc.Path,

		projPB: projPB,
		ctx:    ctx,
	}
	ret = realRet
	// TODO(iannucci): have a deterministic way to find the relConfigRoot; maybe
	// a generated metadata file?
	realRet.relConfigRoot = realRet.relGeneratedConfigRoot

	git := gitRunner{root: realRet.root, ctx: ctx}
	for cur := gitLoc.Path; cur != "."; cur = path.Dir(cur) {
		if git.check("cat-file", "-t", magicUpstreamRef+":"+cur+"/main.star") {
			realRet.relConfigRoot = cur
			if newCheckout {
				git.run("sparse-checkout", "add", cur)
				if err = git.err; git.err != nil {
					return
				}
			}
			break
		}
	}

	return
}

// Shell returns a new 'Shell' object for use in plugins.
func (r *repo) Shell() migrator.Shell {
	return &shell{repo: r, cwd: r.relConfigRoot}
}

type gitRunner struct {
	root string
	err  error
	ctx  context.Context
}

func defaultLogger(ctx context.Context) func(bool, string) {
	return func(fromStdout bool, line string) {
		if fromStdout {
			logging.Infof(ctx, "%s", line)
		} else {
			logging.Errorf(ctx, "%s", line)
		}
	}
}

// Sets up redirection for cmd.Std{out,err} to `log`.
//
// If cmd.Std{err,out} are non-nil prior to running this, they're left alone.
//
// The `log` function will be invoked with each line parsed from Std{out,err}.
// It should actually log this somewhere. `fromStdout` will be true if the line
// originated from the process' Stdout, false otherwise.
//
// If cmd.Args[-1] is exactly the string "2>&1" (i.e. migrator.TieStderr), then
// this will tie Stderr to Stdout. This means that `fromStdout` will always be
// true.
func redirectIOAndWait(cmd *exec.Cmd, log func(fromStdout bool, line string)) error {
	var wg sync.WaitGroup

	shuttleStdio := func(reader io.Reader, stdout bool) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			scanner := bufio.NewReader(reader)
			for {
				line, err := scanner.ReadBytes('\n')
				log(stdout, fmt.Sprintf("%s: %s", cmd.Args[0], bytes.TrimRight(line, "\r\n")))
				if err != nil {
					if err != io.EOF {
						panic(err)
					}
					break
				}
			}
		}()
	}

	tieStderr := false
	if cmd.Args[len(cmd.Args)-1] == migrator.TieStderr {
		tieStderr = true
		cmd.Args = cmd.Args[:len(cmd.Args)-1]
	}

	if cmd.Stdout == nil {
		outReader, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		shuttleStdio(outReader, true)
	}
	if cmd.Stderr == nil {
		if tieStderr {
			cmd.Stderr = cmd.Stdout
		} else {
			errReader, err := cmd.StderrPipe()
			if err != nil {
				panic(err)
			}
			shuttleStdio(errReader, false)
		}
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	wg.Wait()

	return cmd.Wait()
}

func (r *gitRunner) check(args ...string) bool {
	cmd := exec.CommandContext(r.ctx, "git", args...)
	cmd.Dir = r.root
	return cmd.Run() == nil
}

func (r *gitRunner) run(args ...string) {
	if r.err != nil {
		return
	}

	// git uses stderr for normal logging, but uses 'fatal' to indicate that bad
	// stuff happened. See the log function on redirectIOAndWait below.
	fatalLine := false
	args = append(args, migrator.TieStderr)

	logging.Infof(r.ctx, "running git %q", args)

	cmd := exec.CommandContext(r.ctx, "git", args...)
	cmd.Dir = r.root
	err := redirectIOAndWait(cmd, func(fromStdout bool, line string) {
		if strings.HasPrefix(line, "git: fatal: ") {
			fatalLine = true
		}
		if !fatalLine {
			logging.Infof(r.ctx, "%s", line)
		} else {
			logging.Errorf(r.ctx, "%s", line)
		}
	})
	r.err = errors.Annotate(err, "running git %q", args).Err()
}
