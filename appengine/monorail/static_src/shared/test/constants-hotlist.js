// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as issue from './constants-issue.js';
import * as user from './constants-user.js';
import 'shared/typedef.js';

/** @type {string} */
export const NAME = 'hotlists/1234';

/** @type {Hotlist} */
export const HOTLIST = Object.freeze({
  name: NAME,
  displayName: 'Hotlist-Name',
  owner: user.USER,
  editors: [user.USER_2],
  summary: 'Summary',
  description: 'Description',
  defaultColumns: [{column: 'Rank'}, {column: 'ID'}, {column: 'Summary'}],
  hotlistPrivacy: 'PUBLIC',
});

export const HOTLIST_ITEM_NAME = NAME + '/items/56';

/** @type {HotlistItem} */
export const HOTLIST_ITEM = Object.freeze({
  name: HOTLIST_ITEM_NAME,
  issue: issue.NAME,
  // rank: The API excludes the rank field if it's 0.
  adder: user.USER,
  createTime: '2020-01-01T12:00:00Z',
});

/** @type {HotlistIssue} */
export const HOTLIST_ISSUE = Object.freeze({...issue.ISSUE, ...HOTLIST_ITEM});

/** @type {HotlistItem} */
export const HOTLIST_ITEM_OTHER_PROJECT = Object.freeze({
  name: NAME + '/items/78',
  issue: issue.NAME_OTHER_PROJECT,
  rank: 1,
  adder: user.USER,
  createTime: '2020-01-01T12:00:00Z',
});

/** @type {HotlistIssue} */
export const HOTLIST_ISSUE_OTHER_PROJECT = Object.freeze({
  ...issue.ISSUE_OTHER_PROJECT, ...HOTLIST_ITEM_OTHER_PROJECT,
});

/** @type {HotlistItem} */
export const HOTLIST_ITEM_CLOSED = Object.freeze({
  name: NAME + '/items/90',
  issue: issue.NAME_CLOSED,
  rank: 2,
  adder: user.USER,
  createTime: '2020-01-01T12:00:00Z',
});

/** @type {HotlistIssue} */
export const HOTLIST_ISSUE_CLOSED = Object.freeze({
  ...issue.ISSUE_CLOSED, ...HOTLIST_ITEM_CLOSED,
});

/** @type {Object<string, Hotlist>} */
export const BY_NAME = Object.freeze({[NAME]: HOTLIST});

/** @type {Object<string, Array<HotlistItem>>} */
export const HOTLIST_ITEMS = Object.freeze({
  [NAME]: [HOTLIST_ITEM],
});
