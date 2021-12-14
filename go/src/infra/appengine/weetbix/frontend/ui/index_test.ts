// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview
 * This file serves as a single entry to all the test files.
 * It can
 * 1. reduce the time to compile all test bundles, and
 * 2. allow us to use relative source map file path (karma gets confused when
 * trying to map a file that is not at the project root).
 */

// require all modules ending in "_test" from the
// src directory and all subdirectories
const testsContext = require.context('./src', true, /_test\.ts$/);

testsContext.keys().forEach(testsContext);
