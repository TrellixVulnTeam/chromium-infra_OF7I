// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Hotlist actions, selectors, and reducers organized into
 * a single Redux "Duck" that manages updating and retrieving hotlist state
 * on the frontend.
 *
 * The Hotlist data is stored in a normalized format.
 * `name` is a reference to the currently viewed Hotlist.
 * `hotlists` stores all Hotlist data indexed by Hotlist name.
 * `hotlistItems` stores all Hotlist items indexed by Hotlist name.
 * `hotlist` is a selector that gets the currently viewed Hotlist data.
 *
 * Reference: https://github.com/erikras/ducks-modular-redux
 */

import {combineReducers} from 'redux';
import {createSelector} from 'reselect';
import {userIdOrDisplayNameToUserRef, issueNameToRef}
  from 'shared/converters.js';
import {createReducer, createRequestReducer} from './redux-helpers.js';
import * as issue from './issue.js';
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

export const REMOVE_ITEMS_START = 'hotlist/REMOVE_ITEMS_START';
export const REMOVE_ITEMS_SUCCESS = 'hotlist/REMOVE_ITEMS_SUCCESS';
export const REMOVE_ITEMS_FAILURE = 'hotlist/REMOVE_ITEMS_FAILURE';

export const RERANK_ITEMS_START = 'hotlist/RERANK_ITEMS_START';
export const RERANK_ITEMS_SUCCESS = 'hotlist/RERANK_ITEMS_SUCCESS';
export const RERANK_ITEMS_FAILURE = 'hotlist/RERANK_ITEMS_FAILURE';

/* State Shape
{
  name: string,

  hotlists: Object<string, Hotlist>,
  hotlistItems: Object<string, Array<HotlistItem>>,

  requests: {
    fetch: ReduxRequestState,
    fetchItems: ReduxRequestState,
  },
}
*/

// Reducers

/**
 * A reference to the currently viewed Hotlist.
 * @param {?string} state The existing Hotlist name.
 * @param {AnyAction} action
 * @return {?string}
 */
export const nameReducer = createReducer(null, {
  [SELECT]: (_state, {name}) => name,
});

/**
 * All Hotlist data indexed by Hotlist name.
 * @param {Object<string, Hotlist>} state The existing Hotlist data.
 * @param {AnyAction} action
 * @param {Hotlist} action.hotlist The hotlist that was fetched.
 * @return {Object<string, Hotlist>}
 */
export const hotlistsReducer = createReducer({}, {
  [FETCH_SUCCESS]: (state, {hotlist}) => ({...state, [hotlist.name]: hotlist}),
});

/**
 * All Hotlist items indexed by Hotlist name.
 * @param {Object<string, Array<HotlistItem>>} state The existing items.
 * @param {AnyAction} action
 * @param {name} action.name A reference to the Hotlist.
 * @param {Array<HotlistItem>} action.items The Hotlist items fetched.
 * @return {Object<string, Array<HotlistItem>>}
 */
export const hotlistItemsReducer = createReducer({}, {
  [FETCH_ITEMS_SUCCESS]: (state, {name, items}) => ({...state, [name]: items}),
});

const requestsReducer = combineReducers({
  fetch: createRequestReducer(
      FETCH_START, FETCH_SUCCESS, FETCH_FAILURE),
  fetchItems: createRequestReducer(
      FETCH_ITEMS_START, FETCH_ITEMS_SUCCESS, FETCH_ITEMS_FAILURE),
  removeItems: createRequestReducer(
      REMOVE_ITEMS_START, REMOVE_ITEMS_SUCCESS, REMOVE_ITEMS_FAILURE),
  rerankItems: createRequestReducer(
      RERANK_ITEMS_START, RERANK_ITEMS_SUCCESS, RERANK_ITEMS_FAILURE),
});

export const reducer = combineReducers({
  name: nameReducer,

  hotlists: hotlistsReducer,
  hotlistItems: hotlistItemsReducer,

  requests: requestsReducer,
});

// Selectors

/**
 * Returns the currently viewed Hotlist name, or null if there is none.
 * @param {any} state
 * @return {?string}
 */
export const name = (state) => state.hotlist.name;

/**
 * Returns all the Hotlist data in the store as a mapping from name to Hotlist.
 * @param {any} state
 * @return {Object<string, Hotlist>}
 */
export const hotlists = (state) => state.hotlist.hotlists;

/**
 * Returns all the Hotlist items in the store as a mapping
 * from a Hotlist name to its respective array of HotlistItems.
 * @param {any} state
 * @return {Object<string, Array<HotlistItem>>}
 */
export const hotlistItems = (state) => state.hotlist.hotlistItems;

/**
 * Returns the currently viewed Hotlist, or null if there is none.
 * @param {any} state
 * @return {?Hotlist}
 */
export const viewedHotlist = createSelector(
    [hotlists, name],
    (hotlists, name) => name && hotlists[name] || null);

/**
 * Returns an Array containing the items in the currently viewed Hotlist,
 * or [] if there is no current Hotlist or no Hotlist data.
 * @param {any} state
 * @return {Array<HotlistItem>}
 */
