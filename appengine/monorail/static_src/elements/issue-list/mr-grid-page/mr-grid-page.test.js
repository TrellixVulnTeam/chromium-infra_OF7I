// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrGridPage, refetchTriggeringProps} from './mr-grid-page.js';

let element;

describe('mr-grid-page', () => {
  beforeEach(() => {
    element = document.createElement('mr-grid-page');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrGridPage);
  });

  it('progress bar updates properly', async () => {
    await element.updateComplete;
    element.progress = .2499;
    await element.updateComplete;
    const title =
      element.shadowRoot.querySelector('progress').getAttribute('title');
    assert.equal(title, '25%');
  });

  it('displays error when no issues match query', async () => {
    await element.updateComplete;
    element.progress = 1;
    element.totalIssues = 0;
    await element.updateComplete;
    const error =
      element.shadowRoot.querySelector('.empty-search').textContent;
    assert.equal(error.trim(), 'Your search did not generate any results.');
  });

  it('calls to fetchIssueList made when q changes', async () => {
    await element.updateComplete;
    const issueListCall = sinon.stub(element, '_fetchMatchingIssues');
    element._queryParams = {x: 'Blocked'};
    await element.updateComplete;
    sinon.assert.notCalled(issueListCall);

    element._queryParams = {q: 'cc:me'};
    await element.updateComplete;
    sinon.assert.calledOnce(issueListCall);
  });

  describe('refetchTriggeringProps', () => {
    it('includes q and can', () => {
      assert.isTrue(refetchTriggeringProps.has('q'));
      assert.isTrue(refetchTriggeringProps.has('can'));
    });
  });

  describe('_shouldFetchMatchingIssues', () => {
    it('default returns false', () => {
      const result = element._shouldFetchMatchingIssues(new Map());
      assert.isFalse(result);
    });

    it('returns true for projectName', () => {
      const changedProps = new Map();
      changedProps.set('projectName', 'anything');
      const result = element._shouldFetchMatchingIssues(changedProps);
      assert.isTrue(result);
    });

    it('depends on _queryParam\'s q and can when _queryParams changes', () => {
      const changedProps = new Map();
      changedProps.set('_queryParams', {can: '1'});
      let result = element._shouldFetchMatchingIssues(changedProps);
      assert.isTrue(result);

      changedProps.set('_queryParams', {q: 'anything'});
      result = element._shouldFetchMatchingIssues(changedProps);
      assert.isTrue(result);
    });
  });
});

