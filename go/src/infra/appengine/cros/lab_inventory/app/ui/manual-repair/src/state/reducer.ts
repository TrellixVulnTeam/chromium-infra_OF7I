// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {RECEIVE_RECORDS} from './actions';

const INITIAL_STATE = {
  records: []
};

export const reducer = (state = INITIAL_STATE, action) => {
  switch (action.type) {
    case RECEIVE_RECORDS:
      return {
        ...state, records: action.records
      }
    default:
      return state;
  }
}
