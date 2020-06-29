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

  it('unimplemented methods throw errors', () => {
    assert.throws(element.star, 'Method not implemented.');
    assert.throws(element.unstar, 'Method not implemented.');
  });

  describe('clicking star toggles star state', () => {
    beforeEach(() => {
      sinon.stub(element, 'star');
      sinon.stub(element, 'unstar');
      element._isLoggedIn = true;
      element._canStar = true;
    });

    it('unstarred star', async () => {
      element._isStarred = false;

      await element.updateComplete;

      sinon.assert.notCalled(element.star);
      sinon.assert.notCalled(element.unstar);

      element.shadowRoot.querySelector('button').click();

      sinon.assert.calledOnce(element.star);
      sinon.assert.notCalled(element.unstar);
    });

    it('starred star', async () => {
      element._isStarred = true;

      await element.updateComplete;

      sinon.assert.notCalled(element.star);
      sinon.assert.notCalled(element.unstar);

      element.shadowRoot.querySelector('button').click();

      sinon.assert.notCalled(element.star);
      sinon.assert.calledOnce(element.unstar);
    });
  });

  it('clicking while logged out logs you in', async () => {
    sinon.stub(element, 'login');
    element._isLoggedIn = false;
    element._canStar = true;

    await element.updateComplete;

    sinon.assert.notCalled(element.login);

    element.shadowRoot.querySelector('button').click();

    sinon.assert.calledOnce(element.login);
  });

  describe('toggleStar', () => {
    beforeEach(() => {
      sinon.stub(element, 'star');
      sinon.stub(element, 'unstar');
    });

    it('stars when unstarred', () => {
      element._isLoggedIn = true;
      element._canStar = true;
      element._isStarred = false;

      element.toggleStar();

      sinon.assert.calledOnce(element.star);
      sinon.assert.notCalled(element.unstar);
    });

    it('unstars when starred', () => {
      element._isLoggedIn = true;
      element._canStar = true;
      element._isStarred = true;

      element.toggleStar();

      sinon.assert.calledOnce(element.unstar);
      sinon.assert.notCalled(element.star);
    });

    it('does nothing when user is not logged in', () => {
      element._isLoggedIn = false;
      element._canStar = true;
      element._isStarred = true;

      element.toggleStar();

      sinon.assert.notCalled(element.unstar);
      sinon.assert.notCalled(element.star);
    });

    it('does nothing when user does not have permission', () => {
      element._isLoggedIn = true;
      element._canStar = false;
      element._isStarred = true;

      element.toggleStar();

      sinon.assert.notCalled(element.unstar);
      sinon.assert.notCalled(element.star);
    });

    it('does nothing when stars are being fetched', () => {
      element._isLoggedIn = true;
      element._canStar = true;
      element._isStarred = true;
      element._requesting = true;

      element.toggleStar();

      sinon.assert.notCalled(element.unstar);
      sinon.assert.notCalled(element.star);
    });
  });

  describe('disabling star button', () => {
    it('enabled when user is logged in and has permission', async () => {
      element._isLoggedIn = true;
      element._canStar = true;
      element._isStarred = true;
      element._requesting = false;

      await element.updateComplete;

      const star = element.shadowRoot.querySelector('button');
      assert.isFalse(star.disabled);
    });

    it('enabled when user is logged out', async () => {
      element._isLoggedIn = false;
      element._canStar = false;
      element._isStarred = false;
      element._requesting = false;

      await element.updateComplete;

      const star = element.shadowRoot.querySelector('button');
      assert.isFalse(star.disabled);
      assert.isFalse(element._starringEnabled);
    });

    it('disabled when user has no permission', async () => {
      element._isLoggedIn = true;
      element._canStar = false;
      element._isStarred = true;
      element._requesting = false;

      await element.updateComplete;

      const star = element.shadowRoot.querySelector('button');
      assert.isTrue(star.disabled);
    });

    it('disabled when requesting star', async () => {
      element._isLoggedIn = true;
      element._canStar = true;
      element._isStarred = true;
      element._requesting = true;

      await element.updateComplete;

      const star = element.shadowRoot.querySelector('button');
      assert.isTrue(star.disabled);
    });
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

      sinon.stub(element, 'star');
      sinon.stub(element, 'unstar');
    });

    afterEach(() => {
      window.location.hash = oldHash;
    });

    it('clicking to star does not cause navigation', async () => {
      sinon.spy(element, 'toggleStar');
      element._isLoggedIn = true;
      element._canStar = true;
      await element.updateComplete;

      element.shadowRoot.querySelector('button').click();

      assert.notEqual(window.location.hash, '#test-hash');
      sinon.assert.calledOnce(element.toggleStar);
    });

    it('clicking on disabled star does not cause navigation', async () => {
      element._isLoggedIn = true;
      element._canStar = false;
      await element.updateComplete;

      element.shadowRoot.querySelector('button').click();

      assert.notEqual(window.location.hash, '#test-hash');
    });

    it('clicking on link still navigates', async () => {
      element._isLoggedIn = true;
      element._canStar = true;
      await element.updateComplete;

      parent.click();

      assert.equal(window.location.hash, '#test-hash');
    });
  });

  describe('_starToolTip', () => {
    it('not logged in', () => {
      element._isLoggedIn = false;
      element._canStar = false;
      assert.equal(element._starToolTip,
          `Login to star this resource.`);
    });

    it('no permission to star', () => {
      element._isLoggedIn = true;
      element._canStar = false;
      assert.equal(element._starToolTip,
          `You don't have permission to star this resource.`);
    });

    it('star is loading', () => {
      element._isLoggedIn = true;
      element._canStar = true;
      element._requesting = true;
      assert.equal(element._starToolTip,
          `Loading star state for this resource.`);
    });

    it('issue is not starred', () => {
      element._isLoggedIn = true;
      element._canStar = true;
      element._isStarred = false;
      assert.equal(element._starToolTip,
          `Star this resource.`);
    });

    it('issue is starred', () => {
      element._isLoggedIn = true;
      element._canStar = true;
      element._isStarred = true;
      assert.equal(element._starToolTip,
          `Unstar this resource.`);
    });
  });
});
