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
import {createReducer, createRequestReducer} from './redux-helpers.js';

import {prpcClient} from 'prpc-client-instance.js';
import {userIdOrDisplayNameToUserRef, issueNameToRef}
  from 'shared/convertersV0.js';
import {pathsToFieldMask} from 'shared/converters.js';

import * as issueV0 from './issueV0.js';
import * as permissions from './permissions.js';
import * as users from './users.js';

import 'shared/typedef.js';
/** @typedef {import('redux').AnyAction} AnyAction */

// Permissions
export const EDIT = 'HOTLIST_EDIT';
export const ADMINISTER = 'HOTLIST_ADMINISTER';

// Actions
export const SELECT = 'hotlist/SELECT';

export const DELETE_START = 'hotlist/DELETE_START';
export const DELETE_SUCCESS = 'hotlist/DELETE_SUCCESS';
export const DELETE_FAILURE = 'hotlist/DELETE_FAILURE';

export const FETCH_START = 'hotlist/FETCH_START';
export const FETCH_SUCCESS = 'hotlist/FETCH_SUCCESS';
export const FETCH_FAILURE = 'hotlist/FETCH_FAILURE';

export const FETCH_ITEMS_START = 'hotlist/FETCH_ITEMS_START';
export const FETCH_ITEMS_SUCCESS = 'hotlist/FETCH_ITEMS_SUCCESS';
export const FETCH_ITEMS_FAILURE = 'hotlist/FETCH_ITEMS_FAILURE';

export const REMOVE_EDITORS_START = 'hotlist/REMOVE_EDITORS_START';
export const REMOVE_EDITORS_SUCCESS = 'hotlist/REMOVE_EDITORS_SUCCESS';
export const REMOVE_EDITORS_FAILURE = 'hotlist/REMOVE_EDITORS_FAILURE';

export const REMOVE_ITEMS_START = 'hotlist/REMOVE_ITEMS_START';
export const REMOVE_ITEMS_SUCCESS = 'hotlist/REMOVE_ITEMS_SUCCESS';
export const REMOVE_ITEMS_FAILURE = 'hotlist/REMOVE_ITEMS_FAILURE';

export const RERANK_ITEMS_START = 'hotlist/RERANK_ITEMS_START';
export const RERANK_ITEMS_SUCCESS = 'hotlist/RERANK_ITEMS_SUCCESS';
export const RERANK_ITEMS_FAILURE = 'hotlist/RERANK_ITEMS_FAILURE';

export const UPDATE_START = 'hotlist/UPDATE_START';
export const UPDATE_SUCCESS = 'hotlist/UPDATE_SUCCESS';
export const UPDATE_FAILURE = 'hotlist/UPDATE_FAILURE';

/* State Shape
{
  name: string,

  byName: Object<string, Hotlist>,
  hotlistItems: Object<string, Array<HotlistItem>>,

  requests: {
    fetch: ReduxRequestState,
    fetchItems: ReduxRequestState,
    update: ReduxRequestState,
  },
}
*/

// Reducers

/**
 * A reference to the currently viewed Hotlist.
 * @param {?string} state The existing Hotlist resource name.
 * @param {AnyAction} action
 * @return {?string}
 */
export const nameReducer = createReducer(null, {
  [SELECT]: (_state, {name}) => name,
});

/**
 * All Hotlist data indexed by Hotlist resource name.
 * @param {Object<string, Hotlist>} state The existing Hotlist data.
 * @param {AnyAction} action
 * @param {Hotlist} action.hotlist The Hotlist that was fetched.
 * @return {Object<string, Hotlist>}
 */
export const byNameReducer = createReducer({}, {
  [FETCH_SUCCESS]: (state, {hotlist}) => ({...state, [hotlist.name]: hotlist}),
  [UPDATE_SUCCESS]: (state, {hotlist}) => ({...state, [hotlist.name]: hotlist}),
});

/**
 * All Hotlist items indexed by Hotlist resource name.
 * @param {Object<string, Array<HotlistItem>>} state The existing items.
 * @param {AnyAction} action
 * @param {name} action.name The Hotlist resource name.
 * @param {Array<HotlistItem>} action.items The Hotlist items fetched.
 * @return {Object<string, Array<HotlistItem>>}
 */
