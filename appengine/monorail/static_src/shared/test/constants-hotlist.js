// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {ISSUE, ISSUE_OTHER_PROJECT} from './constants-issue.js';
import {USER_REF} from './constants-user.js';
import 'shared/typedef.js';

/** @type {HotlistRef} */
export const HOTLIST_REF = Object.freeze({
  owner: USER_REF,
  name: 'Hotlist-Name',
});

/** @type {Hotlist} */
export const HOTLIST = Object.freeze({
  ownerRef: USER_REF,
  name: 'Hotlist-Name',
  summary: 'Summary',
  description: 'Description',
  defaultColSpec: 'Rank ID Summary',
  isPrivate: false,
});

/** @type {HotlistItem} */
export const HOTLIST_ITEM = Object.freeze({
  issue: ISSUE,
  rank: 1,
  adderRef: USER_REF,
  addedTimestamp: 1575000000,
  note: 'Note',
});

/** @type {HotlistItem} */
export const HOTLIST_ITEM_OTHER_PROJECT = Object.freeze({
  issue: ISSUE_OTHER_PROJECT,
  rank: 1,
  adderRef: USER_REF,
  addedTimestamp: 1575000000,
  note: 'Note',
});

/** @type {string} */
export const HOTLIST_REF_STRING = '12345678:Hotlist-Name';

/** @type {Object.<string, Hotlist>} */
export const HOTLISTS = Object.freeze({[HOTLIST_REF_STRING]: HOTLIST});

/** @type {Object.<string, Array<HotlistItem>>} */
export const HOTLIST_ITEMS = Object.freeze({
  [HOTLIST_REF_STRING]: [HOTLIST_ITEM],
});
