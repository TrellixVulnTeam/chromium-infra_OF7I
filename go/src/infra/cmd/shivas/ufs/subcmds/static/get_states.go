// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package static

import (
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetStatesCmd get/list states offered by UFS.
var GetStatesCmd = &subcommands.Command{
	UsageLine: "states",
	ShortDesc: "Get all the states offered by UFS",
	LongDesc: `Get all the states offered by UFS

Example:

shivas get states`,
	CommandRun: func() subcommands.CommandRun {
		c := &getStates{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)
		return c
	},
}

type getStates struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	keysOnly bool
}

type state struct {
	Name        string `json:"Name"`
	EnumName    string `json:"EnumName"`
	Description string `json:"Description"`
}

type noEmitState struct {
	Name        string `json:"Name,omitempty"`
	EnumName    string `json:"EnumName,omitempty"`
	Description string `json:"Description,omitempty"`
}

func (c *getStates) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getStates) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if c.outputFlags.JSON() {
		emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
		if emit {
			return utils.PrintJSON(getEmitStates(c.keysOnly))
		}
		return utils.PrintJSON(getNoEmitStates(c.keysOnly))
	}

	res := outputStateStr(getEmitStates(c.keysOnly), c.keysOnly)
	if c.outputFlags.Tsv() {
		utils.PrintAllTSVs(res)
	} else {
		utils.PrintAllNormal(utils.StateTitle, res, c.keysOnly)
	}
	return nil
}

func getEmitStates(keysOnly bool) []*state {
	states := make([]*state, 0, 0)
	for stateValue := range ufspb.State_value {
		if !strings.Contains(stateValue, "UNSPECIFIED") {
			s := &state{
				Name: strings.ToLower(ufsUtil.GetSuffixAfterSeparator(stateValue, "_")),
			}
			if !keysOnly {
				s.EnumName = stateValue
				s.Description = ufsUtil.GetStateDescription(stateValue)
			}
			states = append(states, s)
		}
	}
	return states
}

func getNoEmitStates(keysOnly bool) []*noEmitState {
	states := make([]*noEmitState, 0, 0)
	for stateValue := range ufspb.State_value {
		if !strings.Contains(stateValue, "UNSPECIFIED") {
			s := &noEmitState{
				Name: strings.ToLower(ufsUtil.GetSuffixAfterSeparator(stateValue, "_")),
			}
			if !keysOnly {
				s.EnumName = stateValue
				s.Description = ufsUtil.GetStateDescription(stateValue)
			}
			states = append(states, s)
		}
	}
	return states
}

func outputStateStr(states []*state, keysOnly bool) [][]string {
	res := make([][]string, len(states))
	for i := 0; i < len(states); i++ {
		if keysOnly {
			res[i] = []string{states[i].Name}
			continue
		}
		res[i] = []string{
			states[i].Name,
			states[i].EnumName,
			states[i].Description,
		}
	}
	return res
}
