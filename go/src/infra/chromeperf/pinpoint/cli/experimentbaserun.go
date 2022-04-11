// Copyright 2020 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
)

type experimentBaseRun struct {
	baseCommandRun
	waitForJobMixin
	downloadResultsMixin
	downloadArtifactsMixin
	analyzeExperimentMixin
	presetsMixin
	issue          bugValue
	configurations []string

	// Git and Gerrit specific configuration flags for specifying CLs we're
	// running experiments with.
	gitilesHost, gerritHost, repository string

	// Commits and CLs for the base and experimental versions.
	baseCommit, expCommit string
	baseCL, expCL         clValue
}

// TODO(crbug.com/1174964): Move these parsers to luci/common/flags then migrate when done.

type bugValue struct {
	project string
	issueID int64
}

func (b *bugValue) String() string {
	return fmt.Sprintf("%s:%d", b.project, b.issueID)
}

var bugRe = regexp.MustCompile(`^([\w]+):([1-9][\d]*)$`)

func (b *bugValue) Set(i string) error {
	p := bugRe.FindStringSubmatch(i)
	if p == nil {
		return fmt.Errorf("bug id must match %s", bugRe)
	}
	b.project = p[1]
	issueID, err := strconv.ParseInt(p[2], 10, 64)
	if err != nil {
		return fmt.Errorf("bug id must fit in a 64-bit int")
	}
	b.issueID = issueID
	return nil
}

type clValue struct {
	clNum, patchSet int64
}

func (c *clValue) String() string {
	if c.patchSet == 0 {
		return fmt.Sprintf("%d", c.clNum)
	}
	return fmt.Sprintf("%d/%d", c.clNum, c.patchSet)
}

var (
	hostPattern      = `(?:chromium-review.googlesource.com|crrev.com)`
	CLPattern        = `([1-9][\d]*)(?:/([1-9][\d]*))?`
	simpleURLPattern = fmt.Sprintf(
		`^https://%s/c/%s$`,
		hostPattern,
		CLPattern)
	URLPattern = fmt.Sprintf(
		`^https://%s/c/[^+]+/\+/%s$`,
		hostPattern,
		CLPattern)
	simpleURLRe = regexp.MustCompile(simpleURLPattern)
	URLRe       = regexp.MustCompile(URLPattern)
)

func (c *clValue) Set(i string) error {

	p := URLRe.FindStringSubmatch(i)
	if p == nil {
		p = simpleURLRe.FindStringSubmatch(i)
	}
	if p == nil {
		return fmt.Errorf("cl must match either %s or %s", simpleURLRe.String(), URLRe.String())
	}

	clNum, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil {
		return fmt.Errorf("cl number must fit in 64 bits")
	}
	if len(p[2]) > 0 {
		patchSet, err := strconv.ParseInt(p[2], 10, 64)
		if err != nil {
			return fmt.Errorf("patchset must fit in 64 bits")
		}
		c.patchSet = patchSet
	}
	c.clNum = clNum
	return nil
}

