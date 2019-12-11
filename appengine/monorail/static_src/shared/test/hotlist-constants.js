// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import 'shared/typedef.js';

// TODO(dtu): monorail:6886: Create user-constants.js.
/** @type UserRef */
const userRefExample = {
  userId: 12345678,
  displayName: 'example@example.com',
};

// TODO(dtu): monorail:6886: Create issue-constants.js.
/** @type Issue */
const issueExample = {
  projectName: 'project-name',
  localId: 1234,
};

/** @type HotlistRef */
export const hotlistRefExample = {
  owner: userRefExample,
  name: 'Hotlist-Name',
};

/** @type Hotlist */
export const hotlistExample = {
  ownerRef: userRefExample,
  name: 'Hotlist-Name',
  summary: 'Summary',
  description: 'Description',
  defaultColSpec: 'Rank ID Summary',
  isPrivate: false,
};

/** @type HotlistItem */
export const hotlistItemExample = {
  issue: issueExample,
  rank: 1,
  adderRef: userRefExample,
  addedTimestamp: 1575000000,
  note: 'Note',
};

/** @type HotlistItem */
export const hotlistItemDifferentProjectExample = {
  issue: {
    projectName: 'other-project-name',
    localId: 1234,
  },
  rank: 1,
  adderRef: userRefExample,
  addedTimestamp: 1575000000,
  note: 'Note',
};

/** @type string */
export const hotlistRefStringExample = '12345678:Hotlist-Name';

/** @type Object.<string, Hotlist> */
export const hotlistsExample = {[hotlistRefStringExample]: hotlistExample};

/** @type Object.<string, Array<HotlistItem>> */
export const hotlistItemsExample = {
  [hotlistRefStringExample]: [hotlistItemExample],
};
