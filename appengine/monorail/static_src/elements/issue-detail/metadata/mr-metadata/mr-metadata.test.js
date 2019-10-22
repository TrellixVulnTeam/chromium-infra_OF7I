// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrMetadata} from './mr-metadata.js';

let element;

describe('mr-metadata', () => {
  beforeEach(() => {
    element = document.createElement('mr-metadata');
    document.body.appendChild(element);

    element.projectName = 'proj';
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrMetadata);
  });

  it('has table role set', () => {
    assert.equal(element.getAttribute('role'), 'table');
  });

  it('always renders owner', async () => {
    const ownerLabel = 'Owner:';
    await element.updateComplete;

    const trElements = Array.prototype.slice
        .call(element.shadowRoot.children)
        .filter((ele) => ele.tagName === 'TR');
    const rendersOwner = trElements.reduce((acc, ele) => {
      return (
        (ele.children[0].tagName === 'TH' &&
          ele.children[0].textContent === ownerLabel) ||
        acc
      );
    }, false);

    assert.isTrue(rendersOwner);
  });

  it('always renders cc', async () => {
    const ccLabel = 'Owner:';
    await element.updateComplete;

    const trElements = Array.prototype.slice
        .call(element.shadowRoot.children)
        .filter((ele) => ele.tagName === 'TR');
    const rendersOwner = trElements.reduce((acc, ele) => {
      return (
        (ele.children[0].tagName === 'TH' &&
          ele.children[0].textContent === ccLabel) ||
        acc
      );
    }, false);

    assert.isTrue(rendersOwner);
  });
});