func (e *experimentBaseRun) RegisterFlags(p Param) {
	uc := e.baseCommandRun.RegisterFlags(p)
	e.waitForJobMixin.RegisterFlags(&e.Flags, uc)
	e.downloadResultsMixin.RegisterFlags(&e.Flags, uc)
	e.downloadArtifactsMixin.RegisterFlags(&e.Flags, uc)
	e.analyzeExperimentMixin.RegisterFlags(&e.Flags, uc)
	e.presetsMixin.RegisterFlags(&e.Flags, uc)
	e.Flags.Var(&e.issue, "bug", text.Doc(`
		Monorail issue id in the form <project>:<issue id>.
	`))
	// TODO(crbug.com/1172875): Provide a command to query the list of supported configs.
	e.Flags.Var(flag.CommaList(&e.configurations), "cfgs", text.Doc(`
		See "cfg".
	`))
	e.Flags.Var(e.Flags.Lookup("cfgs").Value, "cfg", text.Doc(`
		Configuration name (or comma-separated list of names)
		supported by Pinpoint (AKA bot).
	`))
	e.Flags.StringVar(&e.baseCommit, "base-commit", "HEAD", text.Doc(`
		git commit hash (symbolic like HEAD, short-form, or long-form)
		for the base build.
	`))
	e.Flags.Var(&e.baseCL, "base-patch-url", text.Doc(`
		Gerrit CL to apply to base-commit (optional).
		The input must be a valid Gerrit URL such as
		https://chromium-review.googlesource.com/c/<repo name>/+/<cl number>/<patchset number>
		or just https://chromium-review.googlesource.com/c/<repo name>/+/<cl number>.
		When <patchset number> is not provided, we'll use the latest patchset of the CL.
		crrev.com is also supported.
	`))
	e.Flags.StringVar(&e.expCommit, "exp-commit", "HEAD", text.Doc(`
		git commit hash (symbolic like HEAD, short-form, or long-form)
		for the experiment build. This may be different from -base-commit
		and defaults to what -base-commit is set to.
	`))
	e.Flags.Var(&e.expCL, "exp-patch-url", text.Doc(`
		Gerrit CL to apply to exp-commit.
		The input must be a valid Gerrit URL such as
		https://chromium-review.googlesource.com/c/<repo name>/+/<cl number>/<patchset number>
		or just https://chromium-review.googlesource.com/c/<repo name>/+/<cl number>.
		When <patchset number> is not provided, we'll use the latest patchset of the CL.
		crrev.com is also supported.
	`))

	// We drop the error because we don't want to spam the user if they are
	// running from some random directory.
	gitilesHost, gerritHost, repository, _ := guessRepositoryDefaults(realGitCLIssue)

	e.Flags.StringVar(&e.gitilesHost, "gitiles-host", gitilesHost, text.Doc(`
      Gitiles host to retrieve commits from. This flag's default is inferred
      from the directory where the command is executed.
	`))
	e.Flags.StringVar(&e.gerritHost, "gerrit-host", gerritHost, text.Doc(`
      Gerrit host to retrieve CLs from. This flag's default is inferred from
      the directory where the command is executed.
	`))
	e.Flags.StringVar(&e.repository, "repository", repository, text.Doc(`
      Project associated with Gerrit and Gitiles to fetch code and CLs from.
      This flag's default is inferred from the directory where the command is
      executed.
	`))
}

const (
	defaultGitilesHost = "chromium.googlesource.com"
	defaultGerritHost  = "chromium-review.googlesource.com"
	defaultRepository  = "chromium/src"
)

// guessRepositoryDefaults returns appropriate default values for a variety of
// flags. If no good default value can be inferred from the environment, or an
// error occurs, chromium-specific values will be returned along with any
// relevant error.
func guessRepositoryDefaults(writeJSON writeGitCLJSON) (gitilesHost, gerritHost, repository string, _ error) {
	var tmpName string
	{
		dir, err := os.MkdirTemp("", "pinpoint_git_cl")
		if err != nil {
			return defaultGitilesHost, defaultGerritHost, defaultRepository, err
		}
		defer os.RemoveAll(dir)
		tmpName = filepath.Join(dir, "issue.json")
	}

	if err := writeJSON(tmpName); err != nil {
		return defaultGitilesHost, defaultGerritHost, defaultRepository, err
	}
	bs, err := os.ReadFile(tmpName)
	if err != nil {
		return defaultGitilesHost, defaultGerritHost, defaultRepository, err
	}
	var x struct {
		GerritHost    string `json:"gerrit_host"`
		GerritProject string `json:"gerrit_project"`
	}
	if err := json.Unmarshal(bs, &x); err != nil {
		return defaultGitilesHost, defaultGerritHost, defaultRepository, err
	}
	if x.GerritHost == "" || x.GerritProject == "" {
		return defaultGitilesHost, defaultGerritHost, defaultRepository, errors.Reason("no gerrit_host and/or gerrit_project found in `git cl issue` output: %s", bs).Err()
	}
	// Guess the gitiles host based off the gerrit host
	gitilesHost = strings.Replace(x.GerritHost, "-review", "", 1)
	return gitilesHost, x.GerritHost, x.GerritProject, nil
}

// writeGitCLJSON encapsulates the operation of executing `git cl issue` in a
// way that is easy to swap out for testing. Upon returning, the file at the
// provided path should be overwritten with the output JSON data.
//
// Use realGitCLIssue to actually execute the git command.
type writeGitCLJSON func(intoFile string) error

func realGitCLIssue(intoFile string) error {
	return exec.Command("git", "cl", "issue", "--json", intoFile).Run()
}
