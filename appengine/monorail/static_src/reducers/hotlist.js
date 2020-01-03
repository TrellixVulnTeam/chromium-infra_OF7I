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
 * `hotlistItems` stores all Hotlist items indexed by HotlistRefString.
 * `hotlistRef` is a reference to the currently viewed Hotlist.
 * `hotlist` is a selector that gets the currently viewed Hotlist data.
 *
 * Reference: https://github.com/erikras/ducks-modular-redux
 */

import {combineReducers} from 'redux';
import {createSelector} from 'reselect';
import {hotlistToRef, hotlistToRefString, hotlistRefToString}
  from 'shared/converters-hotlist.js';
import {createReducer, createRequestReducer} from './redux-helpers.js';
import {prpcClient} from 'prpc-client-instance.js';
import 'shared/typedef.js';

/** @typedef {import('redux').AnyAction} AnyAction */

// Actions
export const SELECT = 'hotlist/SELECT';

export const FETCH_START = 'hotlist/FETCH_START';
export const FETCH_SUCCESS = 'hotlist/FETCH_SUCCESS';
export const FETCH_FAILURE = 'hotlist/FETCH_FAILURE';

export const FETCH_ITEMS_START = 'hotlist/FETCH_ITEMS_START';
export const FETCH_ITEMS_SUCCESS = 'hotlist/FETCH_ITEMS_SUCCESS';
export const FETCH_ITEMS_FAILURE = 'hotlist/FETCH_ITEMS_FAILURE';

/* State Shape
{
  hotlists: Object.<string, Hotlist>,
  hotlistItems: Object.<string, Array<HotlistItem>>,

  hotlistRef: HotlistRef,

  requests: {
    fetch: ReduxRequestState,
  },
}
*/

// Reducers

/**
 * All Hotlist data indexed by HotlistRefString.
 * @param {Object<string, Hotlist>} state The mapping of existing Hotlist data.
 * @param {AnyAction} action
 * @param {Hotlist} action.hotlist The hotlist that was fetched.
 * @return {Object.<string, Hotlist>}
 */
export const hotlistsReducer = createReducer({}, {
  [FETCH_SUCCESS]: (state, {hotlist}) => {
    const newState = {...state};
    newState[hotlistToRefString(hotlist)] = hotlist;
    return newState;
  },
});

/**
 * All Hotlist items indexed by HotlistRefString.
 * @param {Object<string, Array<HotlistItem>>} state
 * @param {AnyAction} action
 * @param {HotlistRef} action.hotlistRef A reference to the hotlist that items
 *   were fetched from.
 * @param {Array<HotlistItem>} action.items The hotlist items fetched.
 * @return {Object.<string, Array<HotlistItem>>}
 */
export const hotlistItemsReducer = createReducer({}, {
  [FETCH_ITEMS_SUCCESS]: (state, {hotlistRef, items}) => ({
    ...state,
    [hotlistRefToString(hotlistRef)]: items,
  }),
});

/**
 * A reference to the currently viewed Hotlist.
 * @param {?Hotlist} state The existing HotlistRef.
 * @param {AnyAction} action
 * @return {?Hotlist}
 */
export const hotlistRefReducer = createReducer(null, {
  [SELECT]: (_state, {hotlistRef}) => hotlistRef,
  [FETCH_SUCCESS]: (state, {hotlist}) => {
    // The original HotlistRef may be missing the displayName or userId.
    // If we just fetched the referenced Hotlist, update the missing info.
    if (!state) {
      return state;
    }

    const newRef = hotlistToRef(hotlist);
    const sameName = state.name === newRef.name;
    const sameOwner = state.owner.userId === newRef.owner.userId ||
      state.owner.displayName === newRef.owner.displayName;
    const oldRefMissingField = !state.owner.userId || !state.owner.displayName;
    return sameName && sameOwner && oldRefMissingField ? newRef : state;
  },
});

const requestsReducer = combineReducers({
  fetch: createRequestReducer(
      FETCH_START, FETCH_SUCCESS, FETCH_FAILURE),
});

export const reducer = combineReducers({
  hotlists: hotlistsReducer,
  hotlistItems: hotlistItemsReducer,
  hotlistRef: hotlistRefReducer,

  requests: requestsReducer,
});

// Selectors
/**
 * Returns all the Hotlist data in the store as
 * a mapping of HotlistRef string to Hotlist.
 * @param {any} state
 * @return {Object.<string, Hotlist>}
 */
export const hotlists = (state) => state.hotlist.hotlists;

/**
 * Returns all the Hotlist items in the store as a mapping of
 * HotlistRef string to its respective array of HotlistItems.
 * @param {any} state
 * @return {Object.<string, Array<HotlistItem>>}
 */
export const hotlistItems = (state) => state.hotlist.hotlistItems;

/**
 * Returns the currently viewed HotlistRef, or null if there is none.
 * @param {any} state
 * @return {?HotlistRef}
 */
export const hotlistRef = (state) => state.hotlist.hotlistRef;

/**
 * Returns the currently viewed Hotlist, or null if there is none.
 * @param {any} state
 * @return {?Hotlist}
 */
export const viewedHotlist = createSelector([hotlists, hotlistRef],
    (hotlists, hotlistRef) => {
      if (!hotlistRef) {
        return null;
      }
      return hotlists[hotlistRefToString(hotlistRef)] || null;
    });

/**
 * Returns an Array containing the items in the currently viewed Hotlist,
 * or empty Array if there is no current HotlistRef or no data.
 * @param {any} state
 * @return {Array<HotlistItem>}
 */
export const viewedHotlistItems = createSelector([hotlistItems, hotlistRef],
    (hotlistItems, hotlistRef) => {
      if (!hotlistRef) {
        return [];
      }
      return hotlistItems[hotlistRefToString(hotlistRef)] || [];
    });

// Action Creators
/**
 * Action creator to set the currently viewed Hotlist.
 * @param {HotlistRef} hotlistRef A reference to the Hotlist to select.
 * @return {function(function): void}
 */
export const select = (hotlistRef) => {
  return (dispatch) => dispatch({type: SELECT, hotlistRef});
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

/**
 * Action creator to fetch the items in a Hotlist.
 * @param {HotlistRef} hotlistRef A reference to the Hotlist to fetch.
 * @return {function(function): Promise<void>}
 */
export const fetchItems = (hotlistRef) => async (dispatch) => {
  dispatch({type: FETCH_ITEMS_START});

  try {
    const resp = await prpcClient.call(
        'monorail.Features', 'ListHotlistItems', {hotlistRef});

    dispatch({type: FETCH_ITEMS_SUCCESS, hotlistRef, items: resp.items});
  } catch (error) {
    dispatch({type: FETCH_ITEMS_FAILURE, error});
  };
};
