// Copyright 202 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import labReducer, {
  fetchLabAsync,
  LabState
} from './labSlice';

describe('lab reducer', () => {
  const initialState: LabState = {
    name: '',
    labId: '',
    status: 'idle',
  };
  it('should handle initial state', () => {
    expect(labReducer(undefined, { type: 'unknown' })).toEqual({
      name: '',
      labId: '',
      status: 'idle',
    });
  });
});
