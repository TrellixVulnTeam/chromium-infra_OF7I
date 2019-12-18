// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Shared file for specifying common types related
 * to projects, used in type annotations across Monorail.
 */

import './typedef-common.js';
import './typedef-issue.js';

/**
 * A StatusDef Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} StatusDef
 * @property {string} status
 * @property {boolean} meansOpen
 * @property {number} rank
 * @property {string} docstring
 * @property {boolean} deprecated
 */

/**
 * A LabelDef Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} LabelDef
 * @property {string} label
 * @property {string} docstring
 * @property {boolean} deprecated
 */

/**
 * A ComponentDef Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} ComponentDef
 * @property {string} path
 * @property {string} docstring
 * @property {Array<UserRef>} adminRefs
 * @property {Array<UserRef>} ccRefs
 * @property {boolean} deprecated
 * @property {number} created
 * @property {UserRef} creatorRef
 * @property {number} modified
 * @property {UserRef} modifierRef
 * @property {Array<LabelRef>} labelRefs
 */

/**
 * A FieldDef Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} FieldDef
 * @property {FieldRef} fieldRef
 * @property {string} applicableType
 * @property {boolean} isRequired
 * @property {boolean} isNiche
 * @property {boolean} isMultivalued
 * @property {string} docstring
 * @property {Array<UserRef>} adminRefs
 * @property {boolean} isPhaseField
 * @property {Array<UserRef>} userChoices
 * @property {Array<LabelDef>} enumChoices
 */

/**
 * A ApprovalDef Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} ApprovalDef
 * @property {FieldRef} fieldRef
 * @property {Array<UserRef>} approverRefs
 * @property {string} survey
 */

/**
 * A Config Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} Config
 * @property {string} projectName
 * @property {Array<StatusDef>} statusDefs
 * @property {Array<StatusRef>} statusesOfferMerge
 * @property {Array<LabelDef>} labelDefs
 * @property {Array<string>} exclusiveLabelPrefixes
 * @property {Array<ComponentDef>} componentDefs
 * @property {Array<FieldDef>} fieldDefs
 * @property {Array<ApprovalDef>} approvalDefs
 * @property {boolean} restrictToKnown
 */


/**
 * A PresentationConfig Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} PresentationConfig
 * @property {string} projectThumbnailUrl
 * @property {string} projectSummary
 * @property {string} customIssueEntryUrl
 * @property {string} defaultQuery
 * @property {Array<SavedQuery>} savedQueries
 * @property {string} revisionUrlFormat
 * @property {string} defaultColSpec
 * @property {string} defaultSortSpec
 * @property {string} defaultXAttr
 * @property {string} defaultYAttr
 */

/**
 * A TemplateDef Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} TemplateDef
 * @property {string} templateName
 * @property {string} content
 * @property {string} summary
 * @property {boolean} summaryMustBeEdited
 * @property {UserRef} ownerRef
 * @property {StatusRef} statusRef
 * @property {Array<LabelRef>} labelRefs
 * @property {boolean} membersOnly
 * @property {boolean} ownerDefaultsToMember
 * @property {Array<UserRef>} adminRefs
 * @property {Array<FieldValue>} fieldValues
 * @property {Array<ComponentRef>} componentRefs
 * @property {boolean} componentRequired
 * @property {Array<Approval>} approvalValues
 * @property {Array<PhaseDef>} phases
 */
