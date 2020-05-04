// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

const (
	defaultPageSize int32 = 100
	maxPageSize     int32 = 1000
)

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
