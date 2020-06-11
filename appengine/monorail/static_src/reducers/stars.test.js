// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as stars from './stars.js';
import * as example from 'shared/test/constants-stars.js';

import {prpcClient} from 'prpc-client-instance.js';

let dispatch;


describe('star reducers', () => {
  it('root reducer initial state', () => {
    const actual = stars.reducer(undefined, {type: null});
    const expected = {
      byName: {},
      requests: {
        listProjects: {
          error: null,
          requesting: false,
        },
      },
    };
    assert.deepEqual(actual, expected);
  });

  describe('byNameReducer', () => {
    it('populated on LIST_PROJECTS_SUCCESS', () => {
      const action = {type: stars.LIST_PROJECTS_SUCCESS, stars:
          [example.PROJECT_STAR, example.PROJECT_STAR_2]};
      const actual = stars.byNameReducer({}, action);

      assert.deepEqual(actual, {
        [example.PROJECT_STAR_NAME]: example.PROJECT_STAR,
        [example.PROJECT_STAR_NAME_2]: example.PROJECT_STAR_2,
      });
    });

    it('keeps original state on empty LIST_PROJECTS_SUCCESS', () => {
      const originalState = {
        [example.PROJECT_STAR_NAME]: example.PROJECT_STAR,
        [example.PROJECT_STAR_NAME_2]: example.PROJECT_STAR_2,
      };
      const action = {type: stars.LIST_PROJECTS_SUCCESS, stars: []};
      const actual = stars.byNameReducer(originalState, action);

      assert.deepEqual(actual, originalState);
    });

    it('appends new stars to state on LIST_PROJECTS_SUCCESS', () => {
      const originalState = {
        [example.PROJECT_STAR_NAME]: example.PROJECT_STAR,
      };
      const action = {type: stars.LIST_PROJECTS_SUCCESS,
        stars: [example.PROJECT_STAR_2]};
      const actual = stars.byNameReducer(originalState, action);

      const expected = {
        [example.PROJECT_STAR_NAME]: example.PROJECT_STAR,
        [example.PROJECT_STAR_NAME_2]: example.PROJECT_STAR_2,
      };
      assert.deepEqual(actual, expected);
    });
  });
});

describe('project selectors', () => {
  it('byName', () => {
    const normalizedStars = {
      [example.PROJECT_STAR_NAME]: example.PROJECT_STAR,
    };
    const state = {stars: {
      byName: normalizedStars,
    }};
    assert.deepEqual(stars.byName(state), normalizedStars);
  });


  it('requests', () => {
    const state = {stars: {
      requests: {
        listProjects: {error: null, requesting: false},
      },
    }};
    assert.deepEqual(stars.requests(state), {
      listProjects: {error: null, requesting: false},
    });
  });
});

describe('star action creators', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');
    dispatch = sinon.stub();
  });

  afterEach(() => {
    prpcClient.call.restore();
  });

  describe('listProjects', () => {
    it('success', async () => {
      const starsResponse = {
        projectStars: [example.PROJECT_STAR, example.PROJECT_STAR_2],
      };
      prpcClient.call.returns(Promise.resolve(starsResponse));

      await stars.listProjects('users/1234')(dispatch);

      sinon.assert.calledWith(dispatch, {type: stars.LIST_PROJECTS_START});

      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Users', 'ListProjectStars',
          {parent: 'users/1234'});

      const successAction = {
        type: stars.LIST_PROJECTS_SUCCESS,
        stars: [example.PROJECT_STAR, example.PROJECT_STAR_2],
      };
      sinon.assert.calledWith(dispatch, successAction);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await stars.listProjects('users/1234')(dispatch);

      const action = {
        type: stars.LIST_PROJECTS_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });
});
