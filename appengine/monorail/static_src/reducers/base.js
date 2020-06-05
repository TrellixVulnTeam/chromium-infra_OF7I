// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {connect} from 'pwa-helpers/connect-mixin.js';
import {applyMiddleware, combineReducers, compose, createStore} from 'redux';
import thunk from 'redux-thunk';
import {hotlists} from './hotlists.js';
import * as issueV0 from './issueV0.js';
import * as permissions from './permissions.js';
import * as projects from './projects.js';
import * as projectV0 from './projectV0.js';
import * as sitewide from './sitewide.js';
import * as users from './users.js';
import * as userV0 from './userV0.js';
import * as ui from './ui.js';

/** @typedef {import('redux').AnyAction} AnyAction */

// Actions
const RESET_STATE = 'RESET_STATE';

/* State Shape
{
  hotlists: Object,
  permissions: Object,
  projects: Object,
  sitewide: Object,
  users: Object,

  ui: Object,

  // To be deprecated
  issue: Object,
  projectV0: Object,
  userV0: Object,
}
*/

// Reducers
const reducer = combineReducers({
  hotlists: hotlists.reducer,
  issue: issueV0.reducer,
  permissions: permissions.reducer,
  projects: projects.reducer,
  projectV0: projectV0.reducer,
  users: users.reducer,
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
