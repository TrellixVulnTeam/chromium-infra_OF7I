// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { TodayOutlined } from '@material-ui/icons';
import dataSourceReducer, { actions } from './dataSourcesSlice';

// Tests the updateCurrent reducer
describe('updateCurrent', () => {
  it('updates current', () => {
    const action = actions.updateCurrent('test');
    const state = dataSourceReducer(undefined, action);

    expect(state.current).toEqual(action.payload);
  });
});

// TODO(gatong): Add more tests here.
