// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/* eslint-disable max-len */

/**
 * The label of an issue.
 * @typedef {Object} LabelValue
 * @property {string} label - the string label. e.g. 'Target-99'.
 * @property {string} derivation - How the label was derived. One of 'EXPLICIT', 'RULE'.
 */

/**
 * A user involved in an issue.
 * @typedef {Object} UserValue
 * @property {string} user - The User resource name.
 * @property {string} derivation - How the user was derived. One of 'EXPLICIT', 'RULE'.
 */

/**
 * A component involved in an issue.
 * @typedef {Object} ComponentValue
 * @property {string} component - The ComponentDef resource name.
 * @property {string} derivation - How the component was derived. One of 'EXPLICIT', 'RULE'.
 */

/**
 * A field involved in an issue.
 * @typedef {Object} FieldValue
 * @property {string} field - The FieldDef resource name.
 * @property {string} value - The value associated with the field.
 * @property {string} derivation - How the value was derived. One of 'EXPLICIT', 'RULE'.
 * @property {string} phase - The phase of an issue that this value belongs to, if any.
 */

/**
 * The status of an issue.
 * @typedef {Object} StatusValue
 * @property {string} status - The status. e.g. 'Available'.
 * @property {string} derivation - How the status was derived. One of 'EXPLICIT', 'RULE'.
 */

/**
 * A reference to monorail or external issue.
 * @typedef {Object} IssueRef
 * @property {string} [issue] - The resource name of the issue.
 * @property {string} [extIdentifier] - The identifier of an external issue e.g 'b/123'.
 */

/**
 * An Issue.
 * @typedef {Object} Issue
 * @property {string} name - The resource name of the issue.
 * @property {string} summary - The issue summary.
 * @property {string} state - The current state of the issue. One of 'ACTIVE', 'DELETED', 'SPAM'.
 * @property {string} reporter - The User resource name of the issue reporter.
 * @property {UserValue} owner - The issue's owner.
 * @property {StatusValue} status - The issue status.
 * @property {IssueRef} mergedIntoIssueRef - The issue this issue is merged into.
 * @property {Array<IssueRef>} blockedOnIssueRefs - TODO
 * @property {Array<IssueRef>} blockingIssueRefs - TODO
 * @property {Array<LabelValue>} labels - The labels of the issue.
 * @property {Array<FieldValue>} fieldValues - TODO
 * @property {Array<UserValue>} ccUsers - The users cc'd to this issue.
 * @property {Array<ComponentValue>} components - The Components added to the issue.
 * @property {Number} attachmentCount - The number of attachments this issue holds.
 * @property {Number} starCount - The number of stars this issue has.
 * @property {Array<FieldValue>} fieldValues - The field values of the issue.
 * @property {Array<string>} phases - The names of all Phases in this issue.
 * @property {Object} delta - Holds the pending changes that will be applied with SaveChanges().
 */
// TODO(crbug.com/monorail/6456): createTime, closeTime, modifyTime, componentModifyTime, statusModifyTime, ownerModifyTime

// TODO(crbug.com/monorail/6456): Add other classes.
