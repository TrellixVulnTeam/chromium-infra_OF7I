// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dns

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestReplaceLineContents tests replacing lines in a DNS hosts file.
func TestReplaceLineContents(t *testing.T) {
	t.Parallel()

	expected := []string{
		tabify("addr1-NEW host1"),
		tabify("addr2-NEW host2"),
		tabify("addr3 host3"),
	}

	newRecords := map[string]string{
		"host1": "addr1-NEW",
		"host2": "addr2-NEW",
	}
	classifier := makeClassifier(newRecords)

	replacer := func(line string) string {
		words := strings.Fields(line)
		if len(words) < 2 {
			return ""
		}
		return fmt.Sprintf("%s\t%s", newRecords[words[1]], words[1])
	}

	input := []string{
		tabify("addr1 host1"),
		tabify("addr2 host2"),
		tabify("addr3 host3"),
	}
	actual, err := replaceLineContents(
		make(map[string]bool),
		input,
		classifier,
		replacer,
	)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}

// Tabify replaces arbitrary whitespace with tabs.
func tabify(s string) string {
	return strings.Join(strings.Fields(s), "\t")
}
