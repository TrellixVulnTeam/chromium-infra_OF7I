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

/** @type {User} */
export const USER_2 = Object.freeze({
  userId: 5678,
  name: 'users/5678',
  displayName: 'other_user@example.com',
});

/** @type {Object<string, User>} */
export const BY_NAME = Object.freeze({[NAME]: USER});
