// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as issue from './constants-issue.js';
import * as user from './constants-user.js';
import 'shared/typedef.js';

/** @type {string} */
export const NAME = 'hotlists/1234';

/** @type {HotlistV3} */
export const HOTLIST = Object.freeze({
  name: NAME,
  displayName: 'Hotlist-Name',
  owner: user.USER,
  editors: [user.USER_2],
  summary: 'Summary',
  description: 'Description',
  defaultColumns: [{column: 'Rank'}, {column: 'ID'}, {column: 'Summary'}],
  hotlistPrivacy: 1,
});

export const HOTLIST_ITEM_NAME = NAME + '/items/56';

/** @type {HotlistItemV3} */
export const HOTLIST_ITEM = Object.freeze({
  name: HOTLIST_ITEM_NAME,
  issue: issue.NAME,
  // rank: The API excludes the rank field if it's 0.
  adder: user.USER,
  createTime: '2020-01-01T12:00:00Z',
});

/** @type {HotlistItemV3} */
export const HOTLIST_ITEM_OTHER_PROJECT = Object.freeze({
  name: NAME + '/items/78',
  issue: issue.NAME_OTHER_PROJECT,
  rank: 1,
  adder: user.USER,
  createTime: '2020-01-01T12:00:00Z',
});

/** @type {Object.<string, HotlistV3>} */
export const HOTLISTS = Object.freeze({[NAME]: HOTLIST});

/** @type {Object.<string, Array<HotlistItemV3>>} */
export const HOTLIST_ITEMS = Object.freeze({
  [NAME]: [HOTLIST_ITEM],
});
