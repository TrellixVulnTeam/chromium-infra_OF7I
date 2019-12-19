// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import * as example from 'shared/test/constants-hotlist.js';
import {MrHotlistIssuesPage} from './mr-hotlist-issues-page.js';

let element;

describe('mr-hotlist-issues-page', () => {
  beforeEach(() => {
    element = document.createElement('mr-hotlist-issues-page');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', async () => {
    assert.instanceOf(element, MrHotlistIssuesPage);
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
});
