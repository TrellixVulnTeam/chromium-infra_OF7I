// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package util contains common utility functions.
package util

import (
	"reflect"
)

// UnorderedEqual checks that the two arrays contain the same elements, but
// they don't have to be the same order.
func UnorderedEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]int)
	for _, v := range a {
		am[v]++
	}
	bm := make(map[string]int)
	for _, v := range b {
		bm[v]++
	}
	return reflect.DeepEqual(am, bm)
}

// UnorderedContains checks that arr has certain elements.
func UnorderedContains(arr, has []string) bool {
	elts := make(map[string]int)
	for _, v := range arr {
		elts[v]++
	}
	for _, elt := range has {
		_, ok := elts[elt]
		if !ok {
			return false
		}
	}
	return true
}
