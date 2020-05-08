// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview User actions, selectors, and reducers organized into
 * a single Redux "Duck" that manages updating and retrieving user state
 * on the frontend.
 *
 * The User data is stored in a normalized format.
 * `byName` stores all User data indexed by User name.
 * `user` is a selector that gets the currently viewed User data.
 *
 * Reference: https://github.com/erikras/ducks-modular-redux
 */

import {combineReducers} from 'redux';
import {createReducer, createKeyedRequestReducer} from './redux-helpers.js';
import {prpcClient} from 'prpc-client-instance.js';
import 'shared/typedef.js';

/** @typedef {import('redux').AnyAction} AnyAction */

// Actions
export const BATCH_GET_START = 'user/BATCH_GET_START';
export const BATCH_GET_SUCCESS = 'user/BATCH_GET_SUCCESS';
export const BATCH_GET_FAILURE = 'user/BATCH_GET_FAILURE';

export const GATHER_PROJECT_MEMBERSHIPS_START =
  'user/GATHER_PROJECT_MEMBERSHIPS_START';
export const GATHER_PROJECT_MEMBERSHIPS_SUCCESS =
  'user/GATHER_PROJECT_MEMBERSHIPS_SUCCESS';
export const GATHER_PROJECT_MEMBERSHIPS_FAILURE =
  'user/GATHER_PROJECT_MEMBERSHIPS_FAILURE';

/* State Shape
{
  byName: Object<UserName, User>,

  requests: {
    batchGet: ReduxRequestState,
    gatherProjectMemberships: ReduxRequestState,
  },
}
*/

// Reducers

/**
 * All User data indexed by User name.
 * @param {Object<UserName, User>} state The existing User data.
 * @param {AnyAction} action
 * @param {User} action.user The user that was fetched.
 * @return {Object<UserName, User>}
 */
export const byNameReducer = createReducer({}, {
  [BATCH_GET_SUCCESS]: (state, {users}) => {
    const newState = {...state};
    for (const user of users) {
      newState[user.name] = user;
    }
    return newState;
  },
});

/**
 * ProjectMember data indexed by User name.
 *
 * Pragma: No normalization for ProjectMember objects. There is never a
 *  situation when we will have access to ProjectMember names but not associated
 *  ProjectMember objects so normalizing is unnecessary.
 * @param {Object<UserName, Array<ProjectMember>>} state The existing User data.
 * @param {AnyAction} action
 * @param {UserName} action.userName The resource name of the user that was
 *   fetched.
 * @param {Array<ProjectMember>=} action.projectMemberships The project
 *   memberships for the fetched user.
 * @return {Object<UserName, Array<ProjectMember>>}
 */
export const projectMembershipsReducer = createReducer({}, {
  [GATHER_PROJECT_MEMBERSHIPS_SUCCESS]: (state, {userName,
    projectMemberships}) => {
    const newState = {...state};

    newState[userName] = projectMemberships || [];
    return newState;
  },
});

const requestsReducer = combineReducers({
  batchGet: createKeyedRequestReducer(
      BATCH_GET_START, BATCH_GET_SUCCESS, BATCH_GET_FAILURE),
  gatherProjectMemberships: createKeyedRequestReducer(
      GATHER_PROJECT_MEMBERSHIPS_START, GATHER_PROJECT_MEMBERSHIPS_SUCCESS,
      GATHER_PROJECT_MEMBERSHIPS_FAILURE),
});

export const reducer = combineReducers({
  byName: byNameReducer,
  projectMemberships: projectMembershipsReducer,

  requests: requestsReducer,
});

// Selectors

/**
 * Returns all the User data in the store as a mapping from name to User.
 * @param {any} state
 * @return {Object<UserName, User>}
 */
export const byName = (state) => state.users.byName;

/**
 * Returns all the ProjectMember data in the store, mapped to Users' names.
 * @param {any} state
 * @return {Object<UserName, ProjectMember>}
 */
export const projectMemberships = (state) => state.users.projectMemberships;

// Action Creators

/**
 * Action creator to fetch multiple User objects.
 * @param {Array<UserName>} names The names of the Users to fetch.
 * @return {function(function): Promise<void>}
 */
export const batchGet = (names) => async (dispatch) => {
  dispatch({type: BATCH_GET_START});

  try {
    /** @type {{users: Array<User>}} */
    const {users} = await prpcClient.call(
        'monorail.v1.Users', 'BatchGetUsers', {names});

    dispatch({type: BATCH_GET_SUCCESS, users});
  } catch (error) {
    dispatch({type: BATCH_GET_FAILURE, error});
  };
};

/**
 * Action creator to fetch ProjectMember objects for a given User.
 * @param {UserName} name The resource name of the User.
 * @return {function(function): Promise<void>}
 */
export const gatherProjectMemberships = (name) => async (dispatch) => {
  dispatch({type: GATHER_PROJECT_MEMBERSHIPS_START});

  try {
    /** @type {{projectMemberships: Array<ProjectMember>}} */
    const {projectMemberships} = await prpcClient.call(
        'monorail.v1.Frontend', 'GatherProjectMembershipsForUser',
        {user: name});

    dispatch({type: GATHER_PROJECT_MEMBERSHIPS_SUCCESS,
      userName: name, projectMemberships});
  } catch (error) {
    // TODO(crbug.com/monorail/7627): Catch actual API errors.
    dispatch({type: GATHER_PROJECT_MEMBERSHIPS_FAILURE, error});
  };
};
