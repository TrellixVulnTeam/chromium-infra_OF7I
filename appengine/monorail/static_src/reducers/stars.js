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
import {createReducer, createRequestReducer,
  createKeyedRequestReducer} from './redux-helpers.js';
import {prpcClient} from 'prpc-client-instance.js';
import {projectAndUserToStarName} from 'shared/converters.js';
import 'shared/typedef.js';

/** @typedef {import('redux').AnyAction} AnyAction */

// Actions
export const LIST_PROJECTS_START = 'stars/LIST_PROJECTS_START';
export const LIST_PROJECTS_SUCCESS = 'stars/LIST_PROJECTS_SUCCESS';
export const LIST_PROJECTS_FAILURE = 'stars/LIST_PROJECTS_FAILURE';

export const STAR_PROJECT_START = 'stars/STAR_PROJECT_START';
export const STAR_PROJECT_SUCCESS = 'stars/STAR_PROJECT_SUCCESS';
export const STAR_PROJECT_FAILURE = 'stars/STAR_PROJECT_FAILURE';

export const UNSTAR_PROJECT_START = 'stars/UNSTAR_PROJECT_START';
export const UNSTAR_PROJECT_SUCCESS = 'stars/UNSTAR_PROJECT_SUCCESS';
export const UNSTAR_PROJECT_FAILURE = 'stars/UNSTAR_PROJECT_FAILURE';

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
 * @param {ProjectStar} action.projectStar A single ProjectStar that was
 *   created.
 * @param {StarName} action.starName The StarName that was mutated.
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
  [STAR_PROJECT_SUCCESS]: (state, {projectStar}) => {
    return {...state, [projectStar.name]: projectStar};
  },
  [UNSTAR_PROJECT_SUCCESS]: (state, {starName}) => {
    const newState = {...state};
    delete newState[starName];
    return newState;
  },
});


const requestsReducer = combineReducers({
  listProjects: createRequestReducer(LIST_PROJECTS_START,
      LIST_PROJECTS_SUCCESS, LIST_PROJECTS_FAILURE),
  starProject: createKeyedRequestReducer(STAR_PROJECT_START,
      STAR_PROJECT_SUCCESS, STAR_PROJECT_FAILURE),
  unstarProject: createKeyedRequestReducer(UNSTAR_PROJECT_START,
      UNSTAR_PROJECT_SUCCESS, UNSTAR_PROJECT_FAILURE),
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

/**
 * Stars a given project.
 * @param {ProjectName} project The resource name of the project to star.
 * @param {UserName} user The resource name of the user who is starring
 *   the issue. This will always be the currently logged in user.
 * @return {function(function): Promise<void>}
 */
export const starProject = (project, user) => async (dispatch) => {
  const requestKey = projectAndUserToStarName(project, user);
  dispatch({type: STAR_PROJECT_START, requestKey});
  try {
    const projectStar = await prpcClient.call(
        'monorail.v3.Users', 'StarProject', {project});
    dispatch({type: STAR_PROJECT_SUCCESS, requestKey, projectStar});
  } catch (error) {
    dispatch({type: STAR_PROJECT_FAILURE, requestKey, error});
  };
};

/**
 * Unstars a given project.
 * @param {ProjectName} project The resource name of the project to unstar.
 * @param {UserName} user The resource name of the user who is unstarring
 *   the issue. This will always be the currently logged in user, but
 *   passing in the user's resource name is necessary to make it possible to
 *   generate the resource name of the removed star.
 * @return {function(function): Promise<void>}
 */
export const unstarProject = (project, user) => async (dispatch) => {
  const starName = projectAndUserToStarName(project, user);
  const requestKey = starName;
  dispatch({type: UNSTAR_PROJECT_START, requestKey});

  try {
    await prpcClient.call(
        'monorail.v3.Users', 'UnStarProject', {project});
    dispatch({type: UNSTAR_PROJECT_SUCCESS, requestKey, starName});
  } catch (error) {
    dispatch({type: UNSTAR_PROJECT_FAILURE, requestKey, error});
  };
};

export const stars = {
  byName,
  requests,
  listProjects,
  starProject,
  unstarProject,
};
