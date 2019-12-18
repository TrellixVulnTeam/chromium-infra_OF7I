// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Shared file for specifying common types related
 * to users, used in type annotations across Monorail.
 */

import './typedef-common.js';

/**
 * A User Object returned by the pRPC API user_objects.proto.
 *
 * @typedef {Object} User
 * @property {string=} displayName
 * @property {number=} userId
 * @property {boolean=} isSiteAdmin
 * @property {string=} availability
 * @property {UserRef=} linkedParentRef
 * @property {Array<UserRef>=} linkedChildRefs
 */
