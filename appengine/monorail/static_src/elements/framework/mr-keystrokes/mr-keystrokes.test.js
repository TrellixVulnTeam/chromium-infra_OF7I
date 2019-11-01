// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrKeystrokes} from './mr-keystrokes.js';
import page from 'page';
import Mousetrap from 'mousetrap';

import {issueRefToString} from 'shared/converters.js';

/** @type {MrKeystrokes} */
let element;

describe('mr-keystrokes', () => {
  beforeEach(() => {
    element = /** @type {MrKeystrokes} */ (
      document.createElement('mr-keystrokes'));
    document.body.appendChild(element);

    element.projectName = 'proj';
    element.issueId = 11;

    sinon.stub(page, 'call');
  });

  afterEach(() => {
    document.body.removeChild(element);

    page.call.restore();
  });

  it('initializes', () => {
    assert.instanceOf(element, MrKeystrokes);
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

  it('tracks if the issue is currently starring', async () => {
    await element.updateComplete;
    assert.isFalse(element._isStarring);

    const issueRefStr = issueRefToString(element._issueRef);
    element._starringIssues.set(issueRefStr, {requesting: true});
    assert.isTrue(element._isStarring);
  });

  // TODO(zhangtiff): Figure out how to best test page navigation.
});
