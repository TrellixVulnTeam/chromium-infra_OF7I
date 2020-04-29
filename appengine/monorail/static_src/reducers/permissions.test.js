// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as permissions from './permissions.js';
import * as example from 'shared/test/constants-permissions.js';
import * as exampleIssues from 'shared/test/constants-issueV0.js';

import {prpcClient} from 'prpc-client-instance.js';

let dispatch;

describe('permissions reducers', () => {
  it('root reducer initial state', () => {
    const actual = permissions.reducer(undefined, {type: null});
    const expected = {
      byName: {},
      requests: {
        batchGet: {error: null, requesting: false},
      },
    };
    assert.deepEqual(actual, expected);
  });

  it('byName updates on BATCH_GET_SUCCESS', () => {
    const action = {
      type: permissions.BATCH_GET_SUCCESS,
      permissionSets: [example.PERMISSION_SET_ISSUE],
    };
    const actual = permissions.byNameReducer({}, action);
    const expected = {
      [example.PERMISSION_SET_ISSUE.resource]: example.PERMISSION_SET_ISSUE,
    };
    assert.deepEqual(actual, expected);
  });
});

describe('permissions selectors', () => {
  it('byName', () => {
    const state = {permissions: {byName: example.BY_NAME}};
    const actual = permissions.byName(state);
    assert.deepEqual(actual, example.BY_NAME);
  });
});

describe('permissions action creators', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');
    dispatch = sinon.stub();
  });

  afterEach(() => {
    prpcClient.call.restore();
  });

  describe('batchGet', () => {
    it('success', async () => {
      const response = {permissionSets: [example.PERMISSION_SET_ISSUE]};
      prpcClient.call.returns(Promise.resolve(response));

      await permissions.batchGet([exampleIssues.NAME])(dispatch);

      sinon.assert.calledWith(dispatch, {type: permissions.BATCH_GET_START});

      const args = {names: [exampleIssues.NAME]};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Permissions',
          'BatchGetPermissionSets', args);

      const action = {
        type: permissions.BATCH_GET_SUCCESS,
        permissionSets: [example.PERMISSION_SET_ISSUE],
      };
      sinon.assert.calledWith(dispatch, action);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await permissions.batchGet(exampleIssues.NAME)(dispatch);

      const action = {
        type: permissions.BATCH_GET_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });
});
