// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Project actions, selectors, and reducers organized into
 * a single Redux "Duck" that manages updating and retrieving project state
 * on the frontend.
 *
 * Reference: https://github.com/erikras/ducks-modular-redux
 */

import {combineReducers} from 'redux';
import {createSelector} from 'reselect';
import {createReducer, createRequestReducer} from './redux-helpers.js';
import {prpcClient} from 'prpc-client-instance.js';
import 'shared/typedef.js';

/** @typedef {import('redux').AnyAction} AnyAction */

export const LIST_START = 'projects/LIST_START';
export const LIST_SUCCESS = 'projects/LIST_SUCCESS';
export const LIST_FAILURE = 'projects/LIST_FAILURE';

/* State Shape
{
  name: string,

  byName: Object<ProjectName, Project>,
  allNames: Array<ProjectName>,

  requests: {
    list: ReduxRequestState,
  },
}
*/

/**
 * All Project data indexed by Project name.
 * @param {Object<ProjectName, Project>} state Existing Project data.
 * @param {AnyAction} action
 * @param {Array<Project>} action.projects The Projects that were fetched.
 * @return {Object<ProjectName, Project>}
 */
export const byNameReducer = createReducer({}, {
  [LIST_SUCCESS]: (state, {projects}) => {
    const newProjects = {};
    projects.forEach((proj) => {
      newProjects[proj.name] = proj;
    });
    return {...state, ...newProjects};
  },
});

/**
 * Resource names for all Projects in Monorail.
 * @param {Array<ProjectName>} _state Existing Project data.
 * @param {AnyAction} action
 * @param {Array<Project>} action.projects The Projects that were fetched.
 * @return {Array<ProjectName>}
 */
export const allNamesReducer = createReducer([], {
  [LIST_SUCCESS]: (_state, {projects}) => {
    return projects.map((proj) => proj.name);
  },
});

const requestsReducer = combineReducers({
  list: createRequestReducer(
      LIST_START, LIST_SUCCESS, LIST_FAILURE),
});

export const reducer = combineReducers({
  byName: byNameReducer,
  allNames: allNamesReducer,

  requests: requestsReducer,
});


/**
 * Returns normalized Project data by name.
 * @param {any} state
 * @return {Object<ProjectName, Project>}
 * @private
 */
export const byName = (state) => state.project.byName;

/**
 * Base selector for wrapping the allNames state key.
 * @param {any} state
 * @return {Array<ProjectName>}
 * @private
 */
export const _allNames = (state) => state.project.allNames;

/**
 * Returns all Projects on Monorail, in denormalized form, in
 * the sort order returned by the API.
 * @param {any} state
 * @return {Array<Project>}
 */
export const all = createSelector([byName, _allNames],
    (byName, allNames) => allNames.map((name) => byName[name]));


/**
 * Gets all projects hosted on Monorail.
 * @return {function(function): Promise<Array<Project>>}
 */
export const list = () => async (dispatch) => {
  dispatch({type: LIST_START});
  try {
    /** @type {Array<Project>} */
    const projects = await prpcClient.call(
        'monorail.v1.Projects', 'ListProjects', {});

    dispatch({type: LIST_SUCCESS, projects});

    return projects;
  } catch (error) {
    dispatch({type: LIST_FAILURE, error});
  }
};
