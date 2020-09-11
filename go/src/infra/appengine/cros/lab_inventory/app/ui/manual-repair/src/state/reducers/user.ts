// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * Synchronous Redux actions to update state store.
 */
export const RECEIVE_USER = 'RECEIVE_USER';

export function receiveUser(user: object) {
  return {type: RECEIVE_USER, user};
};

export type UserStateType = {
  signedIn: boolean,
  profile: object,
  authHeaders: object,
}

const emptyState: UserStateType = {
  signedIn: false,
  profile: {},
  authHeaders: {},
};

export function userReducer(state = emptyState, action) {
  switch (action.type) {
    case RECEIVE_USER:
      return {
        signedIn: action.user.signedIn,
        profile: action.user.profile,
        authHeaders: action.user.authHeaders,
      };
    default:
      return state;
  };
};
