// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as project from './project.js';
import * as example from 'shared/test/constants-project.js';

import {prpcClient} from 'prpc-client-instance.js';

let dispatch;


describe('project reducers', () => {
  it('root reducer initial state', () => {
    const actual = project.reducer(undefined, {type: null});
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
      const action = {type: project.LIST_SUCCESS, projects:
          [example.PROJECT, example.PROJECT_2]};
      const actual = project.byNameReducer({}, action);

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
      const action = {type: project.LIST_SUCCESS, projects: []};
      const actual = project.byNameReducer(originalState, action);

      assert.deepEqual(actual, originalState);
    });

    it('appends new issues to state on LIST_SUCCESS', () => {
      const originalState = {
        [example.NAME]: example.PROJECT,
      };
      const action = {type: project.LIST_SUCCESS,
        projects: [example.PROJECT_2]};
      const actual = project.byNameReducer(originalState, action);

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
      const action = {type: project.LIST_SUCCESS,
        projects: [newProject2]};
      const actual = project.byNameReducer(originalState, action);
      const expected = {
        [example.NAME]: example.PROJECT,
        [example.NAME_2]: newProject2,
      };
      assert.deepEqual(actual, expected);
    });
  });

  it('allNames populated on LIST_SUCCESS', () => {
    const action = {type: project.LIST_SUCCESS, projects:
        [example.PROJECT, example.PROJECT_2]};
    const actual = project.allNamesReducer([], action);

    assert.deepEqual(actual, [example.NAME, example.NAME_2]);
  });
});

describe('project selectors', () => {
  it('byName', () => {
    const normalizedProjects = {
      [example.NAME]: example.PROJECT,
    };
    const state = {project: {
      byName: normalizedProjects,
    }};
    assert.deepEqual(project.byName(state), normalizedProjects);
  });

  it('all', () => {
    const state = {project: {
      byName: {
        [example.NAME]: example.PROJECT,
      },
      allNames: [example.NAME],
    }};
    assert.deepEqual(project.all(state), [example.PROJECT]);
  });

  it('requests', () => {
    const state = {project: {
      requests: {
        list: {error: null, requesting: false},
      },
    }};
    assert.deepEqual(project.requests(state), {
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
      const projects = [example.PROJECT, example.PROJECT_2];
      prpcClient.call.returns(Promise.resolve({projects}));

      await project.list()(dispatch);

      sinon.assert.calledWith(dispatch, {type: project.LIST_START});

      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Projects', 'ListProjects', {});

      const successAction = {
        type: project.LIST_SUCCESS,
        projects,
      };
      sinon.assert.calledWith(dispatch, successAction);
    });

    it('failure', async () => {
      prpcClient.call.throws();

      await project.list()(dispatch);

      const action = {
        type: project.LIST_FAILURE,
        error: sinon.match.any,
      };
      sinon.assert.calledWith(dispatch, action);
    });
  });
});
