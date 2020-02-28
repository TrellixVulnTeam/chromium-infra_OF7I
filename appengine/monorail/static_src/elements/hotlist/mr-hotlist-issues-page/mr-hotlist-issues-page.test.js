// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import * as project from 'reducers/project.js';
import * as example from 'shared/test/constants-hotlist.js';
import * as exampleIssue from 'shared/test/constants-issue.js';

import {MrHotlistIssuesPage} from './mr-hotlist-issues-page.js';

/** @type {MrHotlistIssuesPage} */
let element;

describe('mr-hotlist-issues-page', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('mr-hotlist-issues-page');
    element._extractFieldValuesFromIssue =
      project.extractFieldValuesFromIssue({});
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', async () => {
    assert.instanceOf(element, MrHotlistIssuesPage);
  });

  it('shows loading message with null hotlist', async () => {
    await element.updateComplete;
    assert.include(element.shadowRoot.innerHTML, 'Loading');
  });

  it('renders hotlist items with one project', async () => {
    sinon.stub(element, 'stateChanged');
    element._hotlist = example.HOTLIST;
    element._hotlistItems = [example.HOTLIST_ITEM];
    element._issue = () => exampleIssue.ISSUE;
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.deepEqual(issueList.projectName, 'project-name');
  });

  it('renders hotlist items with multiple projects', async () => {
    sinon.stub(element, 'stateChanged');
    element._hotlist = example.HOTLIST;
    element._hotlistItems = [
      example.HOTLIST_ITEM,
      example.HOTLIST_ITEM_OTHER_PROJECT,
    ];
    element._issue = (name) => ({
      [exampleIssue.NAME]: exampleIssue.ISSUE,
      [exampleIssue.NAME_OTHER_PROJECT]: exampleIssue.ISSUE_OTHER_PROJECT,
    }[name]);
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.isNull(issueList.projectName);
  });

  it('computes strings for HotlistIssue fields', async () => {
    sinon.stub(element, 'stateChanged');
    const clock = sinon.useFakeTimers(24 * 60 * 60 * 1000);

    try {
      element._hotlist = {
        ...example.HOTLIST,
        defaultColumns: [
          {column: 'Summary'}, {column: 'Rank'},
          {column: 'Added'}, {column: 'Adder'},
        ],
      };
      element._hotlistItems = [{
        issue: exampleIssue.NAME,
        rank: 52,
        adder: 'users/5678',
        createTime: new Date(0).toISOString(),
      }];
      element._issue = () => ({...exampleIssue.ISSUE, summary: 'Summary'});
      await element.updateComplete;

      const issueList = element.shadowRoot.querySelector('mr-issue-list');
      assert.include(issueList.shadowRoot.innerHTML, 'Summary');
      assert.include(issueList.shadowRoot.innerHTML, '53');
      assert.include(issueList.shadowRoot.innerHTML, 'a day ago');
      assert.include(issueList.shadowRoot.innerHTML, 'users/5678');
    } finally {
      clock.restore();
    }
  });

  it('filters and shows closed issues', async () => {
    sinon.stub(element, 'stateChanged');
    element._hotlist = example.HOTLIST;
    element._hotlistItems = [example.HOTLIST_ITEM];
    element._issue = () => exampleIssue.ISSUE_CLOSED;
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.equal(issueList.issues.length, 0);

    element.shadowRoot.querySelector('chops-filter-chips').select('Closed');
    await element.updateComplete;

    assert.isTrue(element._filter.Closed);
    assert.equal(issueList.issues.length, 1);
  });
});
