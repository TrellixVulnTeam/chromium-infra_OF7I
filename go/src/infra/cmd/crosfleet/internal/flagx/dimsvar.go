// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package flagx

import (
	"flag"
	"fmt"
	"regexp"
	"strings"
)

// dimsVar implements the flag.Value interface. It provides a key-val flag
// handler for  `run test`, `run suite`, `run testplan`, and `dut lease`, that
// is more flexible than the generic handler options. In particular, dimsVar can
// parse single or repeated key-val flags, with each key-val separated by either
// ":" or "=".
type dimsVar struct {
	handle *map[string]string
}

// KeyVals takes an initial map and produces a flag variable that can be set
// from command line arguments
func KeyVals(m *map[string]string) flag.Value {
	if m == nil {
		panic("Argument to KeyVals must be pointing to a map[string]string!")
	}
	return dimsVar{handle: m}
}

// String returns the default value for dimensions represented as a string.
// The default value is an empty map, which stringifies to an empty string.
func (dimsVar) String() string {
	return ""
}

// Set populates the dims map with comma-delimited key-value pairs.
func (d dimsVar) Set(newval string) error {
	if d.handle == nil {
		panic("DimsVar handle must be pointing to a map[string]string!")
	}
	if *d.handle == nil {
		*d.handle = make(map[string]string)
	}
	// strings.Split, if given an empty string, will produce a
	// slice containing a single string.
	if newval == "" {
		return nil
	}
	m := *d.handle
	for _, entry := range strings.Split(newval, ",") {
		key, val, err := splitKeyVal(entry)
		if err != nil {
			return err
		}
		if _, exists := m[key]; exists {
			return fmt.Errorf("key %q is already specified", key)
		}
		m[key] = val
	}
	return nil
}

// splitKeyVal splits a string with "=" or ":" into two key-value
// pairs, and returns an error if this is impossible.
// Strings with multiple "=" or ":" values are considered malformed.
// This
func splitKeyVal(s string) (string, string, error) {
	re := regexp.MustCompile("[=:]")
	res := re.Split(s, -1)
	switch len(res) {
	case 2:
		return res[0], res[1], nil
	default:
		return "", "", fmt.Errorf(`string %q is a malformed key-value pair`, s)
	}
}
