// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Creates a globally shared instance of AutoRefreshPrpcClient
 * to be used across the frontend, to share state and allow easy test stubbing.
 */

import AutoRefreshPrpcClient from 'prpc.js';

// TODO(crbug.com/monorail/5049): Remove usage of window.CS_env here.
export const prpcClient = new AutoRefreshPrpcClient(
  window.CS_env ? window.CS_env.token : '',
  window.CS_env ? window.CS_env.tokenExpiresSec : 0,
);
