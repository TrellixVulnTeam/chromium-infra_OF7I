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
const SET_QUERY_PARAMS = 'SET_QUERY_PARAMS';

// Async actions
const GET_SERVER_STATUS_FAILURE = 'GET_SERVER_STATUS_FAILURE';
const GET_SERVER_STATUS_START = 'GET_SERVER_STATUS_START';
const GET_SERVER_STATUS_SUCCESS = 'GET_SERVER_STATUS_SUCCESS';

/* State Shape
{
  bannerMessage: String,
  bannerTime: Number,
  pageTitle: String,
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
  [SET_PAGE_TITLE]: (_state, action) => action.title || '',
});

const queryParamsReducer = createReducer({}, {
  [SET_QUERY_PARAMS]: (_state, action) => action.queryParams || {},
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

  requests: requestsReducer,
});

// Selectors
export const sitewide = (state) => state.sitewide || {};
export const bannerMessage = createSelector(sitewide,
    (sitewide) => sitewide.bannerMessage);
export const bannerTime = createSelector(sitewide,
    (sitewide) => sitewide.bannerTime);
export const queryParams = createSelector(sitewide,
    (sitewide) => sitewide.queryParams || {});
export const pageTitle = createSelector(sitewide,
    project.config, project.presentationConfig,
    (sitewide, projectConfig, presentationConfig) => {
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

      // If the viewed project has a defined summary, add that summary.
      if (presentationConfig && presentationConfig.projectSummary) {
        titlePieces.push(presentationConfig.projectSummary);
      }

      // TODO(crbug.com/monorail/6470): Change this to be Monorail
      // Local/Dev/Staging/Prod.
      titlePieces.push('Monorail');
      return titlePieces.join(' - ');
    });
export const readOnly = createSelector(sitewide,
    (sitewide) => sitewide.readOnly);

/**
 * Compute the current columns that the user is viewing in the list
 * view, based on default columns and URL parameters.
 */
export const currentColumns = createSelector(
    project.defaultColumns,
    queryParams,
    (defaultColumns, params = {}) => {
      if (params.colspec) {
        return parseColSpec(params.colspec);
      }
      return defaultColumns;
    });

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
export const setQueryParams = (params) => {
  return {
    type: SET_QUERY_PARAMS,
    queryParams: params,
  };
};

export const setPageTitle = (title) => {
  return {
    type: SET_PAGE_TITLE,
    title,
  };
};

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
