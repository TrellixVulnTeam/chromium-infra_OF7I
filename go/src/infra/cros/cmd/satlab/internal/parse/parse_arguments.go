// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package parse

import (
	"errors"
	"strings"
)

// ArgumentParseResult is the result of parsing arguments.
// TODO(gregorynisbet): Merge this with CommandParseResult below.
type ArgumentParseResult struct {
	PositionalArgs []string
	NullaryFlags   map[string]bool
	Flags          map[string]string
}

// CommandParseResult is similar to the result of parsing arguments, but has subcommand
// positional arguments split off from the rest.
type CommandParseResult struct {
	Commands       []string
	PositionalArgs []string
	NullaryFlags   map[string]bool
	Flags          map[string]string
}

const (
	// DefaultArity is the default arity of a command if no arity is
	// provided.
	defaultArity = 2
)

// A FlagKind classifies a command line argument as a flag with an argument,
// a flag without an argument, or a non-flag.
type FlagKind int

const (
	// NonFlag is a command line parameter that is not a flag.
	nonFlag FlagKind = iota
	// NullaryFlag is a flag without an argument.
	nullaryFlag
	// UnaryFlag is a flag with an argument.
	unaryFlag
)

// LooksLikeFlag returns a boolean indicating whether the given string appears to be a flag or not.
func looksLikeFlag(flag string) bool {
	return strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--")
}

// ClassifyFlag determines what type of flag a flag is.
func classifyFlag(args []string, idx int, knownNullaryFlags map[string]bool) FlagKind {
	// If we're past the end, we are not a nullary flag.
	if idx >= len(args) {
		return nonFlag
	}

	if knownNullaryFlags == nil {
		knownNullaryFlags = make(map[string]bool)
	}
	arg := args[idx]
	// The next argument could be after the end of the slice.
	nextArg := ""
	if idx+1 < len(args) {
		nextArg = args[idx+1]
	}
	argContent := strings.TrimPrefix(arg, "-")

	// If we don't look like a flag, we are not a flag.
	if !looksLikeFlag(arg) {
		return nonFlag
	}

	// If the content of the flag is known to be nullary, then
	// we force the interpretation of the flag as nullary.
	if present := knownNullaryFlags[argContent]; present {
		return nullaryFlag
	}

	// If the next argument looks like a flag, assume that the
	// current flag is nullary.
	if looksLikeFlag(nextArg) {
		return nullaryFlag
	}

	// By default, we are a non-nullary flag.
	return unaryFlag
}

// ParseSingleArgument is a helper function that processes a single argument and updates the current argument state.
// ParseSingleArgument returns (0, nil) when done parsing.
// ParseSingleArgument returns (n, nil) for some positive n when it has successfully consumed an argument.
// ParseSingleArgument returns (0, err) for some non-nil error when it has encountered an error and cannot proceed.
//
// TODO(gregorynisbet): Fix the API of this function so it doesn't have knownNullaryFlags.
//
func parseSingleArgument(
	result *ArgumentParseResult,
	args []string,
	idx int,
	knownNullaryFlags map[string]bool,
) (int, error) {
	if idx >= len(args) {
		return 0, nil
	}
	arg := args[idx]
	argContent := strings.TrimPrefix(arg, "-")
	classification := classifyFlag(args, idx, knownNullaryFlags)

	switch classification {
	case nonFlag:
		result.PositionalArgs = append(result.PositionalArgs, arg)
		return 1, nil
	case nullaryFlag:
		result.NullaryFlags[argContent] = true
		return 1, nil
	case unaryFlag:
		nextArg := ""
		if idx+1 < len(args) {
			nextArg = args[idx+1]
		}
		result.Flags[argContent] = nextArg
		// This position and the next position are
		// consumed
		return 2, nil
	default:
		return 0, errors.New("unhandled argument classification")
	}
}

// ParseArguments walks the arguments given to the satlab command and determines what type each of them are.
func parseArguments(args []string, knownNullaryFlags map[string]bool) (*ArgumentParseResult, error) {
	var positionalArgs []string
	nullaryFlags := make(map[string]bool)
	flags := make(map[string]string)
	result := &ArgumentParseResult{
		PositionalArgs: positionalArgs,
		NullaryFlags:   nullaryFlags,
		Flags:          flags,
	}

	i := 0
	for {
		delta, err := parseSingleArgument(
			result,
			args,
			i,
			knownNullaryFlags,
		)
		if err != nil {
			return nil, err
		}
		if delta == 0 {
			break
		}
		if delta < 0 {
			return nil, errors.New("refusing to backtrack")
		}
		i += delta
	}
	return result, nil
}

// ParseCommand takes an argument string, a map of command arities, and known nullary flags and parses out
// individual flags and subcommands.
func ParseCommand(args []string, knownCommandArity map[string]int, knownNullaryFlags map[string]bool) (*CommandParseResult, error) {
	if knownCommandArity == nil {
		knownCommandArity = make(map[string]int)
	}
	if knownNullaryFlags == nil {
		knownNullaryFlags = make(map[string]bool)
	}
	p, err := parseArguments(args, knownNullaryFlags)
	if err != nil {
		return nil, err
	}
	if len(p.PositionalArgs) == 0 {
		return nil, errors.New("command has no arguments")
	}
	arity, ok := knownCommandArity[p.PositionalArgs[0]]
	// Assume that an unrecognized command has the default arity.
	if !ok {
		arity = defaultArity
	}
	if arity <= 0 {
		return nil, errors.New("arity must be positive")
	}
	switch arity {
	case 1, 2:
		return &CommandParseResult{
			Commands:       p.PositionalArgs[:arity],
			PositionalArgs: p.PositionalArgs[arity:],
			NullaryFlags:   p.NullaryFlags,
			Flags:          p.Flags,
		}, nil
	}
	return nil, errors.New("arity of command is greater than number of supplied parameters")
}
