// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"flag"
	"strings"
)

// CSVStringFlag is a flag.Getter implementation representing a []string.
type CSVStringFlag []string

// String returns a comma-separated string representation of the flag values.
func (f CSVStringFlag) String() string {
	return strings.Join(f, ", ")
}

// Set records seeing a flag value.
func (f *CSVStringFlag) Set(val string) error {
	// Split the values if they contain a comma
	if strings.Contains(val, ",") {
		*f = append(*f, strings.Split(val, ",")...)
	} else {
		*f = append(*f, val)
	}
	return nil
}

// Get retrieves the flag value.
func (f CSVStringFlag) Get() interface{} {
	return []string(f)
}

// CSVString returns a flag.Getter which reads flags into the given []string pointer.
func CSVString(s *[]string) flag.Getter {
	return (*CSVStringFlag)(s)
}
