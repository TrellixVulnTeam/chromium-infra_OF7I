// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrIssueEntryPage} from './mr-issue-entry-page.js';

let element;

describe('mr-issue-entry-page', () => {
  beforeEach(() => {
    element = document.createElement('mr-issue-entry-page');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrIssueEntryPage);
  });

  describe('requires user to be logged in', () => {
    it('redirects to loginUrl if not logged in', async () => {
      document.body.removeChild(element);
      element = document.createElement('mr-issue-entry-page');
      assert.isUndefined(element.userDisplayName);

      const EXPECTED = 'abc';
      element.loginUrl = EXPECTED;

      const pageStub = sinon.stub(element, '_page');
      document.body.appendChild(element);
      await element.updateComplete;

      sinon.assert.calledOnce(pageStub);
      sinon.assert.calledWith(pageStub, EXPECTED);
    });

    it('renders when user is logged in', async () => {
      document.body.removeChild(element);
      element = document.createElement('mr-issue-entry-page');

      element.loginUrl = 'abc';
      element.userDisplayName = 'not_undefined';

      const pageStub = sinon.stub(element, '_page');
      const renderSpy = sinon.spy(element, 'render');
      document.body.appendChild(element);
      await element.updateComplete;

      sinon.assert.notCalled(pageStub);
      sinon.assert.calledOnce(renderSpy);
    });
  });
});
