// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import 'shared/typedef.js';

export const NAME = 'users/1234';

export const DISPLAY_NAME = 'example@example.com';

export const ID = 1234;

/** @type {UserRef} */
export const USER_REF = Object.freeze({
  userId: ID,
  displayName: DISPLAY_NAME,
});

/** @type {User} */
export const USER = Object.freeze({
  name: NAME,
  displayName: DISPLAY_NAME,
});

export const NAME_2 = 'users/5678';

export const DISPLAY_NAME_2 = 'other_user@example.com';

/** @type {User} */
export const USER_2 = Object.freeze({
  name: NAME_2,
  displayName: DISPLAY_NAME_2,
});

/** @type {Object<string, User>} */
export const BY_NAME = Object.freeze({[NAME]: USER, [NAME_2]: USER_2});

/** @type {ProjectMember} */
export const PROJECT_MEMBER = Object.freeze({
  name: 'projects/proj/members/1234',
  role: 'CONTRIBUTOR',
});

