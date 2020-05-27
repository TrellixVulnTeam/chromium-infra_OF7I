// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrIssueHotlistsDialog} from './mr-issue-hotlists-dialog.js';

let element;
const EXAMPLE_USER_HOTLISTS = [
  {name: 'Hotlist-1'},
  {name: 'Hotlist-2'},
  {name: 'ac-apple-1'},
  {name: 'ac-frita-1'},
];

describe('mr-issue-hotlists-dialog', () => {
  beforeEach(async () => {
    element = document.createElement('mr-issue-hotlists-dialog');
    document.body.appendChild(element);

    await element.updateComplete;
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrIssueHotlistsDialog);
    assert.include(element.shadowRoot.innerHTML, 'Dialog elements below');
  });

  it('filters hotlists', async () => {
    element.userHotlists = EXAMPLE_USER_HOTLISTS;
    element.open();
    await element.updateComplete;

    const initialHotlists = element.shadowRoot.querySelectorAll('.hotlist');
    assert.equal(initialHotlists.length, 4);
    const filterInput = element.shadowRoot.querySelector('#filter');
    filterInput.value = 'list';
    element.filterHotlists();
    await element.updateComplete;
    let visibleHotlists =
        element.shadowRoot.querySelectorAll('.hotlist');
    assert.equal(visibleHotlists.length, 2);

    filterInput.value = '2';
    element.filterHotlists();
    await element.updateComplete;
    visibleHotlists =
        element.shadowRoot.querySelectorAll('.hotlist');
    assert.equal(visibleHotlists.length, 1);
  });

  it('resets filter on open', async () => {
    element.userHotlists = EXAMPLE_USER_HOTLISTS;
    element.open();
    await element.updateComplete;

    const filterInput = element.shadowRoot.querySelector('#filter');
    filterInput.value = 'ac';
    element.filterHotlists();
    await element.updateComplete;
    let visibleHotlists =
        element.shadowRoot.querySelectorAll('.hotlist');
    assert.equal(visibleHotlists.length, 2);

    element.close();
    element.open();
    await element.updateComplete;

    assert.equal(filterInput.value, '');
    visibleHotlists =
        element.shadowRoot.querySelectorAll('.hotlist');
    assert.equal(visibleHotlists.length, 4);
  });
});
