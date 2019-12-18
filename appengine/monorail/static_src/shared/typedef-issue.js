// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Shared file for specifying common types related
 * to issues, used in type annotations across Monorail.
 */

import './typedef-common.js';

/**
 * An Approval Object returned by the pRPC API issue_objects.proto.
 *
 * @typedef {Object} Approval
 * @property {FieldRef} fieldRef
 * @property {Array<UserRef>} approverRefs
 * @property {ApprovalStatus} status
 * @property {number} setOn
 * @property {UserRef} setterRef
 * @property {PhaseRef} phaseRef
 */

/**
 * An Enum representing the status of an Approval.
 *
 * @typedef {string} ApprovalStatus
 */

/**
 * An Amendment Object returned by the pRPC API issue_objects.proto.
 *
 * @typedef {Object} Amendment
 * @property {string} fieldName
 * @property {string} newOrDeltaValue
 * @property {string} oldValue
 */

/**
 * An Attachment Object returned by the pRPC API issue_objects.proto.
 *
 * @typedef {Object} Attachment
* @property {number} attachmentId
* @property {string} filename
* @property {number} size
* @property {string} contentType
* @property {boolean} isDeleted
* @property {string} thumbnailUrl
* @property {string} viewUrl
* @property {string} downloadUrl
*/

/**
 * A Comment Object returned by the pRPC API issue_objects.proto.
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
 * A FieldValue Object returned by the pRPC API issue_objects.proto.
 *
 * @typedef {Object} FieldValue
 * @property {FieldRef} fieldRef
 * @property {string} value
 * @property {boolean} [isDerived]
 * @property {PhaseRef} [phaseRef]
 */

/**
 * An Issue Object returned by the pRPC API issue_objects.proto.
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
 * An PhaseDef Object returned by the pRPC API issue_objects.proto.
 *
 * @typedef {Object} PhaseDef
 * @property {PhaseRef} phaseRef
 * @property {number} rank
 */

/**
 * An PhaseRef Object returned by the pRPC API issue_objects.proto.
 *
 * @typedef {Object} PhaseRef
 * @property {string} phaseName
 */
