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
	lucipb "go.chromium.org/luci/common/proto"
	configpb "go.chromium.org/luci/common/proto/config"

	"infra/tools/migrator"
)

type repo struct {
	projectDir ProjectDir          // the root migrator project directory
	checkoutID string              // how to name the checkout directory on disk
	remoteURL  string              // https://... repo URL
	remoteRef  string              // e.g. "refs/heads/main"
	projects   []*configpb.Project // LUCI projects located within this repo
	root       string              // the absolute path to the repo checkout
}

// configRootKey is a key for "git config".
func configRootKey(projID string) string {
	return fmt.Sprintf("migrator.%s.configRoot", projID)
}

// generatedConfigRootKey is a key for "git config".
func generatedConfigRootKey(projID string) string {
	return fmt.Sprintf("migrator.%s.generatedConfigRoot", projID)
}

// localProject returns a reference to the local checked out project.
func (r *repo) localProject(ctx context.Context, projID string) migrator.LocalProject {
	git := gitRunner{ctx: ctx, root: r.root}
	return &localProject{
		id:                     migrator.ReportID{Project: projID},
		repo:                   r,
		ctx:                    ctx,
		relConfigRoot:          git.read("config", configRootKey(projID)),
		relGeneratedConfigRoot: git.read("config", generatedConfigRootKey(projID)),
	}
}

// initialize either creates or loads the repo checkout.
func (r *repo) initialize(ctx context.Context) (newCheckout bool, err error) {
	r.root = r.projectDir.CheckoutDir(r.checkoutID)
	switch _, err = os.Stat(r.root); {
	case os.IsNotExist(err):
		return true, r.create(ctx)
	case err == nil:
		return false, r.load(ctx)
	default:
		return false, errors.Annotate(err, "statting checkout").Err()
	}
}

// load verifies the checkout has all LUCI projects we need.
func (r *repo) load(ctx context.Context) error {
	git := gitRunner{ctx: ctx, root: r.root}

	for _, proj := range r.projects {
		configRoot := git.read("config", configRootKey(proj.Id))
		generatedConfigRoot := git.read("config", generatedConfigRootKey(proj.Id))
		if configRoot == "" || generatedConfigRoot == "" {
			return errors.Reason(
				"the checkout %q is lacking LUCI project %q; you may need to rerun the migration with -squeaky -clean flags",
				r.checkoutID, proj.Id,
			).Err()
		}
	}

	return git.err
}

