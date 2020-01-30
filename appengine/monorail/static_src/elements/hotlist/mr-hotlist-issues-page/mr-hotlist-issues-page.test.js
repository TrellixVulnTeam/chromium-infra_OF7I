// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import * as project from 'reducers/project.js';
import * as example from 'shared/test/constants-hotlist.js';
import {MrHotlistIssuesPage, prepareIssues} from './mr-hotlist-issues-page.js';

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
    element.hotlist = example.HOTLIST;
    element.hotlistItems = [example.HOTLIST_ITEM];
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.notInclude(issueList.shadowRoot.innerHTML, 'other-project-name');
  });

  it('renders hotlist items with multiple projects', async () => {
    element.hotlist = example.HOTLIST;
    element.hotlistItems = [
      example.HOTLIST_ITEM,
      example.HOTLIST_ITEM_OTHER_PROJECT,
    ];
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.include(issueList.shadowRoot.innerHTML, 'other-project-name');
  });

  it('sorts items by Rank', async () => {
    const hotlistIssues = prepareIssues([
      example.HOTLIST_ITEM_OTHER_PROJECT,
      example.HOTLIST_ITEM,
    ]);

    assert.lengthOf(hotlistIssues, 2);
    assert.equal(hotlistIssues[0].rank, 1);
    assert.equal(hotlistIssues[1].rank, 2);
  });

  it('computes strings for HotlistIssue fields', async () => {
    element.hotlist = {
      ...example.HOTLIST,
      defaultColSpec: 'Summary Rank Added Adder Note',
    };
    element.hotlistItems = [{
      issue: {projectName: 'project-name', localId: 1234, summary: 'Summary'},
      rank: 53,
      adderRef: {displayName: 'example@example.com'},
      addedTimestamp: Date.now() / 1000 - 24 * 60 * 60, // a day ago
      note: 'Note',
    }];
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.include(issueList.shadowRoot.innerHTML, 'Summary');
    assert.include(issueList.shadowRoot.innerHTML, '53');
    assert.include(issueList.shadowRoot.innerHTML, 'a day ago');
    assert.include(issueList.shadowRoot.innerHTML, 'example@example.com');
    assert.include(issueList.shadowRoot.innerHTML, 'Note');
  });
});
