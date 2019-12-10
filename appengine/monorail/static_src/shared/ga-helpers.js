// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

const TITLE = 'title';
const LOCATION = 'location';
const DIMENSION1 = 'dimension1';
const SET = 'set';

/**
 * Track page-to-page navigation via google analytics. Global window.ga
 * is set in server rendered HTML.
 *
 * @param {String} page
 * @param {String} userDisplayName
 */
export const trackPageChange = (page = '', userDisplayName = '') => {
  ga(SET, TITLE, `Issue ${page}`);
  if (page.startsWith('user')) {
    ga(SET, TITLE, 'A user page');
    ga(SET, LOCATION, 'A user page URL');
  }

  if (userDisplayName) {
    ga(SET, DIMENSION1, 'Logged in');
  } else {
    ga(SET, DIMENSION1, 'Not logged in');
  }

  ga('send', 'pageview');
};
