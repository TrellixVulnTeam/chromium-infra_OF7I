// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrTabs} from './mr-tabs.js';

/** @type {MrTabs} */
let element;

describe('mr-tabs', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('mr-tabs');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrTabs);
  });

  it('renders tabs', async () => {
    element.items = [
      {text: 'Text 1'},
      {text: 'Text 2', icon: 'done', url: 'https://url'},
    ];
    element.selected = 1;
    await element.updateComplete;

    const items = element.shadowRoot.querySelectorAll('li');
    assert.equal(items[0].className, '');
    assert.equal(items[1].className, 'selected');
  });
});
