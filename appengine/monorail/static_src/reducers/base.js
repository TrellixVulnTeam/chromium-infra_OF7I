// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {connect} from 'pwa-helpers/connect-mixin.js';
import {applyMiddleware, combineReducers, compose, createStore} from 'redux';
import thunk from 'redux-thunk';
import * as hotlist from './hotlist.js';
import * as issueV0 from './issueV0.js';
import * as projectV0 from './projectV0.js';
import * as sitewide from './sitewide.js';
import * as user from './user.js';
import * as userV0 from './userV0.js';
import * as ui from './ui.js';

/** @typedef {import('redux').AnyAction} AnyAction */

// Actions
const RESET_STATE = 'RESET_STATE';

/* State Shape
{
  hotlist: Object,
  issue: Object,
  project: Object,
  sitewide: Object,
  user: Object,

  ui: Object,
}
*/

// Reducers
const reducer = combineReducers({
  hotlist: hotlist.reducer,
  issue: issueV0.reducer,
  project: projectV0.reducer,
  user: user.reducer,
  userV0: userV0.reducer,
  sitewide: sitewide.reducer,

  ui: ui.reducer,
});

/**
 * The top level reducer function that all actions pass through.
 * @param {any} state
 * @param {AnyAction} action
 * @return {any}
 */
export function rootReducer(state, action) {
  if (action.type === RESET_STATE) {
    state = undefined;
  }
  return reducer(state, action);
}

// Selectors

// Action Creators

/**
 * Changes Redux state back to its default initial state. Primarily
 * used in testing.
 * @return {AnyAction} An action to reset Redux state to default.
 */
export const resetState = () => ({type: RESET_STATE});

// Store

// For debugging with the Redux Devtools extension:
// https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd/
const composeEnhancers = window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__ || compose;
export const store = createStore(rootReducer, composeEnhancers(
    applyMiddleware(thunk),
));

/**
 * Class mixin function that connects a given HTMLElement class to our
 * store instance.
 * @link https://pwa-starter-kit.polymer-project.org/redux-and-state-management#connecting-an-element-to-the-store
 * @param {typeof HTMLElement} class
 * @return {function} New class type with connected features.
 */
export const connectStore = connect(store);

/**
 * Promise to allow waiting for a state update. Useful in testing.
 * @example
 * store.dispatch(updateState());
 * await stateUpdated;
 * doThingWithUpdatedState();
 *
 * @type {Promise}
 */
export const stateUpdated = new Promise((resolve) => {
  store.subscribe(resolve);
});
