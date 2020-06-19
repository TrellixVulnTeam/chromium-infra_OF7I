// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import {store, resetState} from 'reducers/base.js';
import {hotlists} from 'reducers/hotlists.js';
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

  it('shows hotlist fetch error', async () => {
    element._fetchError = new Error('This is an important error');
    await element.updateComplete;
    assert.include(element.shadowRoot.innerHTML, 'important error');
  });

  it('renders placeholders with no data', async () => {
    await element.updateComplete;

    const placeholders = element.shadowRoot.querySelectorAll('.placeholder');
    assert.equal(placeholders.length, 2);
  });

  it('renders placeholders with editors list but no user data', async () => {
    element._editors = [null, null];
    await element.updateComplete;

    const placeholders = element.shadowRoot.querySelectorAll('.placeholder');
    assert.equal(placeholders.length, 3);
  });

  it('renders "No editors"', async () => {
    element._editors = [];
    await element.updateComplete;

    assert.include(element.shadowRoot.innerHTML, 'No editors');
  });

  it('renders hotlist', async () => {
    element._hotlist = example.HOTLIST;
    element._owner = exampleUsers.USER;
    element._editors = [exampleUsers.USER_2];
    await element.updateComplete;
  });

  it('shows controls iff user has admin permissions', async () => {
    element._editors = [exampleUsers.USER_2];
    await element.updateComplete;

    assert.equal(element.shadowRoot.querySelectorAll('button').length, 0);

    element._permissions = [hotlists.ADMINISTER];
    await element.updateComplete;

    assert.equal(element.shadowRoot.querySelectorAll('button').length, 2);
  });

  it('shows remove button if user is editing themselves', async () => {
    element._editors = [exampleUsers.USER, exampleUsers.USER_2];
    element._currentUserName = exampleUsers.USER_2.name;
    await element.updateComplete;

    assert.equal(element.shadowRoot.querySelectorAll('button').length, 1);
  });
});

describe('mr-hotlist-people-page (connected)', () => {
  beforeEach(() => {
    store.dispatch(resetState());

    // @ts-ignore
    element = document.createElement('mr-hotlist-people-page');
    document.body.appendChild(element);

    // Stop Redux from overriding values being tested.
    sinon.stub(element, 'stateChanged');
  });

  afterEach(() => {
    element.stateChanged.restore();
    document.body.removeChild(element);
  });

  it('initializes', async () => {
    assert.instanceOf(element, MrHotlistPeoplePage);
  });

  it('updates page title and header', async () => {
    element._hotlist = {...example.HOTLIST, displayName: 'Hotlist-Name'};
    await element.updateComplete;

    const state = store.getState();
    assert.deepEqual(sitewide.pageTitle(state), 'People - Hotlist-Name');
    assert.deepEqual(sitewide.headerTitle(state), 'Hotlist Hotlist-Name');
  });

  it('adds editors', async () => {
    element._hotlist = example.HOTLIST;
    element._permissions = [hotlists.ADMINISTER];
    await element.updateComplete;

    const input = /** @type {HTMLInputElement} */
        (element.shadowRoot.getElementById('add'));
    input.value = 'test@example.com, test2@example.com';

    const update = sinon.spy(hotlists, 'update');
    try {
      await element._onAddEditors(new Event('submit'));

      const editors = ['users/test@example.com', 'users/test2@example.com'];
      sinon.assert.calledWith(update, example.HOTLIST.name, {editors});
    } finally {
      update.restore();
    }
  });

  it('_onAddEditors ignores empty input', async () => {
    element._permissions = [hotlists.ADMINISTER];
    await element.updateComplete;

    const input = /** @type {HTMLInputElement} */
        (element.shadowRoot.getElementById('add'));
    input.value = '  ';

    const update = sinon.spy(hotlists, 'update');
    try {
      await element._onAddEditors(new Event('submit'));
      sinon.assert.notCalled(update);
    } finally {
      update.restore();
    }
  });

  it('removes editors', async () => {
    element._hotlist = example.HOTLIST;

    const removeEditors = sinon.spy(hotlists, 'removeEditors');
    try {
      await element._removeEditor(exampleUsers.NAME_2);

      sinon.assert.calledWith(
          removeEditors, example.HOTLIST.name, [exampleUsers.NAME_2]);
    } finally {
      removeEditors.restore();
    }
  });
});

it('mr-hotlist-people-page (stateChanged)', () => {
  // @ts-ignore
  element = document.createElement('mr-hotlist-people-page');
  document.body.appendChild(element);
  assert.instanceOf(element, MrHotlistPeoplePage);
  document.body.removeChild(element);
});
