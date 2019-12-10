// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {trackPageChange} from './ga-helpers.js';

describe.only('trackPageChange', () => {
  beforeEach(() => {
    global.ga = sinon.spy();
  });

  afterEach(() => {
    global.ga.resetHistory();
  });

  it('sets page title', () => {
    trackPageChange('list');
    sinon.assert.calledWith(global.ga, 'set', 'title', 'Issue list');
  });

  it('sets user page title', () => {
    trackPageChange('user-anything');
    sinon.assert.calledWith(global.ga, 'set', 'title', 'A user page');
  });

  it('sets user location', () => {
    trackPageChange('user-anything');
    sinon.assert.calledWith(global.ga, 'set', 'location', 'A user page URL');
  });

  it('defaults dimension1', () => {
    trackPageChange('list');
    sinon.assert.calledWith(global.ga, 'set', 'dimension1', 'Not logged in');
  });

  it('sets dimension1 based on userDisplayName', () => {
    trackPageChange('list', 'somebody');
    sinon.assert.calledWith(global.ga, 'set', 'dimension1', 'Logged in');
  });

  it('sends pageview', () => {
    trackPageChange('list');
    sinon.assert.calledWith(global.ga, 'send', 'pageview');
  });
});
