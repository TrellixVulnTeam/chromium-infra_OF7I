// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as projects from './projects.js';
import * as example from 'shared/test/constants-projects.js';

import {prpcClient} from 'prpc-client-instance.js';

let dispatch;


describe('project reducers', () => {
  it('root reducer initial state', () => {
    const actual = projects.reducer(undefined, {type: null});
    const expected = {
      byName: {},
      allNames: [],
      requests: {
        list: {
          error: null,
          requesting: false,
        },
      },
    };
    assert.deepEqual(actual, expected);
  });

  describe('byNameReducer', () => {
    it('populated on LIST_SUCCESS', () => {
      const action = {type: projects.LIST_SUCCESS, projects:
          [example.PROJECT, example.PROJECT_2]};
      const actual = projects.byNameReducer({}, action);

      assert.deepEqual(actual, {
        [example.NAME]: example.PROJECT,
        [example.NAME_2]: example.PROJECT_2,
      });
    });

    it('keeps original state on empty LIST_SUCCESS', () => {
      const originalState = {
        [example.NAME]: example.PROJECT,
        [example.NAME_2]: example.PROJECT_2,
      };
      const action = {type: projects.LIST_SUCCESS, projects: []};
      const actual = projects.byNameReducer(originalState, action);

      assert.deepEqual(actual, originalState);
    });

    it('appends new issues to state on LIST_SUCCESS', () => {
      const originalState = {
        [example.NAME]: example.PROJECT,
      };
      const action = {type: projects.LIST_SUCCESS,
        projects: [example.PROJECT_2]};
      const actual = projects.byNameReducer(originalState, action);

      const expected = {
        [example.NAME]: example.PROJECT,
        [example.NAME_2]: example.PROJECT_2,
      };
      assert.deepEqual(actual, expected);
    });

    it('overrides outdated data on LIST_SUCCESS', () => {
      const originalState = {
        [example.NAME]: example.PROJECT,
        [example.NAME_2]: example.PROJECT_2,
      };

      const newProject2 = {
        name: example.NAME_2,
        summary: 'I hacked your project!',
      };
      const action = {type: projects.LIST_SUCCESS,
        projects: [newProject2]};
      const actual = projects.byNameReducer(originalState, action);
      const expected = {
        [example.NAME]: example.PROJECT,
        [example.NAME_2]: newProject2,
      };
      assert.deepEqual(actual, expected);
    });
  });

  it('allNames populated on LIST_SUCCESS', () => {
    const action = {type: projects.LIST_SUCCESS, projects:
        [example.PROJECT, example.PROJECT_2]};
    const actual = projects.allNamesReducer([], action);

    assert.deepEqual(actual, [example.NAME, example.NAME_2]);
  });
});

describe('project selectors', () => {
  it('byName', () => {
    const normalizedProjects = {
      [example.NAME]: example.PROJECT,
    };
    const state = {projects: {
      byName: normalizedProjects,
    }};
    assert.deepEqual(projects.byName(state), normalizedProjects);
  });

  it('all', () => {
    const state = {projects: {
      byName: {
        [example.NAME]: example.PROJECT,
      },
      allNames: [example.NAME],
    }};
    assert.deepEqual(projects.all(state), [example.PROJECT]);
  });

  it('requests', () => {
    const state = {projects: {
      requests: {
        list: {error: null, requesting: false},
      },
    }};
    assert.deepEqual(projects.requests(state), {
      list: {error: null, requesting: false},
    });
  });
});

describe('project action creators', () => {
  beforeEach(() => {
    sinon.stub(prpcClient, 'call');
    dispatch = sinon.stub();
  });

  afterEach(() => {
    prpcClient.call.restore();
  });

  describe('list', () => {
    it('success', async () => {
      const projectsResponse = {projects: [example.PROJECT, example.PROJECT_2]};
      prpcClient.call.returns(Promise.resolve(projectsResponse));

      await projects.list()(dispatch);

      sinon.assert.calledWith(dispatch, {type: projects.LIST_START});

      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v3.Projects', 'ListProjects', {});

      const successAction = {
        type: projects.LIST_SUCCESS,
        projects: projectsResponse.projects,
      };
      sinon.assert.calledWith(dispatch, successAction);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await projects.list()(dispatch);

      const action = {
        type: projects.LIST_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });
});
