// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
import sinon from 'sinon';
import {assert} from 'chai';

import {MrStar} from './mr-star.js';

let element;

describe('mr-star', () => {
  beforeEach(() => {
    element = document.createElement('mr-star');
    document.body.appendChild(element);
  });

  afterEach(() => {
    if (document.body.contains(element)) {
      document.body.removeChild(element);
    }
  });

  it('initializes', () => {
    assert.instanceOf(element, MrStar);
  });

  it('clicking star toggles star', async () => {
    sinon.spy(element, 'toggleStar');
    element._canStar = true;

    await element.updateComplete;

    sinon.assert.notCalled(element.toggleStar);

    element.shadowRoot.querySelector('button').click();

    sinon.assert.calledOnce(element.toggleStar);
  });

  it('toggleStar stars when unstarred', () => {
    sinon.spy(element, 'star');
    sinon.spy(element, 'unstar');

    element._canStar = true;
    element._isStarred = false;

    element.toggleStar();

    sinon.assert.calledOnce(element.star);
    sinon.assert.notCalled(element.unstar);
  });

  it('toggleStar unstars when starred', () => {
    sinon.spy(element, 'star');
    sinon.spy(element, 'unstar');

    element._canStar = true;
    element._isStarred = true;

    element.toggleStar();

    sinon.assert.calledOnce(element.unstar);
    sinon.assert.notCalled(element.star);
  });

  it('starring is disabled when canStar is false', async () => {
    element._canStar = false;

    await element.updateComplete;

    const star = element.shadowRoot.querySelector('button');
    assert.isTrue(star.disabled);
  });

  it('isStarred changes displayed icon', async () => {
    element._isStarred = true;
    await element.updateComplete;

    const star = element.shadowRoot.querySelector('button');
    assert.equal(star.textContent.trim(), 'star');

    element._isStarred = false;
    await element.updateComplete;

    assert.equal(star.textContent.trim(), 'star_border');
  });

  describe('mr-star nested inside a link', () => {
    let parent;
    let oldHash;

    beforeEach(() => {
      parent = document.createElement('a');
      parent.setAttribute('href', '#test-hash');
      parent.appendChild(element);

      oldHash = window.location.hash;
    });

    afterEach(() => {
      window.location.hash = oldHash;
    });

    it('clicking to star does not cause navigation', async () => {
      sinon.spy(element, 'toggleStar');
      element._canStar = true;
      await element.updateComplete;

      element.shadowRoot.querySelector('button').click();

      assert.notEqual(window.location.hash, '#test-hash');
      sinon.assert.calledOnce(element.toggleStar);
    });

    it('clicking on disabled star does not cause navigation', async () => {
      element._canStar = false;
      await element.updateComplete;

      element.shadowRoot.querySelector('button').click();

      assert.notEqual(window.location.hash, '#test-hash');
    });

    it('clicking on link still navigates', async () => {
      element._canStar = true;
      await element.updateComplete;

      parent.click();

      assert.equal(window.location.hash, '#test-hash');
    });
  });

  describe('_starToolTip', () => {
    it('no permission to star', () => {
      element._canStar = false;
      assert.equal(element._starToolTip,
          `You don't have permission to star this resource.`);
    });

    it('issue is not starred', () => {
      element._canStar = true;
      element._isStarred = false;
      assert.equal(element._starToolTip,
          `Star this resource.`);
    });

    it('issue is starred', () => {
      element._canStar = true;
      element._isStarred = true;
      assert.equal(element._starToolTip,
          `Unstar this resource.`);
    });
  });
});
