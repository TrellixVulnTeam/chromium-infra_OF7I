// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import assetReducer, {
  fetchAssetAsync,
  AssetState
} from './assetSlice';

describe('asset reducer', () => {
  const initialState: AssetState = {
    name: '',
    assetId: '',
    status: 'idle',
  };
  it('should handle initial state', () => {
    expect(assetReducer(undefined, { type: 'unknown' })).toEqual({
      name: '',
      assetId: '',
      status: 'idle',
    });
  });
});
