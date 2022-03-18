// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

// SuspectJustification represents the heuristic analysis of a CL.
// It how likely the suspect is the real culprit and also the reason for suspecting.
type SuspectJustification struct {
	IsNonBlamable bool
	Items         []*SuspectJustificationItem
}

// SuspectJustificationItem represents one item of SuspectJustification
type SuspectJustificationItem struct {
	Score    int
	FilePath string
	Reason   string
}

func (justification *SuspectJustification) GetScore() int {
	score := 0
	for _, item := range justification.Items {
		score += item.Score
	}
	return score
}

func (justification *SuspectJustification) GetReasons() string {
	if justification.IsNonBlamable {
		return "The author is non-blamable"
	}
	reasons := ""
	for _, item := range justification.Items {
		reasons = reasons + "\n" + item.Reason
	}
	return reasons
}

func (justification *SuspectJustification) AddItem(score int, filePath string, reason string) {
	item := &SuspectJustificationItem{
		Score:    score,
		FilePath: filePath,
		Reason:   reason,
	}
	if justification.Items == nil {
		justification.Items = []*SuspectJustificationItem{}
	}
	justification.Items = append(justification.Items, item)
}
