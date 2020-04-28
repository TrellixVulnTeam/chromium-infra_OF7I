// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package flagx

import (
	"flag"
	"fmt"
	"strings"
)

// dimsVar is a handle to leaseDutRun that implements the Value interface
// and allows the dims map to be modified.
type dimsVar struct {
	handle *map[string]string
}

// Dims takes an initial map and produces a flag variable that can be set
// from command line arguments
func Dims(m *map[string]string) flag.Value {
	if m == nil {
		panic("Argument to Dims must be non-nil pointer to map!")
	}
	return dimsVar{handle: m}
}

// String returns the default value for dimensions represented as a string.
// The default value is an empty map, which stringifies to an empty string.
func (dimsVar) String() string {
	return ""
}

// Set populates the dims map with comma-delimited key-value pairs.
// Setting the dims map always succeeds, regardless of what string is given.
func (d dimsVar) Set(newval string) error {
	if d.handle == nil {
		panic("DimsVar handle must be pointing at a map[string]string!")
	}
	if *d.handle != nil {
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
		m[key] = val
	}
	return nil
}

// splitKeyVal splits a string with "=" into two key-value pairs,
// and returns an error if this is impossible.
// Strings with multiple "=" values are considered malformed.
func splitKeyVal(s string) (string, string, error) {
	res := strings.Split(s, "=")
	switch len(res) {
	case 2:
		return res[0], res[1], nil
	default:
		return "", "", fmt.Errorf(`string %q is a malformed key-value pair`, s)
	}
}
