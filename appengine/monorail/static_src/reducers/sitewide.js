// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as project from 'reducers/project.js';
import {combineReducers} from 'redux';
import {createReducer, createRequestReducer} from './redux-helpers.js';
import {createSelector} from 'reselect';
import {prpcClient} from 'prpc-client-instance.js';
import {SITEWIDE_DEFAULT_CAN, parseColSpec} from 'shared/issue-fields.js';

// Actions
const SET_PAGE_TITLE = 'SET_PAGE_TITLE';
const SET_HEADER_TITLE = 'SET_HEADER_TITLE';
export const SET_QUERY_PARAMS = 'SET_QUERY_PARAMS';

// Async actions
const GET_SERVER_STATUS_FAILURE = 'GET_SERVER_STATUS_FAILURE';
const GET_SERVER_STATUS_START = 'GET_SERVER_STATUS_START';
const GET_SERVER_STATUS_SUCCESS = 'GET_SERVER_STATUS_SUCCESS';

/* State Shape
{
  bannerMessage: String,
  bannerTime: Number,
  pageTitle: String,
  headerTitle: String,
  queryParams: Object,
  readOnly: Boolean,
  requests: {
    serverStatus: Object,
  },
}
*/

// Reducers
const bannerMessageReducer = createReducer('', {
  [GET_SERVER_STATUS_SUCCESS]:
    (_state, action) => action.serverStatus.bannerMessage || '',
});

const bannerTimeReducer = createReducer(0, {
  [GET_SERVER_STATUS_SUCCESS]:
    (_state, action) => action.serverStatus.bannerTime || 0,
});

/**
 * Handle state for the current document title.
 */
const pageTitleReducer = createReducer('', {
  [SET_PAGE_TITLE]: (_state, {title}) => title,
});

const headerTitleReducer = createReducer('', {
  [SET_HEADER_TITLE]: (_state, {title}) => title,
});

const queryParamsReducer = createReducer({}, {
  [SET_QUERY_PARAMS]: (_state, {queryParams}) => queryParams || {},
});

const readOnlyReducer = createReducer(false, {
  [GET_SERVER_STATUS_SUCCESS]:
    (_state, action) => action.serverStatus.readOnly || false,
});

const requestsReducer = combineReducers({
  serverStatus: createRequestReducer(
      GET_SERVER_STATUS_START,
      GET_SERVER_STATUS_SUCCESS,
      GET_SERVER_STATUS_FAILURE),
});

export const reducer = combineReducers({
  bannerMessage: bannerMessageReducer,
  bannerTime: bannerTimeReducer,
  readOnly: readOnlyReducer,
  queryParams: queryParamsReducer,
  pageTitle: pageTitleReducer,
  headerTitle: headerTitleReducer,

  requests: requestsReducer,
});

// Selectors
export const sitewide = (state) => state.sitewide || {};
export const bannerMessage =
    createSelector(sitewide, (sitewide) => sitewide.bannerMessage);
export const bannerTime =
    createSelector(sitewide, (sitewide) => sitewide.bannerTime);
export const queryParams =
    createSelector(sitewide, (sitewide) => sitewide.queryParams || {});
export const pageTitle = createSelector(
    sitewide, project.viewedConfig,
    (sitewide, projectConfig) => {
      const titlePieces = [];

      // If a specific page specifies its own page title, add that
      // to the beginning of the title.
      if (sitewide.pageTitle) {
        titlePieces.push(sitewide.pageTitle);
      }

      // If the user is viewing a project, add the project data.
      if (projectConfig && projectConfig.projectName) {
        titlePieces.push(projectConfig.projectName);
      }

      return titlePieces.join(' - ') || 'Monorail';
    });
export const headerTitle =
    createSelector(sitewide, (sitewide) => sitewide.headerTitle);
export const readOnly =
    createSelector(sitewide, (sitewide) => sitewide.readOnly);

/**
 * Computes the issue list columns from the URL parameters.
 */
export const currentColumns = createSelector(
    queryParams,
    (params = {}) => params.colspec ? parseColSpec(params.colspec) : null);

/**
* Get the default canned query for the currently viewed project.
* Note: Projects cannot configure a per-project default canned query,
* so there is only a sitewide default.
*/
export const currentCan = createSelector(queryParams,
    (params) => params.can || SITEWIDE_DEFAULT_CAN);

/**
 * Compute the current issue search query that the user has
 * entered for a project, based on queryParams and the default
 * project search.
 */
export const currentQuery = createSelector(
    project.defaultQuery,
    queryParams,
    (defaultQuery, params = {}) => {
      // Make sure entering an empty search still works.
      if (params.q === '') return params.q;
      return params.q || defaultQuery;
    });

export const requests = createSelector(sitewide,
    (sitewide) => sitewide.requests || {});

// Action Creators
export const setQueryParams =
    (queryParams) => ({type: SET_QUERY_PARAMS, queryParams});

export const setPageTitle = (title) => ({type: SET_PAGE_TITLE, title});

export const setHeaderTitle = (title) => ({type: SET_HEADER_TITLE, title});

export const getServerStatus = () => async (dispatch) => {
  dispatch({type: GET_SERVER_STATUS_START});

  try {
    const serverStatus = await prpcClient.call(
        'monorail.Sitewide', 'GetServerStatus', {});

    dispatch({type: GET_SERVER_STATUS_SUCCESS, serverStatus});
  } catch (error) {
    dispatch({type: GET_SERVER_STATUS_FAILURE, error});
  }
};
