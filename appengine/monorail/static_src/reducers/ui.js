// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {combineReducers} from 'redux';
import {createReducer} from './redux-helpers.js';

/** @typedef {import('redux').AnyAction} AnyAction */

const DEFAULT_SNACKBAR_TIMEOUT_MS = 10 * 1000;


/**
 * Object of various constant strings used to uniquely identify
 * snackbar instances used in the app.
 * @type {Object.<string, string>}
 */
export const snackbarNames = Object.freeze({
  // Issue list page snackbars.
  FETCH_ISSUE_LIST_ERROR: 'FETCH_ISSUE_LIST_ERROR',
  FETCH_ISSUE_LIST: 'FETCH_ISSUE_LIST',
  UPDATE_HOTLISTS_SUCCESS: 'UPDATE_HOTLISTS_SUCCESS',

  // Issue detail page snackbars.
  ISSUE_COMMENT_ADDED: 'ISSUE_COMMENT_ADDED',
});

// Actions
const INCREMENT_NAVIGATION_COUNT = 'INCREMENT_NAVIGATION_COUNT';
const REPORT_DIRTY_FORM = 'REPORT_DIRTY_FORM';
const CLEAR_DIRTY_FORMS = 'CLEAR_DIRTY_FORMS';
const SET_FOCUS_ID = 'SET_FOCUS_ID';
const SHOW_SNACKBAR = 'SHOW_SNACKBAR';
const HIDE_SNACKBAR = 'HIDE_SNACKBAR';

/**
 * @typedef {Object} Snackbar
 * @param {string} id Unique string identifying the snackbar.
 * @param {string} text The text to show in the snackbar.
 */

/* State Shape
{
  navigationCount: number,
  dirtyForms: Array,
  focusId: String,
  snackbars: Array<Snackbar>,
}
*/

// Reducers


const navigationCountReducer = createReducer(0, {
  [INCREMENT_NAVIGATION_COUNT]: (state) => state + 1,
});

/**
 * Saves state on which forms have been edited, to warn the user
 * about possible data loss when they navigate away from a page.
 * @param {Array<string>} state Dirty form names.
 * @param {AnyAction} action
 * @param {string} action.name The name of the form being updated.
 * @param {boolean} action.isDirty Whether the form is dirty or not dirty.
 * @return {Array<string>}
 */
const dirtyFormsReducer = createReducer([], {
  [REPORT_DIRTY_FORM]: (state, {name, isDirty}) => {
    const newState = [...state];
    const index = state.indexOf(name);
    if (isDirty && index === -1) {
      newState.push(name);
    } else if (!isDirty && index !== -1) {
      newState.splice(index, 1);
    }
    return newState;
  },
  [CLEAR_DIRTY_FORMS]: () => [],
});

const focusIdReducer = createReducer(null, {
  [SET_FOCUS_ID]: (_state, action) => action.focusId,
});

/**
 * Updates snackbar state.
 * @param {Array<Snackbar>} state A snackbar-shaped slice of Redux state.
 * @param {AnyAction} action
 * @param {string} action.text The text to display in the snackbar.
 * @param {string} action.id A unique global ID for the snackbar.
 * @return {Array<Snackbar>} New snackbar state.
 */
export const snackbarsReducer = createReducer([], {
  [SHOW_SNACKBAR]: (state, {text, id}) => {
    return [...state, {text, id}];
  },
  [HIDE_SNACKBAR]: (state, {id}) => {
    return state.filter((snackbar) => snackbar.id !== id);
  },
});

export const reducer = combineReducers({
  // Count of "page" navigations.
  navigationCount: navigationCountReducer,
  // Forms to be checked for user changes before leaving the page.
  dirtyForms: dirtyFormsReducer,
  // The ID of the element to be focused, as given by the hash part of the URL.
  focusId: focusIdReducer,
  // Array of snackbars to render on the page.
  snackbars: snackbarsReducer,
});

// Selectors
export const navigationCount = (state) => state.ui.navigationCount;
export const dirtyForms = (state) => state.ui.dirtyForms;
export const focusId = (state) => state.ui.focusId;

/**
 * Retrieves snackbar data from the Redux store.
 * @param {any} state Redux state.
 * @return {Array<Snackbar>} All the snackbars in the store.
 */
export const snackbars = (state) => state.ui.snackbars;

// Action Creators
export const incrementNavigationCount = () => {
  return {type: INCREMENT_NAVIGATION_COUNT};
};

export const reportDirtyForm = (name, isDirty) => {
  return {type: REPORT_DIRTY_FORM, name, isDirty};
};

export const clearDirtyForms = () => ({type: CLEAR_DIRTY_FORMS});

export const setFocusId = (focusId) => {
  return {type: SET_FOCUS_ID, focusId};
};

/**
 * Displays a snackbar.
 * @param {string} id Unique identifier for a given snackbar. We depend on
 *   snackbar users to keep this unique.
 * @param {string} text The text to be shown in the snackbar.
 * @param {number} timeout An optional timeout in milliseconds for how
 *   long to wait to dismiss a snackbar.
 * @return {function(function): void}
 */
export const showSnackbar = (id, text,
    timeout = DEFAULT_SNACKBAR_TIMEOUT_MS) => (dispatch) => {
  dispatch({type: SHOW_SNACKBAR, text, id});

  if (timeout) {
    window.setTimeout(() => dispatch(hideSnackbar(id)),
        timeout);
  }
};

/**
 * Hides a snackbar.
 * @param {string} id The unique name of the snackbar to be hidden.
 * @return {any} A Redux action.
 */
export const hideSnackbar = (id) => {
  return {
    type: HIDE_SNACKBAR,
    id,
  };
};
