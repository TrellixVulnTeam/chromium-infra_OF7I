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
import * as exampleUser from 'shared/test/constants-user.js';

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

  it('renders a view only hotlist', async () => {
    element._hotlist = {...example.HOTLIST};
    await element.updateComplete;
    assert.notInclude(element.shadowRoot.innerHTML, 'form');
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

  describe('hotlist owner logged in', () => {
    let stateChangedStub;
    beforeEach(async () => {
      // Stop Redux from overriding values being tested.
      stateChangedStub = sinon.stub(element, 'stateChanged');

      element._currentUser = exampleUser.USER_REF;
      element._hotlist = {...example.HOTLIST};
      await element.updateComplete;
    });

    afterEach(() => {
      stateChangedStub.restore();
    });

    it('renders a hotlist with an editable form', () => {
      assert.include(element.shadowRoot.innerHTML, 'form');
    });

    describe('it makes a reducer call', () => {
      let callStub;
      beforeEach(() => {
        // We can't stub reducers/hotlist methods so stub prpcClient.call()
        // instead. https://github.com/sinonjs/sinon/issues/562
        callStub = sinon.stub(prpcClient, 'call');
      });

      afterEach(() => {
        callStub.restore();
      });

      it('deletes hotlist', async () => {
        const deleteButton = element.shadowRoot.getElementById(
            'delete-hotlist');
        assert.isNotNull(deleteButton);

        // Auto confirm deletion of hotlist.
        const confirmStub = sinon.stub(window, 'confirm');
        confirmStub.returns(true);

        const pageStub = sinon.stub(element, 'page');

        try {
          const args = {name: example.NAME};

          await element._delete();

          sinon.assert.calledWith(
              prpcClient.call, 'monorail.v1.Hotlists', 'DeleteHotlist', args);
          sinon.assert.calledWith(
              element.page, `/u/${example.HOTLIST.owner.displayName}/hotlists`);
        } finally {
          pageStub.restore();
          confirmStub.restore();
        }
      });

      it('updates hotlist when there are changes', async () => {
        sinon.stub(element, '_showHotlistSavedSnackbar');
        const saveButton = element.shadowRoot.getElementById('save-hotlist');
        assert.isNotNull(saveButton);
        assert.isTrue(saveButton.hasAttribute('disabled'));

        const hotlist = {
          name: example.HOTLIST.name,
          displayName: element._hotlist.displayName + 'foo',
          summary: element._hotlist.summary + 'abc',
        };
        const args = {hotlist, updateMask: 'displayName,summary'};

        const summaryInput = element.shadowRoot.getElementById('summary');
        /** @type {HTMLInputElement} */ (summaryInput).value += 'abc';
        const nameInput =
            element.shadowRoot.getElementById('displayName');
        /** @type {HTMLInputElement} */ (nameInput).value += 'foo';

        await element.shadowRoot.getElementById('settingsForm').dispatchEvent(
            new Event('change'));
        assert.isFalse(saveButton.hasAttribute('disabled'));

        await element._save();

        sinon.assert.calledWith(
            prpcClient.call, 'monorail.v1.Hotlists', 'UpdateHotlist', args);
        sinon.assert.calledOnce(element._showHotlistSavedSnackbar);
      });
    });
  });
});
