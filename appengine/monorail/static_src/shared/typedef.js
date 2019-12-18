// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Shared file for specifying common types used in type
 * annotations across Monorail.
 */

import './typedef-common.js';
import './typedef-features.js';
import './typedef-issue.js';
import './typedef-project.js';
import './typedef-user.js';

// TODO(zhangtiff): Find out if there's a way we can generate typedef's for
// API object from .proto files.

/**
 * A String containing the data necessary to identify an IssueRef. An IssueRef
 * can reference either an issue in Monorail or an external issue in another
 * tracker.
 *
 * Examples of valid IssueRefStrings:
 * - monorail:1234
 * - chromium:1
 * - 1234
 * - b/123456
 *
 * @typedef {string} IssueRefString
 */

/**
 * An Object containing the metadata associated with tracking async requests
 * through Redux.
 *
 * @typedef {Object} ReduxRequestState
 * @property {boolean} [requesting] Whether a request is in flight.
 * @property {Error} [error] An Error Object returned by the request.
 */