export const hotlistItemsReducer = createReducer({}, {
  [FETCH_ITEMS_SUCCESS]: (state, {name, items}) => ({...state, [name]: items}),
});

const requestsReducer = combineReducers({
  deleteHotlist: createRequestReducer(
      DELETE_START, DELETE_SUCCESS, DELETE_FAILURE),
  fetch: createRequestReducer(
      FETCH_START, FETCH_SUCCESS, FETCH_FAILURE),
  fetchItems: createRequestReducer(
      FETCH_ITEMS_START, FETCH_ITEMS_SUCCESS, FETCH_ITEMS_FAILURE),
  removeEditors: createRequestReducer(
      REMOVE_EDITORS_START, REMOVE_EDITORS_SUCCESS, REMOVE_EDITORS_FAILURE),
  removeItems: createRequestReducer(
      REMOVE_ITEMS_START, REMOVE_ITEMS_SUCCESS, REMOVE_ITEMS_FAILURE),
  rerankItems: createRequestReducer(
      RERANK_ITEMS_START, RERANK_ITEMS_SUCCESS, RERANK_ITEMS_FAILURE),
  update: createRequestReducer(
      UPDATE_START, UPDATE_SUCCESS, UPDATE_FAILURE),
});

export const reducer = combineReducers({
  name: nameReducer,

  byName: byNameReducer,
  hotlistItems: hotlistItemsReducer,

  requests: requestsReducer,
});

// Selectors

/**
 * Returns the currently viewed Hotlist resource name, or null if there is none.
 * @param {any} state
 * @return {?string}
 */
export const name = (state) => state.hotlists.name;

/**
 * Returns all the Hotlist data in the store as a mapping from name to Hotlist.
 * @param {any} state
 * @return {Object<string, Hotlist>}
 */
export const byName = (state) => state.hotlists.byName;

/**
 * Returns all the Hotlist items in the store as a mapping from a
 * Hotlist resource name to its respective array of HotlistItems.
 * @param {any} state
 * @return {Object<string, Array<HotlistItem>>}
 */
export const hotlistItems = (state) => state.hotlists.hotlistItems;

/**
 * Returns the currently viewed Hotlist, or null if there is none.
 * @param {any} state
 * @return {?Hotlist}
 */
export const viewedHotlist = createSelector(
    [byName, name],
    (byName, name) => name && byName[name] || null);

/**
 * Returns the owner of the currently viewed Hotlist, or null if there is none.
 * @param {any} state
 * @return {?User}
 */
export const viewedHotlistOwner = createSelector(
    [viewedHotlist, users.byName],
    (hotlist, usersByName) => {
      return hotlist && usersByName[hotlist.owner] || null;
    });

/**
 * Returns the editors of the currently viewed Hotlist. Returns [] if there is
 * no hotlist data. Includes a null in the array for each editor whose User
 * data is not in the store.
 * @param {any} state
 * @return {Array<User>}
 */
export const viewedHotlistEditors = createSelector(
    [viewedHotlist, users.byName],
    (hotlist, usersByName) => {
      if (!hotlist) return [];
      return hotlist.editors.map((editor) => usersByName[editor] || null);
    });

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
    [viewedHotlistItems, issueV0.issue, users.byName],
    (items, getIssue, usersByName) => {
      // Filter out issues that haven't been fetched yet or failed to fetch.
      // Example: if the user doesn't have permissions to view the issue.
      // <mr-issue-list> assumes that every Issue is populated.
      const itemsWithData = items.filter((item) => getIssue(item.issue));
      return itemsWithData.map((item) => ({
        ...getIssue(item.issue),
        ...item,
        adder: usersByName[item.adder],
      }));
    });

/**
 * Returns the currently viewed Hotlist permissions, or [] if there is none.
 * @param {any} state
 * @return {Array<Permission>}
 */
export const viewedHotlistPermissions = createSelector(
    [viewedHotlist, permissions.byName],
    (hotlist, permissionsByName) => {
      if (!hotlist) return [];
      const permissionSet = permissionsByName[hotlist.name];
      if (!permissionSet) return [];
      return permissionSet.permissions;
    });

/**
 * Returns the Hotlist requests.
 * @param {any} state
 * @return {Object.<string, ReduxRequestState>}
 */
