// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// IsCSVFile return true if its a csv file
func IsCSVFile(filename string) bool {
	return strings.Contains(filename, ".csv")
}

//ParseMCSVFile parse a mcsv file and return the records as 2D string slice
func ParseMCSVFile(specsFile string) ([][]string, error) {
	rawText, err := ioutil.ReadFile(specsFile)
	if err != nil {
		return nil, err
	}
	text := string(rawText)
	if text == "" {
		return nil, errors.New("mcsv file cannot be empty")
	}
	reader := strings.NewReader(text)
	csvReader := csv.NewReader(reader)
	return csvReader.ReadAll()
}

// LooksLikeHeader heuristically determines whether a CSV line looks like
// a CSV header for the MCSV format.
func LooksLikeHeader(rec []string) bool {
	if len(rec) == 0 {
		return false
	}
	return strings.EqualFold(rec[0], "name")
}

// ValidateSameStringArray validates if 2 strings slice are same
func ValidateSameStringArray(expected []string, actual []string) error {
	if len(expected) != len(actual) {
		return errors.New("length mismatch")
	}
	for i, e := range expected {
		a := actual[i]
		if e != a {
			return fmt.Errorf("item mismatch at position (%d) expected (%s) got (%s)", i, e, a)
		}
	}
	return nil
}
