// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package querygs

import (
	"fmt"
	"sync"

	"go.chromium.org/luci/common/gcloud/gs"
)

// maxLookbehind is the number of milestones to look behind in order to find a path to the
// firmware bundle
const maxLookbehind = 40

const maxLookahead = 5

// FindFirmwarePathResult is the result of a search in Google Storage.
type FindFirmwarePathResult struct {
	Image    string
	FullPath gs.Path
}

// milestonesInOrder returns the milestones to be checked for the presence of a firmware bundle
// in order of priority.
// The current milestone isconsidered first, then milestones in increasing order, then
// milestones in decreasing order.
func milestonesInOrder(milestone int) []int {
	out := []int{milestone}
	for i := milestone + 1; i <= milestone+maxLookahead; i++ {
		out = append(out, i)
	}
	for i := milestone - 1; i > 0 && i >= milestone-maxLookbehind; i-- {
		out = append(out, i)
	}
	return out
}

// FindFirmwarePath finds the latest milestone associated with a given firmware image.
func (r *Reader) FindFirmwarePath(board string, milestone int, tip int, branch int, branchBranch string) (*FindFirmwarePathResult, error) {
	var candidates []gs.Path
	var images []string
	milestones := milestonesInOrder(milestone)
	for _, m := range milestones {
		for _, releaseKind := range []string{"release", "firmware"} {
			candidates = append(candidates, gs.Path(fmt.Sprintf("gs://chromeos-image-archive/%s-%s/R%d-%d.%d.%s/firmware_from_source.tar.bz2", board, releaseKind, m, tip, branch, branchBranch)))
			images = append(images, fmt.Sprintf("%s-%s/R%d-%d.%d.%s", board, releaseKind, m, tip, branch, branchBranch))
		}
	}

	successfulCandidates := make([]gs.Path, len(candidates))
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(successfulCandidates))
	for i, candidate := range candidates {
		go func(i int, candidate gs.Path) {
			if err := r.RemoteFileExists(candidate); err == nil {
				successfulCandidates[i] = candidate
			}
			waitGroup.Done()
		}(i, candidate)
	}
	waitGroup.Wait()

	for i, candidate := range successfulCandidates {
		if candidate != "" {
			return &FindFirmwarePathResult{images[i], candidate}, nil
		}
	}

	firstCand := ""
	if len(candidates) != 0 {
		firstCand = string(candidates[0])
	}
	return nil, fmt.Errorf("no gspaths found starting with %q", firstCand)
}
