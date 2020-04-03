// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import {prpcClient} from 'prpc-client-instance.js';
import {store, resetState} from 'reducers/base.js';
import * as hotlist from 'reducers/hotlist.js';
import * as sitewide from 'reducers/sitewide.js';

import * as example from 'shared/test/constants-hotlist.js';

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

  it('shows loading message with null hotlist', async () => {
    await element.updateComplete;
    assert.include(element.shadowRoot.innerHTML, 'Loading');
  });

  it('renders hotlist', async () => {
    element._hotlist = example.HOTLIST;
    await element.updateComplete;
  });

  it('renders private hotlist', async () => {
    element._hotlist = {...example.HOTLIST, hotlistPrivacy: 0};
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
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', async () => {
    assert.instanceOf(element, MrHotlistSettingsPage);
  });

  it('updates page title and header', async () => {
    const hotlistWithName = {...example.HOTLIST, displayName: 'Hotlist-Name'};
    store.dispatch(hotlist.select(example.NAME));
    store.dispatch({type: hotlist.FETCH_SUCCESS, hotlist: hotlistWithName});
    await element.updateComplete;

    const state = store.getState();
    assert.deepEqual(sitewide.pageTitle(state), 'Settings - Hotlist-Name');
    assert.deepEqual(sitewide.headerTitle(state), 'Hotlist Hotlist-Name');
  });

  it('deletes hotlist', async () => {
    const stateChangedStub = sinon.stub(element, 'stateChanged');
    element._currentUser = example.HOTLIST.owner;
    element._hotlist = example.HOTLIST;
    await element.updateComplete;

    const deleteButton = element.shadowRoot.getElementById('delete-hotlist');
    assert.isNotNull(deleteButton);

    // Auto confirm deletion of hotlist.
    const confirmStub = sinon.stub(window, 'confirm');
    confirmStub.returns(true);
    const callStub = sinon.stub(prpcClient, 'call');
    const pageStub = sinon.stub(element, 'page');
    // Stop Redux from overriding values being tested.

    try {
      const args = {name: example.NAME};

      await element._delete();

      // We can't stub hotlist.deleteHotlist(), so stub prpcClient.call()
      // instead. https://github.com/sinonjs/sinon/issues/562
      sinon.assert.calledWith(
          prpcClient.call, 'monorail.v1.Hotlists', 'DeleteHotlist', args);
      sinon.assert.calledWith(
          element.page, `/u/${example.HOTLIST.owner.displayName}/hotlists`);
    } finally {
      pageStub.restore();
      callStub.restore();
      confirmStub.restore();
      stateChangedStub.restore();
    }
  });
});
