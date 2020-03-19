// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import * as userV0 from './userV0.js';
import {prpcClient} from 'prpc-client-instance.js';


let dispatch;

describe('user', () => {
  describe('reducers', () => {
    it('SET_PREFS_SUCCESS updates existing prefs with new prefs', () => {
      const state = {prefs: {
        testPref: 'true',
        anotherPref: 'hello-world',
      }};

      const newPrefs = [
        {name: 'anotherPref', value: 'override'},
        {name: 'newPref', value: 'test-me'},
      ];

      const newState = userV0.currentUserReducer(state,
          {type: userV0.SET_PREFS_SUCCESS, newPrefs});

      assert.deepEqual(newState, {prefs: {
        testPref: 'true',
        anotherPref: 'override',
        newPref: 'test-me',
      }});
    });

    it('FETCH_PROJECTS_SUCCESS overrides existing entry in usersById', () => {
      const state = {
        ['123']: {
          projects: {
            ownerOf: [],
            memberOf: [],
            contributorTo: [],
            starredProjects: [],
          },
        },
      };

      const usersProjects = [
        {
          userRef: {userId: '123'},
          ownerOf: ['chromium'],
        },
      ];

      const newState = userV0.usersByIdReducer(state,
          {type: userV0.FETCH_PROJECTS_SUCCESS, usersProjects});

      assert.deepEqual(newState, {
        ['123']: {
          projects: {
            ownerOf: ['chromium'],
            memberOf: [],
            contributorTo: [],
            starredProjects: [],
          },
        },
      });
    });

    it('FETCH_PROJECTS_SUCCESS adds new entry to usersById', () => {
      const state = {
        ['123']: {
          projects: {
            ownerOf: [],
            memberOf: [],
            contributorTo: [],
            starredProjects: [],
          },
        },
      };

      const usersProjects = [
        {
          userRef: {userId: '543'},
          ownerOf: ['chromium'],
        },
        {
          userRef: {userId: '789'},
          memberOf: ['v8'],
        },
      ];

      const newState = userV0.usersByIdReducer(state,
          {type: userV0.FETCH_PROJECTS_SUCCESS, usersProjects});

      assert.deepEqual(newState, {
        ['123']: {
          projects: {
            ownerOf: [],
            memberOf: [],
            contributorTo: [],
            starredProjects: [],
          },
        },
        ['543']: {
          projects: {
            ownerOf: ['chromium'],
            memberOf: [],
            contributorTo: [],
            starredProjects: [],
          },
        },
        ['789']: {
          projects: {
            ownerOf: [],
            memberOf: ['v8'],
            contributorTo: [],
            starredProjects: [],
          },
        },
      });
    });

    describe('GAPI_LOGIN_SUCCESS', () => {
      it('sets currentUser.gapiEmail', () => {
        const newState = userV0.currentUserReducer({}, {
          type: userV0.GAPI_LOGIN_SUCCESS,
          email: 'rutabaga@rutabaga.com',
        });
        assert.deepEqual(newState, {
          gapiEmail: 'rutabaga@rutabaga.com',
        });
      });

      it('defaults to an empty string', () => {
        const newState = userV0.currentUserReducer({}, {
          type: userV0.GAPI_LOGIN_SUCCESS,
        });
        assert.deepEqual(newState, {
          gapiEmail: '',
        });
      });
    });

    describe('GAPI_LOGOUT_SUCCESS', () => {
      it('sets currentUser.gapiEmail', () => {
        const newState = userV0.currentUserReducer({}, {
          type: userV0.GAPI_LOGOUT_SUCCESS,
          email: '',
        });
        assert.deepEqual(newState, {
          gapiEmail: '',
        });
      });

      it('defaults to an empty string', () => {
        const state = {};
        const newState = userV0.currentUserReducer(state, {
          type: userV0.GAPI_LOGOUT_SUCCESS,
        });
        assert.deepEqual(newState, {
          gapiEmail: '',
        });
      });
    });
  });

  describe('selectors', () => {
    it('prefs', () => {
      const state = wrapCurrentUser({prefs: {
        testPref: 'true',
        anotherPref: 'hello-world',
      }});

      assert.deepEqual(userV0.prefs(state), new Map([
        ['testPref', 'true'],
        ['anotherPref', 'hello-world'],
      ]));
    });

    it('projects', () => {
      assert.deepEqual(userV0.projects(wrapUser({})), {});

      const state = wrapUser({
        currentUser: {userId: '123'},
        usersById: {
          ['123']: {
            projects: {
              ownerOf: ['chromium'],
              memberOf: ['v8'],
              contributorTo: [],
              starredProjects: [],
            },
          },
        },
      });

      assert.deepEqual(userV0.projects(state), {
        ownerOf: ['chromium'],
        memberOf: ['v8'],
        contributorTo: [],
        starredProjects: [],
      });
    });

    it('projectPerUser', () => {
      assert.deepEqual(userV0.projectsPerUser(wrapUser({})), new Map());

      const state = wrapUser({
        usersById: {
          ['123']: {
            projects: {
              ownerOf: ['chromium'],
              memberOf: ['v8'],
              contributorTo: [],
              starredProjects: [],
            },
          },
        },
      });

      assert.deepEqual(userV0.projectsPerUser(state), new Map([
        ['123', {
          ownerOf: ['chromium'],
          memberOf: ['v8'],
          contributorTo: [],
          starredProjects: [],
        }],
      ]));
    });
  });

  describe('action creators', () => {
    beforeEach(() => {
      sinon.stub(prpcClient, 'call');

      dispatch = sinon.stub();
    });

    afterEach(() => {
      prpcClient.call.restore();
    });

    it('fetchProjects succeeds', async () => {
      const action = userV0.fetchProjects([{userId: '123'}]);

      prpcClient.call.returns(Promise.resolve({
        usersProjects: [
          {
            userRef: {
              userId: '123',
            },
            ownerOf: ['chromium'],
          },
        ],
      }));

      await action(dispatch);

      sinon.assert.calledWith(dispatch, {type: userV0.FETCH_PROJECTS_START});

      sinon.assert.calledWith(
          prpcClient.call,
          'monorail.Users',
          'GetUsersProjects',
          {userRefs: [{userId: '123'}]});

      sinon.assert.calledWith(dispatch, {
        type: userV0.FETCH_PROJECTS_SUCCESS,
        usersProjects: [
          {
            userRef: {
              userId: '123',
            },
            ownerOf: ['chromium'],
          },
        ],
      });
    });

    it('fetchProjects fails', async () => {
      const action = userV0.fetchProjects([{userId: '123'}]);

      const error = new Error('mistakes were made');
      prpcClient.call.returns(Promise.reject(error));

      await action(dispatch);

      sinon.assert.calledWith(dispatch, {type: userV0.FETCH_PROJECTS_START});

      sinon.assert.calledWith(
          prpcClient.call,
          'monorail.Users',
          'GetUsersProjects',
          {userRefs: [{userId: '123'}]});

      sinon.assert.calledWith(dispatch, {
        type: userV0.FETCH_PROJECTS_FAILURE,
        error,
      });
    });

    it('setPrefs', async () => {
      const action = userV0.setPrefs([{name: 'pref_name', value: 'true'}]);

      prpcClient.call.returns(Promise.resolve({}));

      await action(dispatch);

      sinon.assert.calledWith(dispatch, {type: userV0.SET_PREFS_START});

      sinon.assert.calledWith(
          prpcClient.call,
          'monorail.Users',
          'SetUserPrefs',
          {prefs: [{name: 'pref_name', value: 'true'}]});

      sinon.assert.calledWith(dispatch, {
        type: userV0.SET_PREFS_SUCCESS,
        newPrefs: [{name: 'pref_name', value: 'true'}],
      });
    });

    it('setPrefs fails', async () => {
      const action = userV0.setPrefs([{name: 'pref_name', value: 'true'}]);

      const error = new Error('mistakes were made');
      prpcClient.call.returns(Promise.reject(error));

      await action(dispatch);

      sinon.assert.calledWith(dispatch, {type: userV0.SET_PREFS_START});

      sinon.assert.calledWith(
          prpcClient.call,
          'monorail.Users',
          'SetUserPrefs',
          {prefs: [{name: 'pref_name', value: 'true'}]});

      sinon.assert.calledWith(dispatch, {
        type: userV0.SET_PREFS_FAILURE,
        error,
      });
    });
  });
});


const wrapCurrentUser = (currentUser = {}) => ({user: {currentUser}});
const wrapUser = (user) => ({user});
