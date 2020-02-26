// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {ChopsFilterChips} from './chops-filter-chips.js';

/** @type {ChopsFilterChips} */
let element;

describe('chops-filter-chips', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('chops-filter-chips');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, ChopsFilterChips);
  });

  it('renders', async () => {
    element.options = ['one', 'two'];
    element.selected = {two: true};
    await element.updateComplete;

    const firstChip = element.shadowRoot.firstElementChild;
    assert.deepEqual(firstChip.className, '');
    assert.deepEqual(firstChip.thumbnail, '');

    const lastChip = element.shadowRoot.lastElementChild;
    assert.deepEqual(lastChip.className, 'selected');
    assert.deepEqual(lastChip.thumbnail, 'check');
  });

  it('click', async () => {
    const onChangeStub = sinon.stub();

    element.options = ['one'];
    await element.updateComplete;

    element.addEventListener('change', onChangeStub);
    element.shadowRoot.firstElementChild.click();

    assert.isTrue(element.selected.one);
    sinon.assert.calledOnce(onChangeStub);

    element.shadowRoot.firstElementChild.click();

    assert.isFalse(element.selected.one);
    sinon.assert.calledTwice(onChangeStub);
  });
});
