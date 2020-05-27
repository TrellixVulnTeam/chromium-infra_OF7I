// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrMoveIssueDialog} from './mr-move-issue-hotlists-dialog.js';
import {prpcClient} from 'prpc-client-instance.js';
import * as example from 'shared/test/constants-hotlists.js';

let element;
let waitForPromises;

describe('mr-move-issue-hotlists-dialog', () => {
  beforeEach(async () => {
    element = document.createElement('mr-move-issue-hotlists-dialog');
    document.body.appendChild(element);

    // We need to wait for promisees to resolve. Alone, the updateComplete
    // returns without allowing our Promise.all to resolve.
    waitForPromises = async () => element.updateComplete;

    element.userHotlists = [
      {name: 'Hotlist-1', ownerRef: {userId: 67890}},
      {name: 'Hotlist-2', ownerRef: {userId: 67890}},
      {name: 'Hotlist-3', ownerRef: {userId: 67890}},
      {name: example.HOTLIST.displayName, ownerRef: {userId: 67890}},
    ];
    element.user = {userId: 67890};
    element.issueRefs = [{localId: 22, projectName: 'test'}];
    element._viewedHotlist = example.HOTLIST;
    await element.updateComplete;

    sinon.stub(prpcClient, 'call');
  });

  afterEach(() => {
    document.body.removeChild(element);

    prpcClient.call.restore();
  });

  it('initializes', () => {
    assert.instanceOf(element, MrMoveIssueDialog);
  });

  it('clicking a hotlist moves the issue', async () => {
    element.open();
    await element.updateComplete;

    const targetHotlist =element.shadowRoot.querySelector(
        '.hotlist[data-hotlist-name="Hotlist-2"]');
    assert.isNotNull(targetHotlist);
    targetHotlist.click();
    await element.updateComplete;

    sinon.assert.calledWith(prpcClient.call, 'monorail.Features',
        'AddIssuesToHotlists', {
          hotlistRefs: [{name: 'Hotlist-2', owner: {userId: 67890}}],
          issueRefs: [{localId: 22, projectName: 'test'}],
        });

    sinon.assert.calledWith(prpcClient.call, 'monorail.Features',
        'RemoveIssuesFromHotlists', {
          hotlistRefs: [{
            name: example.HOTLIST.displayName,
            owner: {userId: 67890},
          }],
          issueRefs: [{localId: 22, projectName: 'test'}],
        });
  });

  it('dispatches event upon successfully moving', async () => {
    element.open();
    const savedStub = sinon.stub();
    element.addEventListener('saveSuccess', savedStub);
    sinon.stub(element, 'close');
    await element.updateComplete;

    const targetHotlist =element.shadowRoot.querySelector(
        '.hotlist[data-hotlist-name="Hotlist-2"]');
    targetHotlist.click();

    await waitForPromises();
    sinon.assert.calledOnce(savedStub);
    sinon.assert.calledOnce(element.close);
  });

  it('dispatches no event upon error saving', async () => {
    const mistakes = 'Mistakes were made';
    const error = new Error(mistakes);
    prpcClient.call.returns(Promise.reject(error));
    const savedStub = sinon.stub();
    element.addEventListener('saveSuccess', savedStub);
    element.open();
    await element.updateComplete;

    const targetHotlist =element.shadowRoot.querySelector(
        '.hotlist[data-hotlist-name="Hotlist-2"]');
    targetHotlist.click();

    await waitForPromises();
    sinon.assert.notCalled(savedStub);
    assert.include(element.shadowRoot.innerHTML, mistakes);
  });
});
