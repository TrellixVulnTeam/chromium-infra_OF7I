// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import {store, resetState} from 'reducers/base.js';
import {hotlists} from 'reducers/hotlists.js';
import * as sitewide from 'reducers/sitewide.js';

import * as example from 'shared/test/constants-hotlists.js';
import * as exampleUsers from 'shared/test/constants-users.js';

import {MrHotlistSettingsPage} from './mr-hotlist-settings-page.js';

/** @type {MrHotlistSettingsPage} */
let element;

describe('mr-hotlist-settings-page (unconnected)', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('mr-hotlist-settings-page-base');
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

  it('shows loading message with null hotlist', async () => {
    await element.updateComplete;
    assert.include(element.shadowRoot.innerHTML, 'Loading');
  });

  it('renders hotlist', async () => {
    element._hotlist = example.HOTLIST;
    await element.updateComplete;
  });

  it('renders a view only hotlist if no permissions', async () => {
    element._hotlist = {...example.HOTLIST};
    await element.updateComplete;
    assert.notInclude(element.shadowRoot.innerHTML, 'form');
  });

  it('renders an editable hotlist if permission to administer', async () => {
    element._hotlist = {...example.HOTLIST};
    element._permissions = [hotlists.ADMINISTER];
    await element.updateComplete;
    assert.include(element.shadowRoot.innerHTML, 'form');
  });

  it('renders private hotlist', async () => {
    element._hotlist = {...example.HOTLIST, hotlistPrivacy: 'PRIVATE'};
    await element.updateComplete;
    assert.include(element.shadowRoot.innerHTML, 'Members only');
  });
});

describe('mr-hotlist-settings-page (connected)', () => {
  beforeEach(() => {
    store.dispatch(resetState());

    // @ts-ignore
    element = document.createElement('mr-hotlist-settings-page');
    document.body.appendChild(element);

    // Stop Redux from overriding values being tested.
    sinon.stub(element, 'stateChanged');
  });

  afterEach(() => {
    element.stateChanged.restore();
    document.body.removeChild(element);
  });

  it('updates page title and header', async () => {
    element._hotlist = {...example.HOTLIST, displayName: 'Hotlist-Name'};
    await element.updateComplete;

    const state = store.getState();
    assert.deepEqual(sitewide.pageTitle(state), 'Settings - Hotlist-Name');
    assert.deepEqual(sitewide.headerTitle(state), 'Hotlist Hotlist-Name');
  });

  it('deletes hotlist', async () => {
    element._hotlist = example.HOTLIST;
    element._permissions = [hotlists.ADMINISTER];
    element._currentUser = exampleUsers.USER;
    await element.updateComplete;

    const deleteButton = element.shadowRoot.getElementById('delete-hotlist');
    assert.isNotNull(deleteButton);

    // Auto confirm deletion of hotlist.
    const confirmStub = sinon.stub(window, 'confirm');
    confirmStub.returns(true);

    const pageStub = sinon.stub(element, 'page');

    const deleteHotlist = sinon.spy(hotlists, 'deleteHotlist');

    try {
      await element._delete();

      sinon.assert.calledWith(deleteHotlist, example.NAME);
      sinon.assert.calledWith(
          element.page, `/u/${exampleUsers.DISPLAY_NAME}/hotlists`);
    } finally {
      deleteHotlist.restore();
      pageStub.restore();
      confirmStub.restore();
    }
  });

  it('updates hotlist when there are changes', async () => {
    element._hotlist = {...example.HOTLIST};
    element._permissions = [hotlists.ADMINISTER];
    await element.updateComplete;

    const saveButton = element.shadowRoot.getElementById('save-hotlist');
    assert.isNotNull(saveButton);
    assert.isTrue(saveButton.hasAttribute('disabled'));

    const hlist = {
      displayName: element._hotlist.displayName + 'foo',
      summary: element._hotlist.summary + 'abc',
    };

    const summaryInput = element.shadowRoot.getElementById('summary');
    /** @type {HTMLInputElement} */ (summaryInput).value += 'abc';
    const nameInput =
        element.shadowRoot.getElementById('displayName');
    /** @type {HTMLInputElement} */ (nameInput).value += 'foo';

    await element.shadowRoot.getElementById('settingsForm').dispatchEvent(
        new Event('change'));
    assert.isFalse(saveButton.hasAttribute('disabled'));

    const snackbarStub = sinon.stub(element, '_showHotlistSavedSnackbar');
    const update = sinon.stub(hotlists, 'update').returns(async () => {});
    try {
      await element._save();
      sinon.assert.calledWith(update, example.HOTLIST.name, hlist);
      sinon.assert.calledOnce(snackbarStub);
    } finally {
      update.restore();
      snackbarStub.restore();
    }
  });
});

it('mr-hotlist-settings-page (stateChanged)', () => {
  // @ts-ignore
  element = document.createElement('mr-hotlist-settings-page');
  document.body.appendChild(element);
  assert.instanceOf(element, MrHotlistSettingsPage);
  document.body.removeChild(element);
});
