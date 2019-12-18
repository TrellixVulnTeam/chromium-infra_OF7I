// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Shared file for specifying common types related
 * to features, used in type annotations across Monorail.
 */

import './typedef-common.js';
import './typedef-issue.js';

/**
 * A Hotlist Object returned by the pRPC API.
 *
 * @typedef {Object} Hotlist
 * @property {UserRef} [ownerRef]
 * @property {string} [name]
 * @property {string} [summary]
 * @property {string} [description]
 * @property {string} [defaultColSpec]
 * @property {boolean} [isPrivate]
 */

/**
 * A HotlistItem Object returned by the pRPC API.
 *
 * @typedef {Object} HotlistItem
 * @property {Issue} [issue]
 * @property {number} [rank]
 * @property {UserRef} [adderRef]
 * @property {number} [addedTimestamp]
 * @property {string} [note]
 */
