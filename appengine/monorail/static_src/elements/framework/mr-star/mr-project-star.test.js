// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {MrProjectStar} from './mr-project-star.js';
import {stars} from 'reducers/stars.js';

let element;

describe('mr-project-star (disconnected)', () => {
  beforeEach(() => {
    element = document.createElement('mr-project-star');
    document.body.appendChild(element);

    sinon.stub(element, 'stateChanged');
    sinon.spy(stars, 'starProject');
    sinon.spy(stars, 'unstarProject');
  });

  afterEach(() => {
    document.body.removeChild(element);

    stars.starProject.restore();
    stars.unstarProject.restore();
  });

  it('initializes', () => {
    assert.instanceOf(element, MrProjectStar);
  });

  it('clicking on star when logged out logs in user', async () => {
    element._currentUserName = undefined;
    sinon.stub(element, 'login');

    await element.updateComplete;

    const star = element.shadowRoot.querySelector('button');
    assert.isFalse(star.disabled);

    star.click();

    sinon.assert.calledOnce(element.login);
  });

  it('star dispatches star request', () => {
    element._currentUserName = 'users/1234';
    element.name = 'projects/monorail';

    element.star();

    sinon.assert.calledWith(stars.starProject,
        'projects/monorail', 'users/1234');
  });

  it('unstar dispatches unstar request', () => {
    element._currentUserName = 'users/1234';
    element.name = 'projects/monorail';

    element.unstar();

    sinon.assert.calledWith(stars.unstarProject,
        'projects/monorail', 'users/1234');
  });

  describe('isStarred', () => {
    beforeEach(() => {
      element._stars = {
        'users/1234/projectStars/monorail':
            {name: 'users/1234/projectStars/monorail'},
        'users/5678/projectStars/chromium':
            {name: 'users/5678/projectStars/chromium'},
      };
    });

    it('false when no data', () => {
      element._stars = {};
      assert.isFalse(element.isStarred);
    });

    it('false when user is not logged in', () => {
      element._currentUserName = '';
      element.name = 'projects/monorail';

      assert.isFalse(element.isStarred);
    });

    it('false when project is not starred', () => {
      element._currentUserName = 'users/1234';
      element.name = 'projects/chromium';

      assert.isFalse(element.isStarred);

      element._currentUserName = 'users/5678';
      element.name = 'projects/monorail';

      assert.isFalse(element.isStarred);
    });

    it('true when user has starred project', () => {
      element._currentUserName = 'users/1234';
      element.name = 'projects/monorail';

      assert.isTrue(element.isStarred);

      element._currentUserName = 'users/5678';
      element.name = 'projects/chromium';

      assert.isTrue(element.isStarred);
    });
  });

  describe('disabled', () => {
    beforeEach(() => {
      element._currentUserName = 'users/1234';
      element.name = 'projects/monorail';
    });

    it('enabled when user is not logged in', () => {
      element._currentUserName = '';

      assert.isFalse(element.disabled);
    });

    it('disabled when stars are being fetched', () => {
      element._fetchingStars = true;
      element._starringProjects = {};
      element._unstarringProjects = {};

      assert.isTrue(element.disabled);
    });

    it('disabled when user is starring project', () => {
      element._fetchingStars = false;
      element._starringProjects =
          {'users/1234/projectStars/monorail': {requesting: true}};
      element._unstarringProjects = {};

      assert.isTrue(element.disabled);
    });

    it('disabled when user is unstarring project', () => {
      element._fetchingStars = false;
      element._starringProjects = {};
      element._unstarringProjects =
          {'users/1234/projectStars/monorail': {requesting: true}};

      assert.isTrue(element.disabled);
    });

    it('enabled when user is starring an unrelated project', () => {
      element._fetchingStars = false;
      element._starringProjects = {
        'users/1234/projectStars/chromium': {requesting: true},
        'users/1234/projectStars/monorail': {requesting: false},
      };
      element._unstarringProjects = {};

      assert.isFalse(element.disabled);
    });

    it('enabled when user is unstarring an unrelated project', () => {
      element._fetchingStars = false;
      element._starringProjects = {};
      element._unstarringProjects = {
        'users/1234/projectStars/chromium': {requesting: true},
        'users/1234/projectStars/monorail': {requesting: false},
      };

      assert.isFalse(element.disabled);
    });

    it('enabled when no in-flight requests', () => {
      element._fetchingStars = false;
      element._starringProjects = {};
      element._unstarringProjects = {};

      assert.isFalse(element.disabled);
    });
  });
});
