// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import "go.chromium.org/luci/common/errors"

// AnnotateEach annotates each error in a multierror.
func AnnotateEach(imerr errors.MultiError, fmt string, args ...interface{}) errors.MultiError {
	var merr errors.MultiError
	for _, err := range imerr {
		merr = append(merr, errors.Annotate(err, fmt, args...).Err())
	}
	return merr
}
