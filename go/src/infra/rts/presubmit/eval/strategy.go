// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"context"

	"infra/rts"

	evalpb "infra/rts/presubmit/eval/proto"
)

// Strategy evaluates how much a given test is affected by given changed files.
type Strategy func(context.Context, Input, *Output) error

// Input is input to a selection strategy.
type Input struct {
	// ChangedFiles is a list of changed files.
	ChangedFiles []*evalpb.SourceFile

	// The strategy needs to decide how much each of these test variants is
	// affected by the changed files.
	TestVariants []*evalpb.TestVariant
}

// ensureChangedFilesInclude ensures that in.ChangedFiles includes all changed
// files in all of the patchsets.
func (in *Input) ensureChangedFilesInclude(pss ...*evalpb.GerritPatchset) {
	type key struct {
		repo, path string
	}
	set := map[key]struct{}{}
	for _, f := range in.ChangedFiles {
		set[key{repo: f.Repo, path: f.Path}] = struct{}{}
	}
	for _, ps := range pss {
		for _, f := range ps.ChangedFiles {
			k := key{repo: f.Repo, path: f.Path}
			if _, ok := set[k]; !ok {
				set[k] = struct{}{}
				in.ChangedFiles = append(in.ChangedFiles, f)
			}
		}
	}
}

// Output is the output of a selection strategy.
type Output struct {
	// TestVariantAffectedness is how much Input.TestVariants are affected by the
	// code change, where TestVariantAffectedness[i]
	// corresponds to Input.TestVariants[i].
	//
	// When Strategy() is called, TestVariantAffectedness is pre-initialized
	// with a slice with the same length as Input.TestVariants, and zero elements.
	// Thus by default all tests are very affected (distance=0).
	TestVariantAffectedness []rts.Affectedness
}
