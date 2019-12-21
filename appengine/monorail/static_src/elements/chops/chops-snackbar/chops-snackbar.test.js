// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {ChopsSnackbar} from './chops-snackbar.js';

let element;

describe('chops-snackbar', () => {
  beforeEach(() => {
    element = document.createElement('chops-snackbar');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, ChopsSnackbar);
  });

  it('dispatches close event on close click', async () => {
    element.opened = true;
    await element.updateComplete;

    const listener = sinon.stub();
    element.addEventListener('close', listener);

    element.shadowRoot.querySelector('button').click();

    sinon.assert.calledOnce(listener);
  });
});
