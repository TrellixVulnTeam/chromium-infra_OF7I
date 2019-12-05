// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Shared file for specifying common types used in type
 * annotations across Monorail.
 */

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

/**
 * A Hotlist Object returned by the pRPC API.
 *
 * @typedef {Object} Hotlist
 * @property {UserRef} [ownerRef]
 * @property {string} [name]
 * @property {string} [summary]
 * @property {string} [description]
 */

/**
 * A HotlistRef Object returned by the pRPC API.
 *
 * @typedef {Object} HotlistRef
 * @property {string} [name]
 * @property {UserRef} [owner]
 */

/**
 * A User Object returned by the pRPC API.
 *
 * @typedef {Object} User
 * @property {string} [displayName]
 * @property {number} [userId]
 * @property {boolean} [isSiteAdmin]
 * @property {string} [availability]
 * @property {UserRef} [linkedParentRef]
 * @property {Array<UserRef>} [linkedChildRefs]
 */

/**
 * A UserRef Object returned by the pRPC API.
 *
 * @typedef {Object} UserRef
 * @property {string} [displayName]
 * @property {number} [userId]
 */

/**
 * A LabelRef Object returned by the pRPC API.
 *
 * @typedef {Object} LabelRef
 * @property {string} label
 * @property {boolean} [isDerived]
 */

/**
 * A StatusRef Object returned by the pRPC API.
 *
 * @typedef {Object} StatusRef
 * @property {string} status
 * @property {boolean} [meansOpen]
 * @property {boolean} [isDerived]
 */

/**
 * A ComponentRef Object returned by the pRPC API.
 *
 * @typedef {Object} ComponentRef
 * @property {string} path
 * @property {boolean} [isDerived]
 */

/**
 * An Issue Object returned by the pRPC API.
 *
 * @typedef {Object} Issue
 * @property {string} projectName
 * @property {number} localId
 * @property {string} [summary]
 * @property {StatusRef} [statusRef]
 * @property {UserRef} [ownerRef]
 * @property {Array<UserRef>} [ccRefs]
 * @property {Array<LabelRef>} [labelRefs]
 * @property {Array<ComponentRef>} [componentRefs]
 * @property {Array<IssueRef>} [blockedOnIssueRefs]
 * @property {Array<IssueRef>} [blockingIssueRefs]
 * @property {Array<IssueRef>} [danglingBlockedOnRefs]
 * @property {Array<IssueRef>} [danglingBlockingRefs]
 * @property {IssueRef} [mergedIntoIssueRef]
 * @property {FieldValue} [fieldValues]
 * @property {boolean} [isDeleted]
 * @property {UserRef} [reporterRef]
 * @property {number} [openedTimestamp]
 * @property {number} [closedTimestamp]
 * @property {number} [modifiedTimestamp]
 * @property {number} [componentModifiedTimestamp]
 * @property {number} [statusModifiedTimestamp]
 * @property {number} [ownerModifiedTimestamp]
 * @property {number} [starCount]
 * @property {boolean} [isSpam]
 * @property {number} [attachmentCount]
 * @property {Array<Approval>} [approvalValues]
 * @property {Array<PhaseDef>} [phases]
 */

/**
 * An IssueRef Object returned by the pRPC API.
 *
 * @typedef {Object} IssueRef
 * @property {string} [projectName]
 * @property {number} [localId]
 * @property {string} [extIdentifier]
 */

/**
 * A Comment Object returned by the pRPC API.
 *
 * Note: This Object is called "Comment" in the backend but is named
 * "IssueComment" here to avoid a collision with an internal JSDoc Intellisense
 * type.
 *
 * @typedef {Object} IssueComment
 * @property {string} projectName
 * @property {number} localId
 * @property {number} [sequenceNum]
 * @property {boolean} [isDeleted]
 * @property {UserRef} [commenter]
 * @property {number} [timestamp]
 * @property {string} [content]
 * @property {string} [inboundMessage]
 * @property {Array<Amendment>} [amendments]
 * @property {Array<Attachment>} [attachments]
 * @property {FieldRef} [approvalRef]
 * @property {number} [descriptionNum]
 * @property {boolean} [isSpam]
 * @property {boolean} [canDelete]
 * @property {boolean} [canFlag]
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
 * A FieldValue Object returned by the pRPC API common.proto.
 *
 * @typedef {Object} FieldValue
 * @property {FieldRef} fieldRef
 * @property {string} value
 * @property {boolean} [isDerived]
 * @property {PhaseRef} [phaseRef]
 */

/**
 * A StatusDef Object returned as part of the pRPC API project_objects.proto.
 *
 * @typedef {Object} StatusDef
 * @property {boolean} deprecated
 * @property {string} docstring
 * @property {boolean} meansOpen
 * @property {string} status
 */

// TODO(zhangtiff): Define properties for the typedefs below.
/**
 * @typedef {Object} Approval
 */

/**
 * @typedef {Object} PhaseRef
 */

/**
 * @typedef {Object} PhaseDef
 */

/**
 * @typedef {Object} Amendment
 */

/**
 * @typedef {Object} Attachment
 */

/**
 * @typedef {Object} FieldDef
 */
