// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import * as issue from './constants-issue.js';
import 'shared/typedef.js';

/** @type {Permission} */
export const PERMISSION_ISSUE_EDIT = 'ISSUE_EDIT';

/** @type {PermissionSet} */
export const PERMISSION_SET_ISSUE = {
  resource: issue.NAME,
  permissions: [PERMISSION_ISSUE_EDIT],
};

/** @type {Object<string, PermissionSet>} */
export const PERMISSION_SETS = {
  [issue.NAME]: PERMISSION_SET_ISSUE,
};
