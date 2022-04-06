// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"strconv"
	"strings"
	"time"

	"infra/cros/recovery/internal/log"
)

const (
	// This character separates the name and values for extra
	// arguments defined for actions.
	DefaultSplitter = ":"

	// This character demarcates the individual values among
	// multi-valued extra arguments defined for actions.
	MultiValueSplitter = ","
)

// The map representing key-value pairs parsed from extra args in the
// configuration.
type ParsedArgs map[string]string

// AsBool returns the value for the passed key as a boolean. If the
// key does not exist in the parsed arguments, the passed defaultValue
// is returned.
func (parsedArgs ParsedArgs) AsBool(ctx context.Context, key string, defaultValue bool) bool {
	if value, ok := parsedArgs[key]; ok {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		log.Debugf(ctx, "Parsed Args As Bool: value %q for key %q is not a valid boolean, returning default value %t.", value, key, defaultValue)
	} else {
		log.Debugf(ctx, "Parsed Args As Bool: key %q does not exist in the parsed arguments, returning default value %t.", key, defaultValue)
	}
	return defaultValue
}

// AsString returns the value for the passed key as a string.
// If the key does not exist in the parsed arguments, the passed defaultValue
// is returned.
func (parsedArgs ParsedArgs) AsString(ctx context.Context, key, defaultValue string) string {
	if value, ok := parsedArgs[key]; ok {
		log.Debugf(ctx, "Parsed Args As String: value %q found for key %q", value, key)
		return value
	}
	log.Debugf(ctx, "Parsed Args As String: key %q not found, default value of empty string returned", key)
	return defaultValue
}

// AsStringSlice returns the value for the passed key as a slice of string.
// If the key does not exist in the parsed arguments, an empty slice is returned.
func (parsedArgs ParsedArgs) AsStringSlice(ctx context.Context, key string, defaultValue []string) []string {
	value := parsedArgs.AsString(ctx, key, "")
	if len(value) > 0 {
		return strings.Split(value, MultiValueSplitter)
	}
	return defaultValue
}

// AsInt returns the value for the passed key as a int.
// If the value cannot be interpreted as int, then the passed defaultValue is returned.
func (parsedArgs ParsedArgs) AsInt(ctx context.Context, key string, defaultValue int) int {
	if value, ok := parsedArgs[key]; ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Debugf(ctx, "Parsed Args As int: value %q for key %q is not a valid integer, returning default value %d.", value, key, defaultValue)
	} else {
		log.Debugf(ctx, "Parsed Args As int: key %q does not exist in the parsed arguments, returning default value %d.", key, defaultValue)
	}
	return defaultValue
}

// AsFloat64 returns the value for the passed key as a float64.
// If the value cannot be interpreted as int, then the passed defaultValue is returned.
func (parsedArgs ParsedArgs) AsFloat64(ctx context.Context, key string, defaultValue float64) float64 {
	if value, ok := parsedArgs[key]; ok {
		if val, err := strconv.ParseFloat(value, 64); err == nil {
			return val
		}
		log.Debugf(ctx, "Parsed Args As Float64: value %q for key %q is not a valid integer, returning default value %d.", value, key, defaultValue)
	} else {
		log.Debugf(ctx, "Parsed Args As Float64: key %q does not exist in the parsed arguments, returning default value %d.", key, defaultValue)
	}
	return defaultValue
}

// AsDuration returns the value of the passed key as type: time.Duration.
// If the value cannot be interpreted as int, then the passed defaultValue is returned.
//
// @params unit: the unit of the time Duration, can be Nanosecond, Mircrosecond, Millisecond, Second, Minute
func (parsedArgs ParsedArgs) AsDuration(ctx context.Context, key string, defaultValue int, unit time.Duration) time.Duration {
	defaultDuration := time.Duration(defaultValue) * unit
	if value, ok := parsedArgs[key]; ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return time.Duration(intVal) * unit
		}
		log.Debugf(ctx, "Parsed Args As duration: value %q for key %q is not a valid integer, returning default duration %v.", value, key, defaultDuration)
	} else {
		log.Debugf(ctx, "Parsed Args As duration: key %q does not exist in the parsed arguments, returning default duration %v.", key, defaultDuration)
	}
	return defaultDuration
}

// ParseActionArgs returns parsed action arguments with default splitter.
func (ei *ExecInfo) GetActionArgs(ctx context.Context) ParsedArgs {
	return ParseActionArgs(ctx, ei.ActionArgs, DefaultSplitter)
}

// ParseActionArgs parses the action arguments using the splitter, and
// returns ParsedArgs object containing key and values in the action
// arguments. If any mal-formed action arguments are found their value
// is set to empty string.
func ParseActionArgs(ctx context.Context, actionArgs []string, splitter string) ParsedArgs {
	parsedArgs := ParsedArgs(make(map[string]string))
	for _, a := range actionArgs {
		a := strings.TrimSpace(a)
		if a == "" {
			continue
		}
		log.Debugf(ctx, "Parse Action Args: action arg %q", a)
		i := strings.Index(a, splitter)
		// Separator has to be at least second letter in the string to provide one letter key.
		if i < 1 {
			log.Debugf(ctx, "Parse Action Args: malformed action arg %q", a)
			parsedArgs[a] = ""
		} else {
			k := strings.TrimSpace(a[:i])
			v := strings.TrimSpace(a[i+1:])
			log.Debugf(ctx, "Parse Action Args: k: %q, v: %q", k, v)
			parsedArgs[k] = v
		}
	}
	return parsedArgs
}
