// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

// ErrToString enables safe conversions of errors to
// strings. Returns an empty string for nil errors.
func ErrToString(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
