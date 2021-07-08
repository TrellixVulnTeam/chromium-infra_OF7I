// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build go1.12

package main

// NOTE: If we uncomment this import, "nope" package *will* end up in the
// outputs even though newergo.go itself is skipped. It is because `go list ...`
// used by packages.Load(...) always uses the current Go version to filter files
// before exploring their dependencies, and thus it will "explore" dependencies
// of newergo.go, even though cloudbuildhelper assembles a tarball targeting
// go1.11. There appears to be no easy way to discard such dependencies.
//
// import "infra/cmd/cloudbuildhelper/builder/testing/nope"
