// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as user from './user.js';
import * as example from 'shared/test/constants-user.js';

import {prpcClient} from 'prpc-client-instance.js';

let dispatch;

describe('user reducers', () => {
  it('root reducer initial state', () => {
    const actual = user.reducer(undefined, {type: null});
    const expected = {
      byName: {},
      requests: {
        batchGet: {},
      },
    };
    assert.deepEqual(actual, expected);
  });

  it('byName updates on BATCH_GET_SUCCESS', () => {
    const action = {type: user.BATCH_GET_SUCCESS, users: [example.USER]};
    const actual = user.byNameReducer({}, action);
    assert.deepEqual(actual, {[example.NAME]: example.USER});
  });
});

describe('user selectors', () => {
  it('byName', () => {
    const state = {user: {byName: example.BY_NAME}};
    assert.deepEqual(user.byName(state), example.BY_NAME);
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

      await user.batchGet([example.NAME])(dispatch);

      sinon.assert.calledWith(dispatch, {type: user.BATCH_GET_START});

      const args = {names: [example.NAME]};
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Users', 'BatchGetUsers', args);

      const action = {type: user.BATCH_GET_SUCCESS, users: [example.USER]};
      sinon.assert.calledWith(dispatch, action);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await user.batchGet([example.NAME])(dispatch);

      const action = {type: user.BATCH_GET_FAILURE, error: sinon.match.any};
      sinon.assert.calledWith(dispatch, action);
    });
  });
});
