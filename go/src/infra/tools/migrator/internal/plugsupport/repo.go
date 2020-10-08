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
	"path/filepath"
	"strings"
	"sync"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	configpb "go.chromium.org/luci/common/proto/config"

	"infra/tools/migrator"
)

type repo struct {
	root string

	relConfigRoot          string
	relGeneratedConfigRoot string

	projID string

	ctx context.Context
}

func (r *repo) Project() migrator.Project {
	return &localProject{
		id:  migrator.ReportID{Project: r.projID},
		dir: filepath.Join(r.root, r.relGeneratedConfigRoot),
		ctx: r.ctx,
	}
}

const (
	generatedConfigRootKey = "migrator.generatedConfigRoot"
	relConfigRootKey       = "migrator.relConfigRoot"
)

func (r *repo) ConfigRoot() string          { return "/" + r.relConfigRoot }
func (r *repo) GeneratedConfigRoot() string { return "/" + r.relGeneratedConfigRoot }

func loadRepo(ctx context.Context, project ProjectDir, projID string) (ret migrator.Repo, err error) {
	git := gitRunner{ctx: ctx, root: project.ProjectRepo(projID)}

	ret = &repo{
		root: git.root,

		relConfigRoot:          git.read("config", relConfigRootKey),
		relGeneratedConfigRoot: git.read("config", generatedConfigRootKey),

		projID: projID,

		ctx: ctx,
	}

	return ret, git.err
}

func createRepo(ctx context.Context, project ProjectDir, projPB *configpb.Project) (err error) {
	realPath := project.ProjectRepo(projPB.Id)
	gitLoc := projPB.GetGitilesLocation()

	// We do this because `git cl` makes very broad assumptions about ref names.
	var originRef string
	if prefix := "refs/heads/"; strings.HasPrefix(gitLoc.Ref, prefix) {
		originRef = strings.Replace(gitLoc.Ref, prefix, "refs/remotes/origin/", 1)
	} else if prefix := "refs/branch-heads/"; strings.HasPrefix(gitLoc.Ref, prefix) {
		originRef = strings.Replace(gitLoc.Ref, prefix, "refs/remotes/branch-heads/", 1)
	} else {
		err = errors.Reason("malformed GitilesLocation.Ref, must be `refs/heads/` or `refs/branch-heads/`: %q", gitLoc.Ref).Err()
		return
	}

	git := gitRunner{ctx: ctx, root: project.ProjectRepoTemp(projPB.Id)}

	if err = os.Mkdir(git.root, 0777); err != nil {
		err = errors.Annotate(err, "creating repo checkout").Err()
		return
	}

	// "sso://" simplifies authenticating into internal repos.
	remoteURL := strings.Replace(gitLoc.Repo, "https://", "sso://", 1)

	// Bail early with a clear error message if we have no read access.
	git.run("ls-remote", remoteURL, gitLoc.Ref)
	if git.err != nil {
		err = errors.Reason("no read access to %q ref %q", gitLoc.Repo, gitLoc.Ref).Err()
		return
	}

	git.run("init")
	git.run("config", "extensions.PartialClone", "origin")
	git.run("config", "depot-tools.upstream", originRef)
	git.run("remote", "add", "origin", remoteURL)
	git.run("config", "remote.origin.fetch", "+"+gitLoc.Ref+":"+originRef)
	git.run("config", "remote.origin.partialclonefilter", "blob:none")
	git.run("fetch", "--depth", "1", "origin")

	// toAdd will have the list of file patterns we want from our sparse checkout;
	// We do the `sparse-checkout add` call at most once because it's pretty slow
	// on each invocation (it updates some internal git state and may also do
	// network fetches to pull down missing blobs; this is optimized if you feed
	// it all the new patterns simultaneously).
	toAdd := stringset.Set{}
	toAdd.Add(gitLoc.Path)

	var foundRelConfigRoot bool
	relConfigRoot := ""

	// Run from gitLoc.Path all the way up to "."; We need to add all OWNERS files
	// and will calculate relConfigRoot along the way.
	//
	// TODO(iannucci): have a deterministic way to find the relConfigRoot; maybe
	// a generated metadata file?
	for cur := gitLoc.Path; cur != "."; cur = path.Dir(cur) {
		if !foundRelConfigRoot && git.check("cat-file", "-t", originRef+":"+cur+"/main.star") {
			foundRelConfigRoot = true
			relConfigRoot = cur
			toAdd.Add(cur)
		}
		toAdd.Add(filepath.Join(cur, "DIR_METADATA"))
		toAdd.Add(filepath.Join(cur, "OWNERS"))
		toAdd.Add(filepath.Join(cur, "PRESUBMIT.py"))
	}

	if relConfigRoot == "" {
		// We didn't find it heuristically.
		relConfigRoot = gitLoc.Path
	}

	// Finalize the checkout.

	// We do a sparse checkout iff the relConfigRoot is somewhere deeper than
	// the root of the repo. Otherwise the whole checkout is the config
	// directory.
	if !(relConfigRoot == "" || relConfigRoot == ".") {
		git.run("sparse-checkout", "init")
		git.run(append([]string{"sparse-checkout", "add"}, toAdd.ToSortedSlice()...)...)
		if err = git.err; err != nil {
			return
		}
	}

	git.run("new-branch", "fix_config")

	git.run("config", generatedConfigRootKey, gitLoc.Path)
	git.run("config", relConfigRootKey, relConfigRoot)

	if err = git.err; err != nil {
		return
	}

	return os.Rename(git.root, realPath)
}

// CreateOrLoadRepo loads a new repo, checking it out if it's not available
// locally.
//
// If `projPB` is nil, the repo MUST exist locally, or this returns an error.
//
// Returns `true` if this did a new checkout.
func CreateOrLoadRepo(ctx context.Context, project ProjectDir, projID string, projPB *configpb.Project) (ret migrator.Repo, newCheckout bool, err error) {
	realPath := project.ProjectRepo(projID)

	if _, err = os.Stat(realPath); err != nil && !os.IsNotExist(err) {
		err = errors.Annotate(err, "statting checkout").Err()
		return
	} else if os.IsNotExist(err) {
		if projPB == nil {
			err = errors.Reason("projPB==nil and project %q is not already checked out", projID).Err()
			return
		}
		newCheckout = true
		if err = createRepo(ctx, project, projPB); err != nil {
			return
		}
	}

	ret, err = loadRepo(ctx, project, projID)
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
				line = bytes.TrimRight(line, "\r\n")
				if err == io.EOF && len(line) == 0 {
					break
				}
				log(stdout, fmt.Sprintf("%s: %s", cmd.Args[0], line))
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

func (r *gitRunner) read(args ...string) string {
	if r.err != nil {
		return ""
	}

	logging.Infof(r.ctx, "running git %q", args)

	buf := &bytes.Buffer{}

	cmd := exec.CommandContext(r.ctx, "git", args...)
	cmd.Stdout = buf
	cmd.Dir = r.root
	err := redirectIOAndWait(cmd, func(fromStdout bool, line string) {
		logging.Errorf(r.ctx, "%s", line)
	})
	r.err = errors.Annotate(err, "running git %q", args).Err()
	return buf.String()
}
