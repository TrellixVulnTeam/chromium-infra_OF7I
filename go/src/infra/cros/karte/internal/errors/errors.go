// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package errors

import (
	"fmt"

	"go.chromium.org/luci/common/errors"
)

// An annotator is the result of wrapping an error via annotate.
type Annotator interface {
	Err() error
}

// New creates a new error with the given tags.
// TODO(gregorynisbet): Consider replacing with calls to Reason exclusively.
func New(msg string, tags ...errors.TagValueGenerator) error {
	return errors.New(msg, tags...)
}

// Annotate annotates an error.
func Annotate(err error, reason string, args ...interface{}) Annotator {
	return errors.Annotate(err, reason, args...)
}

// Errorf creates a new error given a format string.
func Errorf(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}
