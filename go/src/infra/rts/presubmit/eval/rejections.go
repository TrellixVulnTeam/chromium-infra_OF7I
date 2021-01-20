// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"sort"
	"strings"

	"infra/rts"
	evalpb "infra/rts/presubmit/eval/proto"
)

type rejectionPrinter struct {
	*printer
}

// printRejection prints a rejection.
func (p *rejectionPrinter) rejection(rej *evalpb.Rejection, mostAffected rts.Affectedness) error {
	pf := p.printf

	pf("Rejection:\n")
	p.Level++

	pf("Most affected test: %f distance, %d rank\n", mostAffected.Distance, mostAffected.Rank)

	// Print patchsets.
	if len(rej.Patchsets) == 1 {
		p.patchset(rej.Patchsets[0])
	} else {
		pf("- patchsets:\n")
		p.Level++
		for _, ps := range rej.Patchsets {
			p.patchset(ps)
		}
		p.Level--
	}

	p.testVariant(rej.FailedTestVariants)

	p.Level--
	return p.err
}

func (p *rejectionPrinter) patchset(ps *evalpb.GerritPatchset) {
	p.printf("%s\n", psURL(ps))

	paths := make([]string, len(ps.ChangedFiles))
	for i, f := range ps.ChangedFiles {
		paths[i] = f.Path
	}
	sort.Strings(paths)

	p.Level++
	for _, f := range paths {
		p.printf("%s\n", f)
	}
	p.Level--
}

// printTestVariants prints tests grouped by variant.
func (p *rejectionPrinter) testVariant(testVariants []*evalpb.TestVariant) {
	pf := p.printf

	// Group by variant.
	byVariant := map[string][]*evalpb.TestVariant{}
	var keys []string
	for _, tv := range testVariants {
		sort.Strings(tv.Variant) // a side effect, but innocuous.
		key := strings.Join(tv.Variant, " | ")
		tests, ok := byVariant[key]
		byVariant[key] = append(tests, tv)
		if !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	// Print tests grouped by variant.
	pf("Failed and not selected tests:\n")
	p.Level++
	for _, key := range keys {
		pf("- ")
		if key == "" {
			pf("<empty test variant>\n")
		} else {
			pf("%s\n", key)
		}

		ts := byVariant[key]
		sort.Slice(ts, func(i, j int) bool {
			return ts[i].Id < ts[j].Id
		})

		byFileName := map[string][]string{}
		for _, t := range ts {
			byFileName[t.FileName] = append(byFileName[t.FileName], t.Id)
		}

		p.Level++
		for fileName, testIDs := range byFileName {
			if fileName != "" {
				p.printf("in %s\n", fileName)
			} else {
				p.printf("in <unknown file>\n")
			}
			p.Level++
			for _, id := range testIDs {
				p.printf("%s\n", id)
			}
			p.Level--
		}
		p.Level--
	}
	p.Level--
}