// create initializes a new repo checkout.
func (r *repo) create(ctx context.Context) error {
	// We do this because `git cl` makes very broad assumptions about ref names.
	var originRef string
	if prefix := "refs/heads/"; strings.HasPrefix(r.remoteRef, prefix) {
		originRef = strings.Replace(r.remoteRef, prefix, "refs/remotes/origin/", 1)
	} else if prefix := "refs/branch-heads/"; strings.HasPrefix(r.remoteRef, prefix) {
		originRef = strings.Replace(r.remoteRef, prefix, "refs/remotes/branch-heads/", 1)
	} else {
		return errors.Reason("malformed remote ref, must be `refs/heads/` or `refs/branch-heads/`: %q", r.remoteRef).Err()
	}

	// Bail early if the migrator config is broken.
	migratorCfg, err := r.projectDir.LoadConfigFile()
	if err != nil {
		return errors.Annotate(err, "bad migrator config in %q", r.projectDir).Err()
	}

	git := gitRunner{ctx: ctx, root: r.projectDir.CheckoutTemp(r.checkoutID)}
	if err = os.Mkdir(git.root, 0777); err != nil {
		return errors.Annotate(err, "creating repo checkout").Err()
	}

	// "sso://" simplifies authenticating into internal repos.
	remoteURL := strings.Replace(r.remoteURL, "https://", "sso://", 1)

	// Bail early with a clear error message if we have no read access.
	git.run("ls-remote", remoteURL, r.remoteRef)
	if git.err != nil {
		return errors.Reason("no read access to %q ref %q", remoteURL, r.remoteRef).Err()
	}

	// Fetch the state into the git guts, but do not check out it yet.
	git.run("init")
	for key, val := range migratorCfg.GetGit().Config {
		git.run("config", key, val)
	}
	git.run("config", "extensions.PartialClone", "origin")
	git.run("config", "depot-tools.upstream", originRef)
	git.run("remote", "add", "origin", remoteURL)
	git.run("config", "remote.origin.fetch", "+"+r.remoteRef+":"+originRef)
	git.run("config", "remote.origin.partialclonefilter", "blob:none")
	git.run("fetch", "--depth", "1", "origin")

	// Figure out what directories we need to have in the checkout.
	toAdd := stringset.Set{}
	for _, proj := range r.projects {
		if err := r.prepRepoForProject(&git, originRef, proj, toAdd); err != nil {
			return errors.Annotate(err, "when examining LUCI project %q", proj.Id).Err()
		}
	}

	// We do a sparse checkout iff the stuff we want is somewhere deeper than
	// the root of the repo. Otherwise the whole checkout is the config
	// directory.
	if !toAdd.Has(".") {
		git.run("sparse-checkout", "init")
		git.run(append([]string{"sparse-checkout", "add"}, toAdd.ToSortedSlice()...)...)
	}
	git.run("new-branch", "fix_config")
	if git.err != nil {
		return git.err
	}

	return os.Rename(git.root, r.root)
}

// prepRepoForProject figures out what directories we need to check out.
func (r *repo) prepRepoForProject(git *gitRunner, originRef string, proj *configpb.Project, toAdd stringset.Set) error {
	// Path where generated configs (e.g. project.cfg) are.
	generatedRoot := proj.GetGitilesLocation().GetPath()
	if generatedRoot == "" {
		generatedRoot = "."
	}

	// Need to checkout all generated files themselves.
	toAdd.Add(generatedRoot)

	// Run from generatedRoot all the way up to "."; We need to add all OWNERS
	// files.
	for cur := generatedRoot; cur != "."; cur = path.Dir(cur) {
		toAdd.Add(filepath.Join(cur, "DIR_METADATA"))
		toAdd.Add(filepath.Join(cur, "OWNERS"))
		toAdd.Add(filepath.Join(cur, "PRESUBMIT.py"))
	}

	// Attempt to read project.cfg from the git guts. It contains lucicfg metadata
	// describing how to find the root of the lucicfg config tree.
	var projectCfg configpb.ProjectCfg
	blob := git.read("cat-file", "-p", fmt.Sprintf("%s:%s/project.cfg", originRef, generatedRoot))
	if blob != "" {
		if err := lucipb.UnmarshalTextML(blob, &projectCfg); err != nil {
			return errors.Annotate(err, "failed to unmarshal project.cfg").Err()
		}
	}

	// We need to checkout the directory with lucicfg's main package. Grab its
	// location from the project config metadata but fallback to a heuristic of
	// finding the main.star for projects that don't have the metadata yet.
	var configRoot string
	if packageDir := projectCfg.GetLucicfg().GetPackageDir(); packageDir != "" {
		configRoot = path.Join(generatedRoot, packageDir)
	} else {
		// Go up until we see main.star.
		for configRoot = generatedRoot; configRoot != "."; configRoot = path.Dir(configRoot) {
			if git.check("cat-file", "-t", originRef+":"+configRoot+"/main.star") {
				break
			}
		}
	}
	toAdd.Add(configRoot)

	// Store these directories for reuse in load(...) and localProject(...).
	git.run("config", configRootKey(proj.Id), configRoot)
	git.run("config", generatedConfigRootKey(proj.Id), generatedRoot)

	return git.err
}

type gitRunner struct {
	root string
	err  error
	ctx  context.Context
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
	return strings.TrimSpace(buf.String())
}
