// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrHotlistHeader} from './mr-hotlist-header.js';

/** @type {MrHotlistHeader} */
let element;

describe('mr-hotlist-header', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('mr-hotlist-header');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrHotlistHeader);
  });

  it('renders', async () => {
    element.selected = 2;
    await element.updateComplete;

    assert.equal(element.shadowRoot.querySelector('mr-tabs').selected, 2);
  });
});
