// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * Synchronous Redux actions to update state store.
 */
export const RECEIVE_APP_MESSAGE = 'RECEIVE_APP_MESSAGE';
export const CLEAR_APP_MESSAGE = 'CLEAR_APP_MESSAGE';

export function receiveAppMessage(applicationMessage: string) {
  return {type: RECEIVE_APP_MESSAGE, applicationMessage};
};
export function clearAppMessage() {
  return {type: CLEAR_APP_MESSAGE};
};

export type MessageStateType = {
  applicationMessage: string,
}

const emptyState = {
  applicationMessage: '',
};

/**
 * Reducer for global messages. Other state slices may contain slice specific
 * errors.
 */
export function messageReducer(state = emptyState, action) {
  switch (action.type) {
    case RECEIVE_APP_MESSAGE:
      return {
        ...state, applicationMessage: action.applicationMessage,
      }
    case CLEAR_APP_MESSAGE:
      return {
        ...state, applicationMessage: emptyState.applicationMessage,
      }
    default:
      return state;
  };
};
