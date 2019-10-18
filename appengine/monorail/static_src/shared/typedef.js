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
 * A JS Object in the format of a User Object returned by the pRPC API.
 *
 * @typedef {Object} User
 * @property {String} [displayName]
 * @property {Number} [userId]
 * @property {Boolean} [isSiteAdmin]
 * @property {String} [availability]
 * @property {UserRef} [linkedParentRef]
 * @property {Array<UserRef>} [linkedChildRefs]
 */

/**
 * A JS Object in the format of a UserRef Object returned by the pRPC API.
 *
 * @typedef {Object} UserRef
 * @property {String} [displayName]
 * @property {Number} [userId]
 */

/**
 * A JS Object in the format of a LabelRef Object returned by the pRPC API.
 *
 * @typedef {Object} LabelRef
 * @property {String} label
 * @property {Boolean} [isDerived]
 */

/**
 * A JS Object in the format of a StatusRef Object returned by the pRPC API.
 *
 * @typedef {Object} StatusRef
 * @property {String} status
 * @property {Boolean} [meansOpen]
 * @property {Boolean} [isDerived]
 */

/**
 * A JS Object in the format of a ComponentRef Object returned by the pRPC API.
 *
 * @typedef {Object} ComponentRef
 * @property {String} path
 * @property {Boolean} [isDerived]
 */

/**
 * A JS Object in the format of an Issue Object returned by the pRPC API.
 *
 * @typedef {Object} Issue
 * @property {String} projectName
 * @property {Number} localId
 * @property {String} [summary]
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
 * @property {Boolean} [isDeleted]
 * @property {UserRef} [reporterRef]
 * @property {Number} [openedTimestamp]
 * @property {Number} [closedTimestamp]
 * @property {Number} [modifiedTimestamp]
 * @property {Number} [componentModifiedTimestamp]
 * @property {Number} [statusModifiedTimestamp]
 * @property {Number} [ownerModifiedTimestamp]
 * @property {Number} [starCount]
 * @property {Boolean} [isSpam]
 * @property {Number} [attachmentCount]
 * @property {Array<Approval>} [approvalValues]
 * @property {Array<PhaseDef>} [phases]
 */

/**
 * A JS Object in the format of an IssueRef Object returned by the pRPC API.
 *
 * @typedef {Object} IssueRef
 * @property {String} [projectName]
 * @property {Number} [localId]
 * @property {String} [extIdentifier]
 */

/**
 * A JS Object in the format of a Comment Object returned by the pRPC API.
 *
 * Note: This Object is called "Comment" in the backend but is named
 * "IssueComment" here to avoid a collision with an internal JSDoc Intellisense
 * type.
 *
 * @typedef {Object} IssueComment
 * @property {String} projectName
 * @property {Number} localId
 * @property {Number} [sequenceNum]
 * @property {Boolean} [isDeleted]
 * @property {UserRef} [commenter]
 * @property {Number} [timestamp]
 * @property {String} [content]
 * @property {String} [inboundMessage]
 * @property {Array<Amendment>} [amendments]
 * @property {Array<Attachment>} [attachments]
 * @property {FieldRef} [approvalRef]
 * @property {Number} [descriptionNum]
 * @property {Boolean} [isSpam]
 * @property {Boolean} [canDelete]
 * @property {Boolean} [canFlag]
 */

/**
 * An Enum representing the type that a custom field uses.
 *
 * @typedef {String} FieldType
 */

/**
 * A JS Object in the format of a FieldRef Object returned by the pRPC API.
 *
 * @typedef {Object} FieldRef
 * @property {Number} fieldId
 * @property {String} fieldName
 * @property {FieldType} type
 * @property {String} approvalName
 */

/**
 * A JS Object in the format of a FieldValue Object returned by the pRPC API.
 *
 * @typedef {Object} FieldValue
 * @property {FieldRef} fieldRef
 * @property {String} value
 * @property {Boolean} [isDerived]
 * @property {PhaseRef} [phaseRef]
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
