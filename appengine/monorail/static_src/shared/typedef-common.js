// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Shared file for specifying common types used in type
 * annotations across Monorail.
 */

/**
 * A ComponentRef Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} ComponentRef
 * @property {string} path
 * @property {boolean=} isDerived
 */

/**
 * An Enum representing the type that a custom field uses.
 *
 * @typedef {string} FieldType
 */

/**
 * A FieldRef Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} FieldRef
 * @property {number} fieldId
 * @property {string} fieldName
 * @property {FieldType} type
 * @property {string} approvalName
 */

/**
 * A LabelRef Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} LabelRef
 * @property {string} label
 * @property {boolean=} isDerived
 */

/**
 * A StatusRef Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} StatusRef
 * @property {string} status
 * @property {boolean=} meansOpen
 * @property {boolean=} isDerived
 */

/**
 * An IssueRef Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} IssueRef
 * @property {string=} projectName
 * @property {number=} localId
 * @property {string=} extIdentifier
 */

/**
 * A UserRef Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} UserRef
 * @property {string=} displayName
 * @property {number=} userId
 */

/**
 * A HotlistRef Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} HotlistRef
 * @property {string=} name
 * @property {UserRef=} owner
 */

/**
 * A SavedQuery Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} SavedQuery
 * @property {number} queryId
 * @property {string} name
 * @property {string} query
 * @property {Array<string>} projectNames
 */
