// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"strings"
)

const (
	defaultPageSize int32 = 100
	maxPageSize     int32 = 1000
)

const separator string = "/"

// GetPageSize gets the correct page size for List pagination
func GetPageSize(pageSize int32) int32 {
	switch {
	case pageSize == 0:
		return defaultPageSize
	case pageSize > maxPageSize:
		return maxPageSize
	default:
		return pageSize
	}
}

// RemovePrefix extracts string appearing after a "/"
func RemovePrefix(name string) string {
	// Get substring after a string.
	name = strings.TrimSpace(name)
	pos := strings.Index(name, separator)
	if pos == -1 {
		return name
	}
	adjustedPos := pos + len(separator)
	if adjustedPos >= len(name) {
		return name
	}
	return name[adjustedPos:]
}

// AddPrefix concats 2 strings along with a "/" in between
func AddPrefix(collection string, entity string) string {
	return collection + separator + entity
}
