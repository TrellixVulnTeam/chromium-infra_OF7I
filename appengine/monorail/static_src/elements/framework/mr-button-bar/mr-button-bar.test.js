// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';

import {MrButtonBar} from './mr-button-bar.js';

/** @type {MrButtonBar} */
let element;

describe('mr-button-bar', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('mr-button-bar');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrButtonBar);
  });

  it('renders button items', async () => {
    const handler = sinon.stub();

    element.items = [{icon: 'emoji_nature', text: 'Pollinate', handler}];
    await element.updateComplete;

    const button = element.shadowRoot.querySelector('button');
    button.click();

    assert.include(button.innerHTML, 'emoji_nature');
    assert.include(button.innerHTML, 'Pollinate');
    sinon.assert.calledOnce(handler);
  });

  it('renders dropdown items', async () => {
    const items = [{icon: 'emoji_nature', text: 'Pollinate'}];
    element.items = [{icon: 'more_vert', text: 'More actions...', items}];
    await element.updateComplete;

    /** @type {MrDropdown} */
    const dropdown = element.shadowRoot.querySelector('mr-dropdown');
    assert.strictEqual(dropdown.icon, 'more_vert');
    assert.strictEqual(dropdown.label, 'More actions...');
    assert.strictEqual(dropdown.items, items);
  });
});
