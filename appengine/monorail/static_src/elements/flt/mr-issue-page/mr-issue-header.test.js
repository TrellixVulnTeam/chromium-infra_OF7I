// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrIssueHeader} from './mr-issue-header.js';
import {store, actionType} from '../../redux/redux-mixin.js';
import {flush} from '@polymer/polymer/lib/utils/flush.js';

let element;
let lockIcon;
let lockTooltip;

suite('mr-issue-header', () => {
  setup(() => {
    element = document.createElement('mr-issue-header');
    document.body.appendChild(element);

    lockIcon = element.shadowRoot.querySelector('.lock-icon');
    lockTooltip = element.shadowRoot.querySelector('.lock-tooltip');
  });

  teardown(() => {
    document.body.removeChild(element);
    element.dispatchAction({type: actionType.RESET_STATE});
  });

  test('initializes', () => {
    assert.instanceOf(element, MrIssueHeader);
  });

  test('updating issue id changes header', function() {
    assert.equal(store.getState().issueId, 0);

    store.dispatch({
      type: actionType.UPDATE_ISSUE_REF,
      issueId: 1,
    });

    assert.equal(store.getState().issueId, 1);

    store.dispatch({
      type: actionType.FETCH_ISSUE_SUCCESS,
      issue: {summary: 'test'},
    });

    assert.deepEqual(store.getState().issue, {summary: 'test'});

    // TODO(zhangtiff): Figure out how to properly test
    // state changes propagating to the element. As is, state
    // changes don't seem to actually make it to the element.
    // assert.deepEqual(element.issue, {summary: 'test'});
  });

  test('shows restricted icon only when restricted', () => {
    assert.isTrue(lockIcon.hasAttribute('hidden'));

    element.isRestricted = true;

    flush();

    assert.isFalse(lockIcon.hasAttribute('hidden'));

    element.isRestricted = false;

    flush();

    assert.isTrue(lockIcon.hasAttribute('hidden'));
  });

  test('displays view restrictions', () => {
    element.isRestricted = true;

    element.restrictions = {
      view: ['Google', 'hello'],
      edit: ['Editor', 'world'],
      comment: ['commentor'],
    };

    const restrictString =
      'Only users with Google and hello permission can see this issue.';
    assert.equal(element._restrictionText, restrictString);

    assert.equal(lockIcon.title, restrictString);
    assert.include(lockTooltip.textContent, restrictString);
  });

  test('displays edit restrictions', () => {
    element.isRestricted = true;

    element.restrictions = {
      view: [],
      edit: ['Editor', 'world'],
      comment: ['commentor'],
    };

    const restrictString =
      'Only users with Editor and world permission may make changes.';
    assert.equal(element._restrictionText, restrictString);

    assert.equal(lockIcon.title, restrictString);
    assert.include(lockTooltip.textContent, restrictString);
  });

  test('displays comment restrictions', () => {
    element.isRestricted = true;

    element.restrictions = {
      view: [],
      edit: [],
      comment: ['commentor'],
    };

    const restrictString =
      'Only users with commentor permission may comment.';
    assert.equal(element._restrictionText, restrictString);

    assert.equal(lockIcon.title, restrictString);
    assert.include(lockTooltip.textContent, restrictString);
  });
});
