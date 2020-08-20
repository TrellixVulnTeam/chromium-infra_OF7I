// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {RECEIVE_RECORDS, RECEIVE_USER} from './actions';

const INITIAL_STATE = {
  user: {signedIn: false, profile: Object(), authHeaders: Object()},
  records: []
};

export const reducer = (state = INITIAL_STATE, action) => {
  switch (action.type) {
    case RECEIVE_USER:
      return {...state, user: action.user};
    case RECEIVE_RECORDS:
      return {...state, records: action.records};
    default:
      return state;
  }
}
