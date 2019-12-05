// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Hotlist actions, selectors, and reducers organized into
 * a single Redux "Duck" that manages updating and retrieving hotlist state
 * on the frontend.
 *
 * The Hotlist data is stored in a normalized format.
 * `hotlists` stores all Hotlist data indexed by HotlistRefString.
 * `hotlistRef` is a reference to the currently viewed Hotlist.
 * `hotlist` is a selector that gets the currently viewed Hotlist data.
 *
 * Reference: https://github.com/erikras/ducks-modular-redux
 */

import {combineReducers} from 'redux';
import {createSelector} from 'reselect';
import {userIdOrDisplayNameToUserRef} from 'shared/converters.js';
import {hotlistToRefString, hotlistRefToString}
  from 'shared/converters-hotlist.js';
import {createReducer, createRequestReducer} from './redux-helpers.js';
import {prpcClient} from 'prpc-client-instance.js';
import 'shared/typedef.js';

// Actions
export const SELECT_HOTLIST = 'hotlist/SELECT_HOTLIST';

export const FETCH_START = 'hotlist/FETCH_START';
export const FETCH_SUCCESS = 'hotlist/FETCH_SUCCESS';
export const FETCH_FAILURE = 'hotlist/FETCH_FAILURE';

/* State Shape
{
  hotlists: Object.<HotlistRefString, Hotlist>,

  hotlistRef: HotlistRef,

  requests: {
    fetch: ReduxRequestState,
  },
}
*/

// Reducers

/**
 * All Hotlist data indexed by HotlistRefString.
 * @param {Object<string, Hotlist>} state The existing mapping of Hotlist data.
 * @param {import('redux').AnyAction} action A Redux action.
 * @return {Object.<string, Hotlist>}
 */
export const hotlistsReducer = createReducer({}, {
  [FETCH_SUCCESS]: (state, action) => {
    const newState = {...state};
    newState[hotlistToRefString(action.hotlist)] = action.hotlist;
    return newState;
  },
});

/**
 * A reference to the currently viewed Hotlist.
 * @param {?Hotlist} state The existing HotlistRef.
 * @param {import('redux').AnyAction} action A Redux action.
 * @return {?Hotlist}
 */
export const hotlistRefReducer = createReducer(null, {
  [SELECT_HOTLIST]: (_state, action) => action.hotlistRef,
});

const requestsReducer = combineReducers({
  fetch: createRequestReducer(
      FETCH_START, FETCH_SUCCESS, FETCH_FAILURE),
});

export const reducer = combineReducers({
  hotlists: hotlistsReducer,
  hotlistRef: hotlistRefReducer,

  requests: requestsReducer,
});

// Selectors
/**
 * Returns all the Hotlist data in the store as
 * a mapping of HotlistRef string to Hotlist.
 * @param {any} state The Redux store.
 * @return {Object.<string, Hotlist>}
 */
export const hotlists = (state) => state.hotlist.hotlists;

/**
 * Returns the currently viewed HotlistRef, or null if there is none.
 * @param {any} state The Redux store.
 * @return {?HotlistRef}
 */
export const hotlistRef = (state) => state.hotlist.hotlistRef;

/**
 * Returns the currently viewed Hotlist, or null if there is none.
 * @param {any} state The Redux store.
 * @return {?Hotlist}
 */
export const hotlist = createSelector([hotlists, hotlistRef],
    (hotlists, hotlistRef) => {
      if (!hotlistRef) {
        return null;
      }
      return hotlists[hotlistRefToString(hotlistRef)] || null;
    });

// Action Creators
/**
 * Action creator to set the currently viewed Hotlist.
 * @param {string} userDisplayName The user who owns the Hotlist.
 * @param {string} hotlistName The name of the Hotlist.
 * @return {function(function): void}
 */
export const selectHotlist = (userDisplayName, hotlistName) => {
  return (dispatch) => {
    const hotlistRef = {
      owner: userIdOrDisplayNameToUserRef(userDisplayName),
      name: hotlistName,
    };
    dispatch({type: SELECT_HOTLIST, hotlistRef});
  };
};

/**
 * Action creator to fetch a Hotlist object.
 * @param {HotlistRef} hotlistRef A reference to the Hotlist to fetch.
 * @return {function(function): Promise<void>}
 */
export const fetch = (hotlistRef) => async (dispatch) => {
  dispatch({type: FETCH_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Features', 'GetHotlist', {hotlistRef});

    dispatch({type: FETCH_SUCCESS, hotlist: resp.hotlist});
  } catch (error) {
    dispatch({type: FETCH_FAILURE, error});
  };
};
