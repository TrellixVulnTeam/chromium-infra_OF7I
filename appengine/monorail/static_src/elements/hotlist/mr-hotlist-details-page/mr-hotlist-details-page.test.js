// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {hotlistExample} from 'shared/test/hotlist-constants.js';
import {MrHotlistDetailsPage} from './mr-hotlist-details-page.js';

let element;

describe('mr-hotlist-details-page', () => {
  beforeEach(() => {
    element = document.createElement('mr-hotlist-details-page');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', async () => {
    assert.instanceOf(element, MrHotlistDetailsPage);
  });

  it('shows loading message with null hotlist', async () => {
    await element.updateComplete;
  });

  it('renders hotlist', async () => {
    element.hotlist = hotlistExample;
    await element.updateComplete;
  });
});