export const requests = (state) => state.hotlists.requests;

// Action Creators

/**
 * Action creator to delete the Hotlist. We would have liked to have named this
 * `delete` but it's a reserved word in JS.
 * @param {string} name The resource name of the Hotlist to delete.
 * @return {function(function): Promise<void>}
 */
export const deleteHotlist = (name) => async (dispatch) => {
  dispatch({type: DELETE_START});

  try {
    const args = {name};
    await prpcClient.call('monorail.v1.Hotlists', 'DeleteHotlist', args);

    dispatch({type: DELETE_SUCCESS});
  } catch (error) {
    dispatch({type: DELETE_FAILURE, error});
  };
};

/**
 * Action creator to fetch a Hotlist object.
 * @param {string} name The resource name of the Hotlist to fetch.
 * @return {function(function): Promise<Hotlist>}
 */
export const fetch = (name) => async (dispatch) => {
  dispatch({type: FETCH_START});

  try {
    /** @type {Hotlist} */
    const hotlist = await prpcClient.call(
        'monorail.v1.Hotlists', 'GetHotlist', {name});
    if (!hotlist.editors) hotlist.editors = [];

    const editors = hotlist.editors.map((editor) => editor);
    editors.push(hotlist.owner);
    await dispatch(users.batchGet(editors));

    dispatch({type: FETCH_SUCCESS, hotlist});
  } catch (error) {
    dispatch({type: FETCH_FAILURE, error});
  };
};

/**
 * Action creator to fetch the items in a Hotlist.
 * @param {string} name The resource name of the Hotlist to fetch.
 * @return {function(function): Promise<Array<HotlistItem>>}
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
    await dispatch(issueV0.fetchIssues(issueRefs));

    const adderNames = [...new Set(items.map((item) => item.adder))];
    await dispatch(users.batchGet(adderNames));

    dispatch({type: FETCH_ITEMS_SUCCESS, name, items: itemsWithRank});
    return itemsWithRank;
  } catch (error) {
    dispatch({type: FETCH_ITEMS_FAILURE, error});
  };
};

/**
 * Action creator to remove editors from a Hotlist.
 * @param {string} name The resource name of the Hotlist.
 * @param {Array<string>} editors The resource names of the Users to remove.
 * @return {function(function): Promise<void>}
 */
export const removeEditors = (name, editors) => async (dispatch) => {
  dispatch({type: REMOVE_EDITORS_START});

  try {
    const args = {name, editors};
    await prpcClient.call('monorail.v1.Hotlists', 'RemoveHotlistEditors', args);

    dispatch({type: REMOVE_EDITORS_SUCCESS});

    await dispatch(fetch(name));
  } catch (error) {
    dispatch({type: REMOVE_EDITORS_FAILURE, error});
  };
};

/**
 * Action creator to remove items from a Hotlist.
 * @param {string} name The resource name of the Hotlist.
 * @param {Array<string>} issues The resource names of the Issues to remove.
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
 * @param {string} name The resource name of the Hotlist.
 * @param {Array<string>} items The resource names of the HotlistItems to move.
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

/**
 * Action creator to set the currently viewed Hotlist.
 * @param {string} name The resource name of the Hotlist to select.
 * @return {AnyAction}
 */
export const select = (name) => ({type: SELECT, name});

/**
 * Action creator to update the Hotlist metadata.
 * @param {string} name The resource name of the Hotlist to delete.
 * @param {Hotlist} hotlist This represents the updated version of the Hotlist
 *    with only the fields that need to be updated.
 * @return {function(function): Promise<void>}
 */
export const update = (name, hotlist) => async (dispatch) => {
  dispatch({type: UPDATE_START});
  try {
    const paths = pathsToFieldMask(Object.keys(hotlist));
    const hotlistArg = {...hotlist, name};
    const args = {hotlist: hotlistArg, updateMask: paths};

    /** @type {Hotlist} */
    const updatedHotlist = await prpcClient.call(
        'monorail.v1.Hotlists', 'UpdateHotlist', args);

    dispatch({type: UPDATE_SUCCESS, hotlist: updatedHotlist});
  } catch (error) {
    dispatch({type: UPDATE_FAILURE, error});
  }
};

// Helpers

/**
 * Helper to fetch a Hotlist ID given its owner and display name.
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
