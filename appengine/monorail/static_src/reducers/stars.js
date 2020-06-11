// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Star actions, selectors, and reducers organized into
 * a single Redux "Duck" that manages updating and retrieving star state
 * on the frontend.
 *
 * Reference: https://github.com/erikras/ducks-modular-redux
 */

import {combineReducers} from 'redux';
import {createReducer, createRequestReducer} from './redux-helpers.js';
import {prpcClient} from 'prpc-client-instance.js';
import 'shared/typedef.js';

/** @typedef {import('redux').AnyAction} AnyAction */

// Actions
export const LIST_PROJECTS_START = 'stars/LIST_PROJECTS_START';
export const LIST_PROJECTS_SUCCESS = 'stars/LIST_PROJECTS_SUCCESS';
export const LIST_PROJECTS_FAILURE = 'stars/LIST_PROJECTS_FAILURE';

/* State Shape
{
  byName: Object.<StarName, Star>,

  requests: {
    listProjects: ReduxRequestState,
  },
}
*/

/**
 * All star data indexed by resource name.
 * @param {Object.<ProjectName, Star>} state Existing Project data.
 * @param {AnyAction} action
 * @param {Array<Star>} action.star The Stars that were fetched.
 * @return {Object.<ProjectName, Star>}
 */
export const byNameReducer = createReducer({}, {
  [LIST_PROJECTS_SUCCESS]: (state, {stars}) => {
    const newStars = {};
    stars.forEach((star) => {
      newStars[star.name] = star;
    });
    return {...state, ...newStars};
  },
});


const requestsReducer = combineReducers({
  listProjects: createRequestReducer(),
});


export const reducer = combineReducers({
  byName: byNameReducer,
  requests: requestsReducer,
});


/**
 * Returns normalized star data by name.
 * @param {any} state
 * @return {Object.<StarName, Star>}
 * @private
 */
export const byName = (state) => state.stars.byName;

/**
 * Returns star requests.
 * @param {any} state
 * @return {Object.<string, ReduxRequestState>}
 */
export const requests = (state) => state.stars.requests;

/**
 * Retrieves the starred projects for a given user.
 * @param {UserName} user The resource name of the user to fetch
 *   starred projects for.
 * @return {function(function): Promise<void>}
 */
export const listProjects = (user) => async (dispatch) => {
  dispatch({type: LIST_PROJECTS_START});

  try {
    const {projectStars} = await prpcClient.call(
        'monorail.v3.Users', 'ListProjectStars', {parent: user});
    dispatch({type: LIST_PROJECTS_SUCCESS, stars: projectStars});
  } catch (error) {
    dispatch({type: LIST_PROJECTS_FAILURE, error});
  };
};


export const stars = {
  byName,
  requests,
  listProjects,
};
