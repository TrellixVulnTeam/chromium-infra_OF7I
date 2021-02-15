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
	"fmt"
	"regexp"
	"strconv"

	"go.chromium.org/luci/common/data/text"
)

type experimentBaseRun struct {
	baseCommandRun
	issue         bugValue
	configuration string
	repository    string
	baseCommit    string
	expCommit     string
	baseCL, expCL clValue
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

var clRe = regexp.MustCompile(`^([1-9][\d]*)(/([1-9][\d]*))?$`)

func (c *clValue) Set(i string) error {
	p := clRe.FindStringSubmatch(i)
	if p == nil {
		return fmt.Errorf("cl must match %s", clRe.String())
	}
	clNum, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil {
		return fmt.Errorf("cl number must fit in 64 bits")
	}
	if len(p[3]) > 0 {
		patchSet, err := strconv.ParseInt(p[3], 10, 64)
		if err != nil {
			return fmt.Errorf("patchset must fit in 64 bits")
		}
		c.patchSet = patchSet
	}
	c.clNum = clNum
	return nil
}

func (e *experimentBaseRun) RegisterDefaultFlags(p Param) {
	e.baseCommandRun.RegisterDefaultFlags(p)
	e.Flags.Var(&e.issue, "bug", text.Doc(`
		Monorail issue id in the form <project>:<issue id>.
	`))
	// TODO(crbug.com/1172875): Provide a command to query the list of supported configs.
	e.Flags.StringVar(&e.configuration, "cfg", "", text.Doc(`
		Configuration name supported by Pinpoint (AKA bot).
		`))
	// TODO(crbug.com/1172875): Provide a command to query the list of supported repositories.
	e.Flags.StringVar(&e.repository, "repo", "chromium", text.Doc(`
		Repository to build/run the Pinpoint job against.`))
	e.Flags.StringVar(&e.baseCommit, "base-commit", "HEAD", text.Doc(`
		git commit hash (symbolic like HEAD, short-form, or long-form)
		for the base build.`))
	e.Flags.Var(&e.baseCL, "base-cl", text.Doc(`
		Gerrit CL to apply to base-commit (optional).
		This must be of the form <cl number>/<patchset number> or just <cl number>.
		When <patchset number> is not provided, we'll use the latest patchset of the CL.
	`))
	e.Flags.StringVar(&e.expCommit, "exp-commit", "HEAD", text.Doc(`
		git commit hash (symbolic like HEAD, short-form, or long-form)
		for experiment build. This may be different from -base-commit.
		`))
	e.Flags.Var(&e.expCL, "exp-cl", text.Doc(`
		Gerrit CL to apply to exp-commit.
		This must be of the form <cl number>/<patchset number> or just <cl number>.
		When <patchset number> is not provided, we'll use the latest patchset of the CL.
	`))
}
