// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * Synchronous Redux actions to update state store.
 */
export const RECEIVE_QUERY_STORE = 'RECEIVE_QUERY_STORE';
export const CLEAR_QUERY_STORE = 'CLEAR_QUERY_STORE';

export function receiveQueryStore(queryStore: object) {
  return {type: RECEIVE_QUERY_STORE, queryStore};
};
export function clearQueryStore() {
  return {type: CLEAR_QUERY_STORE};
};

export type QueryStoreStateType = {};

const emptyState = {};

/**
 * Reducer for query parameters parsed from query string.
 */
export function queryReducer(state = emptyState, action) {
  switch (action.type) {
    case RECEIVE_QUERY_STORE:
      return action.queryStore;
    case CLEAR_QUERY_STORE:
      return emptyState;
    default:
      return state;
  };
};
