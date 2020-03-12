// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import {prpcClient} from 'prpc-client-instance.js';
import {store, resetState} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import * as project from 'reducers/project.js';
import * as sitewide from 'reducers/sitewide.js';

import * as example from 'shared/test/constants-hotlist.js';
import * as exampleUser from 'shared/test/constants-user.js';

import {MrHotlistIssuesPage} from './mr-hotlist-issues-page.js';

/** @type {MrHotlistIssuesPage} */
let element;

describe('mr-hotlist-issues-page (unconnected)', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('mr-hotlist-issues-page-base');
    element._extractFieldValuesFromIssue =
      project.extractFieldValuesFromIssue({});
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('shows loading message with null hotlist', async () => {
    await element.updateComplete;
    assert.include(element.shadowRoot.innerHTML, 'Loading');
  });

  it('renders hotlist items with one project', async () => {
    element._hotlist = example.HOTLIST;
    element._items = [example.HOTLIST_ISSUE];
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.deepEqual(issueList.projectName, 'project-name');
  });

  it('renders hotlist items with multiple projects', async () => {
    element._hotlist = example.HOTLIST;
    element._items = [
      example.HOTLIST_ISSUE,
      example.HOTLIST_ISSUE_OTHER_PROJECT,
    ];
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.isNull(issueList.projectName);
  });

  it('computes strings for HotlistIssue fields', async () => {
    const clock = sinon.useFakeTimers(24 * 60 * 60 * 1000);

    try {
      element._hotlist = example.HOTLIST;
      element._items = [{
        ...example.HOTLIST_ISSUE,
        summary: 'Summary',
        rank: 52,
        adder: exampleUser.USER,
        createTime: new Date(0).toISOString(),
      }];
      element._columns = ['Summary', 'Rank', 'Added', 'Adder'];
      await element.updateComplete;

      const issueList = element.shadowRoot.querySelector('mr-issue-list');
      assert.include(issueList.shadowRoot.innerHTML, 'Summary');
      assert.include(issueList.shadowRoot.innerHTML, '53');
      assert.include(issueList.shadowRoot.innerHTML, 'a day ago');
      assert.include(issueList.shadowRoot.innerHTML, exampleUser.DISPLAY_NAME);
    } finally {
      clock.restore();
    }
  });

  it('filters and shows closed issues', async () => {
    element._hotlist = example.HOTLIST;
    element._items = [example.HOTLIST_ISSUE_CLOSED];
    await element.updateComplete;

    const issueList = element.shadowRoot.querySelector('mr-issue-list');
    assert.equal(issueList.issues.length, 0);

    element.shadowRoot.querySelector('chops-filter-chips').select('Closed');
    await element.updateComplete;

    assert.isTrue(element._filter.Closed);
    assert.equal(issueList.issues.length, 1);
  });
});

describe('mr-hotlist-issues-page (connected)', () => {
  beforeEach(() => {
    store.dispatch(resetState());
    // @ts-ignore
    element = document.createElement('mr-hotlist-issues-page');
    element._extractFieldValuesFromIssue =
      project.extractFieldValuesFromIssue({});
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrHotlistIssuesPage);
  });

  it('query string overrides hotlist default columns', () => {
    const defaultColumns = [{column: 'Rank'}, {column: 'Summary'}];
    const hotlistWithColumns = {...example.HOTLIST, defaultColumns};
    store.dispatch(hotlist.select(example.NAME));
    store.dispatch({type: hotlist.FETCH_SUCCESS, hotlist: hotlistWithColumns});

    assert.deepEqual(element._columns, ['Rank', 'Summary']);

    const queryParams = {colspec: 'Rank ID Summary'};
    store.dispatch(sitewide.setQueryParams(queryParams));

    assert.deepEqual(element._columns, ['Rank', 'ID', 'Summary']);
  });

  it('updates page title and header', async () => {
    const hotlistWithName = {...example.HOTLIST, displayName: 'Hotlist-Name'};
    store.dispatch(hotlist.select(example.NAME));
    store.dispatch({type: hotlist.FETCH_SUCCESS, hotlist: hotlistWithName});
    await element.updateComplete;

    const state = store.getState();
    assert.deepEqual(sitewide.pageTitle(state), 'Issues - Hotlist-Name');
    assert.deepEqual(sitewide.headerTitle(state), 'Hotlist Hotlist-Name');
  });

  it('reranks', () => {
    sinon.stub(prpcClient, 'call');

    try {
      element._hotlist = example.HOTLIST;
      element._items = [
        example.HOTLIST_ISSUE,
        example.HOTLIST_ISSUE_CLOSED,
        example.HOTLIST_ISSUE_OTHER_PROJECT,
      ];

      element._rerank([example.HOTLIST_ITEM_NAME], 1);

      // We can't stub hotlist.rerankItems(), so stub prpcClient.call() instead.
      // https://github.com/sinonjs/sinon/issues/562
      const args = {
        name: example.NAME,
        hotlistItems: [example.HOTLIST_ITEM_NAME],
        targetPosition: 2,
      };
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Hotlists', 'RerankHotlistItems', args);
    } finally {
      prpcClient.call.restore();
    }
  });
});
