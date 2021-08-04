// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

// TODO(gregorynisbet): Validate existence of required flags.

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/commands/dns"
	"infra/cros/cmd/satlab/internal/parse"
	"infra/cros/cmd/satlab/internal/paths"

	"go.chromium.org/luci/common/errors"
)

// AddDUT adds a DUT.
func AddDUT(serviceAccountJSON string, satlabPrefix string, p *parse.CommandParseResult) error {
	if p == nil {
		return errors.New("command parse cannot be nil")
	}

	// Check for the flags needed to construct or verify the DNS entry up front.
	host, ok := p.Flags["name"]
	if !ok {
		return errors.New("add dut: hostname (-name) is required")
	}
	addr, ok := p.Flags["address"]
	if !ok {
		return errors.New("add dut: address (-address) is required")
	}

	// TODO(gregorynisbet): verify that the DNS host information is correct too.

	// Add the rack if it doesn't exist.
	if err := addRackIfApplicable(serviceAccountJSON, satlabPrefix, p); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	// Add the Asset if it doesn't exist.
	if err := addAssetIfApplicable(serviceAccountJSON, satlabPrefix, p); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	// Add the DUT if it doesn't exist.
	if err := addDUTIfApplicable(serviceAccountJSON, satlabPrefix, host, p); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}

	if err := dns.UpdateRecord(host, addr); err != nil {
		return errors.Annotate(err, "add dut").Err()
	}
	return nil
}

// AddAssetIfApplicable adds an asset to UFS if the asset does not already exist.
func addAssetIfApplicable(serviceAccountJSON string, satlabPrefix string, p *parse.CommandParseResult) error {
	for _, flag := range []string{"model", "board", "rack", "zone"} {
		if p.Flags[flag] == "" {
			return errors.New(fmt.Sprintf("add asset if applicable: required flag %q is not present", flag))
		}
	}

	args := (&commands.CommandWithFlags{
		Commands:       []string{paths.ShivasPath, "get", "asset"},
		PositionalArgs: []string{p.Flags["asset"]},
		Flags: map[string][]string{
			"json": nil,
		},
	}).ToCommand()
	command := exec.Command(args[0], args[1:]...)
	command.Stderr = os.Stderr
	assetMsg, err := command.Output()
	if err != nil {
		return errors.Annotate(err, "add asset if applicable").Err()
	}

	if len(assetMsg) == 0 {
		// Add the asset.
		fmt.Fprintf(os.Stderr, "Adding asset\n")
		args := (&commands.CommandWithFlags{
			Commands: []string{paths.ShivasPath, "add", "asset"},
			Flags: map[string][]string{
				"model": {p.Flags["model"]},
				"board": {p.Flags["board"]},
				"rack":  {p.Flags["rack"]},
				"zone":  {p.Flags["zone"]},
			},
		}).ToCommand()
		command := exec.Command(args[0], args[1:]...)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return errors.Annotate(err, "add asset if applicable").Err()
		}
	} else {
		fmt.Fprintf(os.Stderr, "Asset already added\n")
	}
	return nil
}

// AddRackIfApplicable adds a rack to UFS if the asset does not already exist.
func addRackIfApplicable(serviceAccountJSON string, satlabPrefix string, p *parse.CommandParseResult) error {
	// TODO(gregorynisbet): Generate a rack name.
	for _, flag := range []string{"rack", "namespace"} {
		if p.Flags[flag] == "" {
			return errors.New(fmt.Sprintf("add asset if applicable: required flag %q is not present", flag))
		}
	}

	args := (&commands.CommandWithFlags{
		Commands:       []string{paths.ShivasPath, "get", "rack"},
		PositionalArgs: []string{p.Flags["rack"]},
		Flags: map[string][]string{
			"json": nil,
		},
	}).ToCommand()
	command := exec.Command(args[0], args[1:]...)
	command.Stderr = os.Stderr
	rackMsg, err := command.Output()
	if err != nil {
		return errors.Annotate(err, "add rack if applicable").Err()
	}

	if len(rackMsg) == 0 {
		fmt.Fprintf(os.Stderr, "Adding rack\n")
		args := (&commands.CommandWithFlags{
			Commands: []string{paths.ShivasPath, "add", "rack"},
			Flags: map[string][]string{
				"namespace": {p.Flags["namespace"]},
				"name":      {fmt.Sprintf("%s-%s", satlabPrefix, p.Flags["rack"])},
			},
		}).ToCommand()
		command := exec.Command(args[0], args[1:]...)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return errors.Annotate(err, "add rack if applicable").Err()
		}
	} else {
		fmt.Fprintf(os.Stderr, "Rack already added\n")
	}
	return nil
}

// AddDUTIfApplicable adds a DUT to UFS if the asset does not already exist.
func addDUTIfApplicable(serviceAccountJSON string, satlabPrefix string, host string, p *parse.CommandParseResult) error {
	args := (&commands.CommandWithFlags{
		Commands: []string{paths.ShivasPath, "get", "dut"},
		Flags: map[string][]string{
			"namespace": {p.Flags["namespace"]},
			"zone":      {p.Flags["zone"]},
		},
		PositionalArgs: []string{fmt.Sprintf("%s-%s", satlabPrefix, p.PositionalArgs[0])},
	}).ToCommand()
	command := exec.Command(args[0], args[1:]...)
	command.Stderr = os.Stderr
	dutMsg, err := command.Output()
	if err != nil {
		return errors.Annotate(err, "add dut if applicable: running %s", strings.Join(args, " ")).Err()
	}
	if len(dutMsg) == 0 {
		fmt.Fprintf(os.Stderr, "Adding DUT\n")
		flags := make(map[string][]string)
		for k, v := range p.Flags {
			flags[k] = []string{v}
		}
		for k := range p.NullaryFlags {
			flags[k] = nil
		}
		flags["name"] = []string{
			fmt.Sprintf("%s-%s", satlabPrefix, flags["name"]),
		}

		args := (&commands.CommandWithFlags{
			Commands: []string{paths.ShivasPath, "add", "dut"},
			Flags:    flags,
		}).ApplyFlagFilter(true, map[string]bool{
			"model":   false,
			"board":   false,
			"rack":    false,
			"address": false,
		}).ToCommand()
		command := exec.Command(args[0], args[1:]...)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return errors.Annotate(
				err,
				fmt.Sprintf(
					"add dut if applicable: running %s",
					strings.Join(args, " "),
				),
			).Err()
		}
	} else {
		fmt.Fprintf(os.Stderr, "DUT already added\n")
	}
	return nil
}
