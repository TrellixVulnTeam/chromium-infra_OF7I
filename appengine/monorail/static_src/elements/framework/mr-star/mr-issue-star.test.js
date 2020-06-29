// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrIssueStar} from './mr-issue-star.js';
import {issueRefToString} from 'shared/convertersV0.js';
import sinon from 'sinon';


let element;

describe('mr-issue-star', () => {
  beforeEach(() => {
    element = document.createElement('mr-issue-star');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrIssueStar);
  });

  it('starring logins user when user is not logged in', async () => {
    element._currentUserName = undefined;
    sinon.stub(element, 'login');

    await element.updateComplete;

    const star = element.shadowRoot.querySelector('button');
    assert.isFalse(star.disabled);

    star.click();

    sinon.assert.calledOnce(element.login);
  });

  it('_isStarring true only when issue ref is being starred', async () => {
    element._starringIssues = new Map([['chromium:22', {requesting: true}]]);
    element.issueRef = {projectName: 'chromium', localId: 5};

    assert.isFalse(element._isStarring);

    element.issueRef = {projectName: 'chromium', localId: 22};

    assert.isTrue(element._isStarring);

    element._starringIssues = new Map([['chromium:22', {requesting: false}]]);

    assert.isFalse(element._isStarring);
  });

  it('starring is disabled when _isStarring true', async () => {
    element._currentUserName = 'users/1234';
    sinon.stub(element, '_isStarring').get(() => true);

    await element.updateComplete;

    const star = element.shadowRoot.querySelector('button');
    assert.isTrue(star.disabled);
  });

  it('starring is disabled when _fetchingIsStarred true', async () => {
    element._currentUserName = 'users/1234';
    element._fetchingIsStarred = true;

    await element.updateComplete;

    const star = element.shadowRoot.querySelector('button');
    assert.isTrue(star.disabled);
  });

  it('_starredIssues changes displayed icon', async () => {
    element.issueRef = {projectName: 'proj', localId: 1};

    element._starredIssues = new Set([issueRefToString(element.issueRef)]);

    await element.updateComplete;

    const star = element.shadowRoot.querySelector('button');
    assert.equal(star.textContent.trim(), 'star');

    element._starredIssues = new Set();

    await element.updateComplete;

    assert.equal(star.textContent.trim(), 'star_border');
  });
});
