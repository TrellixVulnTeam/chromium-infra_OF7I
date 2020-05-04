// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"infra/unifiedfleet/app/constants"
)

// GetPageSize gets the correct page size for List pagination
func GetPageSize(pageSize int32) int32 {
	switch {
	case pageSize == 0:
		return constants.DefaultPageSize
	case pageSize > constants.MaxPageSize:
		return constants.MaxPageSize
	default:
		return pageSize
	}
}
