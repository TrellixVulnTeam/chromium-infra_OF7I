// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Test setup code that defines functionality meant to run
 * before each test.
 */

import {resetState, store} from 'reducers/base.js';

Mocha.beforeEach(() => {
  // We reset the Redux state before each test run to prevent Redux
  // state changes in previous tests from affecting results.
  store.dispatch(resetState());
});
