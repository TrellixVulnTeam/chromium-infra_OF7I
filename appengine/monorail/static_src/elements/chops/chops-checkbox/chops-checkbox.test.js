// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {ChopsCheckbox} from './chops-checkbox.js';

let element;

describe('chops-checkbox', () => {
  beforeEach(() => {
    element = document.createElement('chops-checkbox');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, ChopsCheckbox);
  });

  it('clicking checkbox dispatches checked-change event', async () => {
    element.checked = false;
    sinon.stub(window, 'CustomEvent');
    sinon.stub(element, 'dispatchEvent');

    await element.updateComplete;

    element.shadowRoot.querySelector('#checkbox').click();

    assert.deepEqual(window.CustomEvent.args[0][0], 'checked-change');
    assert.deepEqual(window.CustomEvent.args[0][1], {
      detail: {checked: true},
    });

    assert.isTrue(window.CustomEvent.calledOnce);
    assert.isTrue(element.dispatchEvent.calledOnce);

    window.CustomEvent.restore();
    element.dispatchEvent.restore();
  });

  it('updating checked property updates native <input>', async () => {
    element.checked = false;

    await element.updateComplete;

    assert.isFalse(element.checked);
    assert.isFalse(element.shadowRoot.querySelector('input').checked);

    element.checked = true;

    await element.updateComplete;

    assert.isTrue(element.checked);
    assert.isTrue(element.shadowRoot.querySelector('input').checked);
  });

  it('updating checked attribute updates native <input>', async () => {
    element.setAttribute('checked', true);
    await element.updateComplete;

    assert.equal(element.getAttribute('checked'), 'true');
    assert.isTrue(element.shadowRoot.querySelector('input').checked);

    element.click();
    await element.updateComplete;

    // We expect the 'checked' attribute to remain the same even as the
    // corresponding property changes when the user clicks the checkbox.
    assert.equal(element.getAttribute('checked'), 'true');
    assert.isFalse(element.shadowRoot.querySelector('input').checked);

    element.click();
    await element.updateComplete;
    assert.isTrue(element.shadowRoot.querySelector('input').checked);

    element.removeAttribute('checked');
    await element.updateComplete;
    assert.isNotTrue(element.getAttribute('checked'));
    assert.isFalse(element.shadowRoot.querySelector('input').checked);
  });
});
