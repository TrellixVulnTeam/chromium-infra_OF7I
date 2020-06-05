// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as users from './users.js';
import * as example from 'shared/test/constants-users.js';

import {prpcClient} from 'prpc-client-instance.js';

let dispatch;

describe('user reducers', () => {
  it('root reducer initial state', () => {
    const actual = users.reducer(undefined, {type: null});
    const expected = {
      currentUserName: {},
      byName: {},
      projectMemberships: {},
      requests: {
        batchGet: {},
        fetch: {},
        gatherProjectMemberships: {},
      },
    };
    assert.deepEqual(actual, expected);
  });

  it('byName updates on BATCH_GET_SUCCESS', () => {
    const action = {type: users.BATCH_GET_SUCCESS, users: [example.USER]};
    const actual = users.byNameReducer({}, action);
    assert.deepEqual(actual, {[example.NAME]: example.USER});
  });

  describe('projectMembershipsReducer', () => {
    it('updates on GATHER_PROJECT_MEMBERSHIPS_SUCCESS', () => {
      const action = {type: users.GATHER_PROJECT_MEMBERSHIPS_SUCCESS,
        userName: example.NAME, projectMemberships: [example.PROJECT_MEMBER]};
      const actual = users.projectMembershipsReducer({}, action);
      assert.deepEqual(actual, {[example.NAME]: [example.PROJECT_MEMBER]});
    });

    it('sets empty on GATHER_PROJECT_MEMBERSHIPS_SUCCESS', () => {
      const action = {type: users.GATHER_PROJECT_MEMBERSHIPS_SUCCESS,
        userName: example.NAME, projectMemberships: undefined};
      const actual = users.projectMembershipsReducer({}, action);
      assert.deepEqual(actual, {[example.NAME]: []});
    });
  });
});

describe('user selectors', () => {
  it('byName', () => {
    const state = {users: {byName: example.BY_NAME}};
    assert.deepEqual(users.byName(state), example.BY_NAME);
  });

  it('projectMemberships', () => {
    const membershipsByName = {[example.NAME]: [example.PROJECT_MEMBER]};
    const state = {users: {projectMemberships: membershipsByName}};
    assert.deepEqual(users.projectMemberships(state), membershipsByName);
  });
});

describe('user action creators', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');
    dispatch = sinon.stub();
  });

  afterEach(() => {
    prpcClient.call.restore();
  });

  describe('batchGet', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve({users: [example.USER]}));

      await users.batchGet([example.NAME])(dispatch);

      sinon.assert.calledWith(dispatch, {type: users.BATCH_GET_START});

      const args = {names: [example.NAME]};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Users', 'BatchGetUsers', args);

      const action = {type: users.BATCH_GET_SUCCESS, users: [example.USER]};
      sinon.assert.calledWith(dispatch, action);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await users.batchGet([example.NAME])(dispatch);

      const action = {type: users.BATCH_GET_FAILURE, error: sinon.match.any};
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('fetch', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve(example.USER));

      await users.fetch(example.NAME)(dispatch);

      sinon.assert.calledWith(dispatch, {type: users.FETCH_START});

      const args = {name: example.NAME};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Users', 'GetUser', args);

      const fetchAction = {type: users.FETCH_SUCCESS, user: example.USER};
      sinon.assert.calledWith(dispatch, fetchAction);

      const logInAction = {type: users.LOG_IN, user: example.USER};
      sinon.assert.calledWith(dispatch, logInAction);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await users.fetch(example.NAME)(dispatch);

      const action = {type: users.FETCH_FAILURE, error: sinon.match.any};
      sinon.assert.calledWith(dispatch, action);
    });
  });

  describe('gatherProjectMemberships', () => {
    it('success', async () => {
      prpcClient.call.returns(Promise.resolve({projectMemberships: [
        example.PROJECT_MEMBER,
      ]}));

      await users.gatherProjectMemberships(
          example.NAME)(dispatch);

      sinon.assert.calledWith(dispatch,
          {type: users.GATHER_PROJECT_MEMBERSHIPS_START});

      const args = {user: example.NAME};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Frontend',
          'GatherProjectMembershipsForUser', args);

      const action = {
        type: users.GATHER_PROJECT_MEMBERSHIPS_SUCCESS,
        projectMemberships: [example.PROJECT_MEMBER],
        userName: example.NAME,
      };
      sinon.assert.calledWith(dispatch, action);
    });

    it('failure', async () => {
      prpcClient.call.throws(new Error());

      await users.batchGet([example.NAME])(dispatch);

      const action = {type: users.BATCH_GET_FAILURE,
        error: sinon.match.instanceOf(Error)};
      sinon.assert.calledWith(dispatch, action);
    });
  });
});
