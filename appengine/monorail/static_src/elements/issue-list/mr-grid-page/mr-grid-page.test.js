// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrGridPage} from './mr-grid-page.js';

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

  it('calls to fetchIssueList made when _currentQuery changes', async () => {
    await element.updateComplete;
    const issueListCall = sinon.stub(element, '_fetchMatchingIssues');
    element._queryParams = {x: 'Blocked'};
    await element.updateComplete;
    sinon.assert.notCalled(issueListCall);

    element._presentationConfigLoaded = true;
    element._currentQuery = 'cc:me';
    await element.updateComplete;
    sinon.assert.calledOnce(issueListCall);
  });

  it('calls to fetchIssueList made when _currentCan changes', async () => {
    await element.updateComplete;
    const issueListCall = sinon.stub(element, '_fetchMatchingIssues');
    element._queryParams = {y: 'Blocked'};
    await element.updateComplete;
    sinon.assert.notCalled(issueListCall);

    element._presentationConfigLoaded = true;
    element._currentCan = 1;
    await element.updateComplete;
    sinon.assert.calledOnce(issueListCall);
  });

  describe('_shouldFetchMatchingIssues', () => {
    it('default returns false', () => {
      const result = element._shouldFetchMatchingIssues(new Map());
      assert.isFalse(result);
    });

    it('returns true for projectName', () => {
      element._queryParams = {q: ''};
      const changedProps = new Map();
      changedProps.set('projectName', 'anything');
      const result = element._shouldFetchMatchingIssues(changedProps);
      assert.isTrue(result);
    });

    it('returns true when _currentQuery changes', () => {
      element._presentationConfigLoaded = true;

      element._currentQuery = 'owner:me';
      const changedProps = new Map();
      changedProps.set('_currentQuery', '');

      const result = element._shouldFetchMatchingIssues(changedProps);
      assert.isTrue(result);
    });

    it('returns true when _currentCan changes', () => {
      element._presentationConfigLoaded = true;

      element._currentCan = 1;
      const changedProps = new Map();
      changedProps.set('_currentCan', 2);

      const result = element._shouldFetchMatchingIssues(changedProps);
      assert.isTrue(result);
    });

    it('returns false when presentation config not loaded', () => {
      element._presentationConfigLoaded = false;

      const changedProps = new Map();
      changedProps.set('projectName', 'anything');
      const result = element._shouldFetchMatchingIssues(changedProps);

      assert.isFalse(result);
    });

    it('returns true when presentationConfig fetch completes', () => {
      element._presentationConfigLoaded = true;

      const changedProps = new Map();
      changedProps.set('_presentationConfigLoaded', false);
      const result = element._shouldFetchMatchingIssues(changedProps);

      assert.isTrue(result);
    });
  });
});
