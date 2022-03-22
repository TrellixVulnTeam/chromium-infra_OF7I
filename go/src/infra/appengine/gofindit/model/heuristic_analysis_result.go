// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

import "sort"

type HeuristicAnalysisResult struct {
	// A slice of possible culprit, sorted by score descendingly
	Items []*HeuristicAnalysisResultItem
}

type HeuristicAnalysisResultItem struct {
	Commit        string
	ReviewUrl     string
	Justification *SuspectJustification
}

// AddItem adds a suspect to HeuristicAnalysisResult.
func (r *HeuristicAnalysisResult) AddItem(commit string, reviewUrl string, justification *SuspectJustification) {
	item := &HeuristicAnalysisResultItem{
		Commit:        commit,
		ReviewUrl:     reviewUrl,
		Justification: justification,
	}
	r.Items = append(r.Items, item)
}

// Sort items descendingly based on score (CLs with higher possibility to be
// culprit will come first).
func (r *HeuristicAnalysisResult) Sort() {
	sort.Slice(r.Items, func(i, j int) bool {
		return r.Items[i].Justification.GetScore() > r.Items[j].Justification.GetScore()
	})
}
