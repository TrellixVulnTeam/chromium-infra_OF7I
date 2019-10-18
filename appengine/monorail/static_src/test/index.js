// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Root file for running our frontend tests. Finds all files
 * in the static_src folder that have the ".test.js" extension.
 */

const testsContext = require.context('../', true, /\.test\.js$/);
testsContext.keys().forEach(testsContext);