export const viewedHotlistItems = createSelector(
    [hotlistItems, name],
    (hotlistItems, name) => name && hotlistItems[name] || []);

/**
 * Returns an Array containing the HotlistIssues in the currently viewed
 * Hotlist, or [] if there is no current Hotlist or no Hotlist data.
 * A HotlistIssue merges the HotlistItem and Issue into one flat object.
 * @param {any} state
 * @return {Array<HotlistIssue>}
 */
export const viewedHotlistIssues = createSelector(
    [viewedHotlistItems, issue.issue],
    (items, getIssue) => {
      // Filter out issues that haven't been fetched yet or failed to fetch.
      // Example: if the user doesn't have permissions to view the issue.
      // <mr-issue-list> assumes that every Issue is populated.
      const itemsWithData = items.filter((item) => getIssue(item.issue));
      return itemsWithData.map((item) => ({
        ...getIssue(item.issue),
        ...item,
      }));
    });

// Action Creators

/**
 * Action creator to set the currently viewed Hotlist.
 * @param {string} name The name of the Hotlist to select.
 * @return {AnyAction}
 */
export const select = (name) => ({type: SELECT, name});

/**
 * Action creator to fetch a Hotlist object.
 * @param {string} name The name of the Hotlist to fetch.
 * @return {function(function): Promise<void>}
 */
export const fetch = (name) => async (dispatch) => {
  dispatch({type: FETCH_START});

  try {
    /** @type {Hotlist} */
    const hotlist = await prpcClient.call(
        'monorail.v1.Hotlists', 'GetHotlist', {name});
    if (!hotlist.editors) {
      hotlist.editors = [];
    }

    dispatch({type: FETCH_SUCCESS, hotlist});
  } catch (error) {
    dispatch({type: FETCH_FAILURE, error});
  };
};

/**
 * Action creator to fetch the items in a Hotlist.
 * @param {string} name The name of the Hotlist to fetch.
 * @return {function(function): Promise<void>}
 */
export const fetchItems = (name) => async (dispatch) => {
  dispatch({type: FETCH_ITEMS_START});

  try {
    const args = {parent: name, orderBy: 'rank'};
    /** @type {{items: Array<HotlistItem>}} */
    const {items} = await prpcClient.call(
        'monorail.v1.Hotlists', 'ListHotlistItems', args);
    const itemsWithRank =
        items.map((item) => item.rank ? item : {...item, rank: 0});

    const issueRefs = items.map((item) => issueNameToRef(item.issue));
    dispatch(issue.fetchIssues(issueRefs));

    dispatch({type: FETCH_ITEMS_SUCCESS, name, items: itemsWithRank});
  } catch (error) {
    dispatch({type: FETCH_ITEMS_FAILURE, error});
  };
};

/**
 * Action creator to remove items from a Hotlist.
 * @param {string} name The name of the Hotlist.
 * @param {Array<string>} issues The names of the Issues to remove.
 * @return {function(function): Promise<void>}
 */
export const removeItems = (name, issues) => async (dispatch) => {
  dispatch({type: REMOVE_ITEMS_START});

  try {
    const args = {parent: name, issues};
    await prpcClient.call('monorail.v1.Hotlists', 'RemoveHotlistItems', args);

    dispatch({type: REMOVE_ITEMS_SUCCESS});

    await dispatch(fetchItems(name));
  } catch (error) {
    dispatch({type: REMOVE_ITEMS_FAILURE, error});
  };
};

/**
 * Action creator to rerank the items in a Hotlist.
 * @param {string} name The name of the Hotlist.
 * @param {Array<string>} items The names of the HotlistItems to move.
 * @param {number} index The index to insert the moved items.
 * @return {function(function): Promise<void>}
 */
export const rerankItems = (name, items, index) => async (dispatch) => {
  dispatch({type: RERANK_ITEMS_START});

  try {
    const args = {name, hotlistItems: items, targetPosition: index};
    await prpcClient.call('monorail.v1.Hotlists', 'RerankHotlistItems', args);

    dispatch({type: RERANK_ITEMS_SUCCESS});

    await dispatch(fetchItems(name));
  } catch (error) {
    dispatch({type: RERANK_ITEMS_FAILURE, error});
  };
};

// Helpers

/**
 * Helper to fetch a Hotlist ID given its owner and name.
 * @param {string} owner The Hotlist owner's user id or display name.
 * @param {string} hotlist The Hotlist's id or display name.
 * @return {Promise<?string>}
 */
export const getHotlistName = async (owner, hotlist) => {
  const hotlistRef = {
    owner: userIdOrDisplayNameToUserRef(owner),
    name: hotlist,
  };

  try {
    /** @type {{hotlistId: number}} */
    const {hotlistId} = await prpcClient.call(
        'monorail.Features', 'GetHotlistID', {hotlistRef});
    return 'hotlists/' + hotlistId;
  } catch (error) {
    return null;
  };
};
