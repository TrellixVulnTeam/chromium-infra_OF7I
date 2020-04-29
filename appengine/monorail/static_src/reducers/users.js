// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview User actions, selectors, and reducers organized into
 * a single Redux "Duck" that manages updating and retrieving user state
 * on the frontend.
 *
 * The User data is stored in a normalized format.
 * `users` stores all User data indexed by User name.
 * `user` is a selector that gets the currently viewed User data.
 *
 * Reference: https://github.com/erikras/ducks-modular-redux
 */

import {combineReducers} from 'redux';
import {createReducer, createKeyedRequestReducer} from './redux-helpers.js';
import {prpcClient} from 'prpc-client-instance.js';
import 'shared/typedef.js';

/** @typedef {import('redux').AnyAction} AnyAction */

// Actions
export const BATCH_GET_START = 'user/BATCH_GET_START';
export const BATCH_GET_SUCCESS = 'user/BATCH_GET_SUCCESS';
export const BATCH_GET_FAILURE = 'user/BATCH_GET_FAILURE';

/* State Shape
{
  byName: Object<string, User>,

  requests: {
    batchGet: ReduxRequestState,
  },
}
*/

// Reducers

/**
 * All User data indexed by User name.
 * @param {Object<string, User>} state The existing User data.
 * @param {AnyAction} action
 * @param {User} action.user The user that was fetched.
 * @return {Object<string, User>}
 */
export const byNameReducer = createReducer({}, {
  [BATCH_GET_SUCCESS]: (state, {users}) => {
    const newState = {...state};
    for (const user of users) {
      newState[user.name] = user;
    }
    return newState;
  },
});

const requestsReducer = combineReducers({
  batchGet: createKeyedRequestReducer(
      BATCH_GET_START, BATCH_GET_SUCCESS, BATCH_GET_FAILURE),
});

export const reducer = combineReducers({
  byName: byNameReducer,

  requests: requestsReducer,
});

// Selectors

/**
 * Returns all the User data in the store as a mapping from name to User.
 * @param {any} state
 * @return {Object<string, User>}
 */
export const byName = (state) => state.users.byName;

// Action Creators

/**
 * Action creator to fetch multiple User objects.
 * @param {Array<string>} names The names of the Users to fetch.
 * @return {function(function): Promise<void>}
 */
export const batchGet = (names) => async (dispatch) => {
  dispatch({type: BATCH_GET_START});

  try {
    /** @type {{users: Array<User>}} */
    const {users} = await prpcClient.call(
        'monorail.v1.Users', 'BatchGetUsers', {names});

    dispatch({type: BATCH_GET_SUCCESS, users});
  } catch (error) {
    dispatch({type: BATCH_GET_FAILURE, error});
  };
};
