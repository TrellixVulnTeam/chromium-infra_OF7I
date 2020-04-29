// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';

import {store, resetState} from 'reducers/base.js';
import * as hotlists from 'reducers/hotlists.js';
import * as sitewide from 'reducers/sitewide.js';

import * as example from 'shared/test/constants-hotlists.js';
import * as exampleUsers from 'shared/test/constants-users.js';

import {MrHotlistPeoplePage} from './mr-hotlist-people-page.js';

/** @type {MrHotlistPeoplePage} */
let element;

describe('mr-hotlist-people-page (unconnected)', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('mr-hotlist-people-page-base');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('shows loading message with null hotlist', async () => {
    await element.updateComplete;
    assert.include(element.shadowRoot.innerHTML, 'Loading');
  });

  it('renders with no user data', async () => {
    element._hotlist = example.HOTLIST;
    await element.updateComplete;

    assert.isNotNull(element.shadowRoot.querySelector('.placeholder'));
  });

  it('renders hotlist', async () => {
    element._hotlist = example.HOTLIST;
    element._owner = exampleUsers.USER;
    element._editors = [exampleUsers.USER_2];
    await element.updateComplete;
  });
});

describe('mr-hotlist-people-page (connected)', () => {
  beforeEach(() => {
    store.dispatch(resetState());
    // @ts-ignore
    element = document.createElement('mr-hotlist-people-page');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', async () => {
    assert.instanceOf(element, MrHotlistPeoplePage);
  });

  it('updates page title and header', async () => {
    const hotlistWithName = {...example.HOTLIST, displayName: 'Hotlist-Name'};
    store.dispatch(hotlists.select(example.NAME));
    store.dispatch({type: hotlists.FETCH_SUCCESS, hotlist: hotlistWithName});
    await element.updateComplete;

    const state = store.getState();
    assert.deepEqual(sitewide.pageTitle(state), 'People - Hotlist-Name');
    assert.deepEqual(sitewide.headerTitle(state), 'Hotlist Hotlist-Name');
  });
});
