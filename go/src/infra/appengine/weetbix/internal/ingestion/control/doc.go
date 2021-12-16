// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package control provides methods to read and write records used to:
// 1. Ensure exactly-once ingestion of test results from builds.
// 2. Synchronise build completion and presubmit run completion, so that
//    ingestion only proceeds when both build and presubmit run have
//    completed.
package control
