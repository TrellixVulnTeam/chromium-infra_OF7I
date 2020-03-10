// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrKeystrokes} from './mr-keystrokes.js';
import Mousetrap from 'mousetrap';

import {issueRefToString} from 'shared/converters.js';

/** @type {MrKeystrokes} */
let element;

describe('mr-keystrokes', () => {
  beforeEach(() => {
    element = /** @type {MrKeystrokes} */ (
      document.createElement('mr-keystrokes'));
    document.body.appendChild(element);

    element._projectName = 'proj';
    element.issueId = 11;
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrKeystrokes);
  });

  it('tracks if the issue is currently starring', async () => {
    await element.updateComplete;
    assert.isFalse(element._isStarring);

    const issueRefStr = issueRefToString(element._issueRef);
    element._starringIssues.set(issueRefStr, {requesting: true});
    assert.isTrue(element._isStarring);
  });

  it('? and esc open and close dialog', async () => {
    await element.updateComplete;
    assert.isFalse(element._opened);

    Mousetrap.trigger('?');

    await element.updateComplete;
    assert.isTrue(element._opened);

    Mousetrap.trigger('esc');

    await element.updateComplete;
    assert.isFalse(element._opened);
  });

  describe('issue detail keys', () => {
    beforeEach(() => {
      sinon.stub(element, '_page');
      sinon.stub(element, '_jumpToEditForm');
      sinon.stub(element, '_starIssue');
    });

    it('not bound when _projectName not set', async () => {
      element._projectName = '';
      element.issueId = 1;

      await element.updateComplete;

      // Navigation hot keys.
      Mousetrap.trigger('k');
      Mousetrap.trigger('j');
      Mousetrap.trigger('u');
      sinon.assert.notCalled(element._page);

      // Jump to edit form hot key.
      Mousetrap.trigger('r');
      sinon.assert.notCalled(element._jumpToEditForm);

      // Star issue hotkey.
      Mousetrap.trigger('s');
      sinon.assert.notCalled(element._starIssue);
    });

    it('not bound when issueId not set', async () => {
      element._projectName = 'proj';
      element.issueId = 0;

      await element.updateComplete;

      // Navigation hot keys.
      Mousetrap.trigger('k');
      Mousetrap.trigger('j');
      Mousetrap.trigger('u');
      sinon.assert.notCalled(element._page);

      // Jump to edit form hot key.
      Mousetrap.trigger('r');
      sinon.assert.notCalled(element._jumpToEditForm);

      // Star issue hotkey.
      Mousetrap.trigger('s');
      sinon.assert.notCalled(element._starIssue);
    });

    it('binds j and k navigation hot keys', async () => {
      element.queryParams = {q: 'something'};

      await element.updateComplete;

      Mousetrap.trigger('k');
      sinon.assert.calledWith(element._page,
          '/p/proj/issues/detail/previous?q=something');

      Mousetrap.trigger('j');
      sinon.assert.calledWith(element._page,
          '/p/proj/issues/detail/next?q=something');

      Mousetrap.trigger('u');
      sinon.assert.calledWith(element._page,
          '/p/proj/issues/list?q=something&cursor=proj%3A11');
    });

    it('u key navigates back to issue list wth cursor set', async () => {
      element.queryParams = {q: 'something'};

      await element.updateComplete;

      Mousetrap.trigger('u');
      sinon.assert.calledWith(element._page,
          '/p/proj/issues/list?q=something&cursor=proj%3A11');
    });

    it('u key navigates back to hotlist when hotlist_id set', async () => {
      element.queryParams = {hotlist_id: 1234};

      await element.updateComplete;

      Mousetrap.trigger('u');
      sinon.assert.calledWith(element._page,
          '/p/proj/issues/detail/list?hotlist_id=1234&cursor=proj%3A11');
    });

    it('does not star when user does not have permission', async () => {
      element.queryParams = {q: 'something'};
      element._issuePermissions = [];

      await element.updateComplete;

      Mousetrap.trigger('s');
      sinon.assert.notCalled(element._starIssue);
    });

    it('does star when user has permission', async () => {
      element.queryParams = {q: 'something'};
      element._issuePermissions = ['setstar'];

      await element.updateComplete;

      Mousetrap.trigger('s');
      sinon.assert.calledOnce(element._starIssue);
    });

    it('does not star when user does not have permission', async () => {
      element.queryParams = {q: 'something'};
      element._issuePermissions = [];

      await element.updateComplete;

      Mousetrap.trigger('s');
      sinon.assert.notCalled(element._starIssue);
    });

    it('does not jump to edit form when user cannot comment', async () => {
      element.queryParams = {q: 'something'};
      element._issuePermissions = [];

      await element.updateComplete;

      Mousetrap.trigger('r');
      sinon.assert.notCalled(element._jumpToEditForm);
    });

    it('does jump to edit form when user can comment', async () => {
      element.queryParams = {q: 'something'};
      element._issuePermissions = ['addissuecomment'];

      await element.updateComplete;

      Mousetrap.trigger('r');
      sinon.assert.calledOnce(element._jumpToEditForm);
    });
  });
});
