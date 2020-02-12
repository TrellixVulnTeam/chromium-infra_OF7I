// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrHotlistPeoplePage} from './mr-hotlist-people-page.js';

/** @type {MrHotlistPeoplePage} */
let element;

describe('mr-hotlist-people-page', () => {
  beforeEach(() => {
    // @ts-ignore
    element = document.createElement('mr-hotlist-people-page');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', async () => {
    assert.instanceOf(element, MrHotlistPeoplePage);
  });
});
