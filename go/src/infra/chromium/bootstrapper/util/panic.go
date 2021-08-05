// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"fmt"
	"os/exec"
)

// PanicIf will panic with a custom error if a condition is true.
//
// In fakes and tests, it is often desirable to panic when some precondition is
// violated to provide a loud indication that something wasn't set correctly.
// PanicIf allows for panicking in those situations without having to introduce
// uncovered conditional code and/or adding tests to cover those situations.
//
// This should be limited to test code and fakes. The code for the actual binary
// should use proper error handling.
func PanicIf(condition bool, message string, args ...interface{}) {
	if condition {
		panic(fmt.Errorf(message, args...))
	}
}

// PanicOnError will panic if err is not nil.
//
// Specific error types will have additional information added to the panic e.g.
// the stderr of a failed process will be included in the panic in the case of
// *exec.ExitError.
//
// In fakes and tests, it is often desirable to panic rather than returning an
// error (e.g. the fake data is in an inconsistent state or a function should
// always succeed) because the situation isn't one where the errors should be
// handled. PanicOnError allows for panicking in those situations without having
// to introduce uncovered conditional code and/or adding tests to cover those
// situations.
//
// This should be limited to test code and fakes. The code for the actual binary
// should use proper error handling.
func PanicOnError(err error) {
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			panic(fmt.Errorf("%w\nstderr:\n%s", err, string(err.Stderr)))
		}
		panic(err)
	}
}
