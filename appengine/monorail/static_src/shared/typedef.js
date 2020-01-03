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
 * Types used in the app that don't come from any Proto files.
 */

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
 * @property {boolean=} requesting Whether a request is in flight.
 * @property {Error=} error An Error Object returned by the request.
 */


/**
 * Types defined in common.proto.
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
 * @property {string=} approvalName
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


/**
 * Types defined in issue_objects.proto.
 */

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
 * @property {number=} sequenceNum
 * @property {boolean=} isDeleted
 * @property {UserRef=} commenter
 * @property {number=} timestamp
 * @property {string=} content
 * @property {string=} inboundMessage
 * @property {Array<Amendment>=} amendments
 * @property {Array<Attachment>=} attachments
 * @property {FieldRef=} approvalRef
 * @property {number=} descriptionNum
 * @property {boolean=} isSpam
 * @property {boolean=} canDelete
 * @property {boolean=} canFlag
 */

/**
 * A FieldValue Object returned by the pRPC API issue_objects.proto.
 *
 * @typedef {Object} FieldValue
 * @property {FieldRef} fieldRef
 * @property {string} value
 * @property {boolean=} isDerived
 * @property {PhaseRef=} phaseRef
 */

/**
 * An Issue Object returned by the pRPC API issue_objects.proto.
 *
 * @typedef {Object} Issue
 * @property {string} projectName
 * @property {number} localId
 * @property {string=} summary
 * @property {StatusRef=} statusRef
 * @property {UserRef=} ownerRef
 * @property {Array<UserRef>=} ccRefs
 * @property {Array<LabelRef>=} labelRefs
 * @property {Array<ComponentRef>=} componentRefs
 * @property {Array<IssueRef>=} blockedOnIssueRefs
 * @property {Array<IssueRef>=} blockingIssueRefs
 * @property {Array<IssueRef>=} danglingBlockedOnRefs
 * @property {Array<IssueRef>=} danglingBlockingRefs
 * @property {IssueRef=} mergedIntoIssueRef
 * @property {FieldValue=} fieldValues
 * @property {boolean=} isDeleted
 * @property {UserRef=} reporterRef
 * @property {number=} openedTimestamp
 * @property {number=} closedTimestamp
 * @property {number=} modifiedTimestamp
 * @property {number=} componentModifiedTimestamp
 * @property {number=} statusModifiedTimestamp
 * @property {number=} ownerModifiedTimestamp
 * @property {number=} starCount
 * @property {boolean=} isSpam
 * @property {number=} attachmentCount
 * @property {Array<Approval>=} approvalValues
 * @property {Array<PhaseDef>=} phases
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


/**
 * Types defined in project_objects.proto.
 */

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
 * @property {string=} docstring
 * @property {boolean=} deprecated
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
 * @property {string=} applicableType
 * @property {boolean=} isRequired
 * @property {boolean=} isNiche
 * @property {boolean=} isMultivalued
 * @property {string=} docstring
 * @property {Array<UserRef>=} adminRefs
 * @property {boolean=} isPhaseField
 * @property {Array<UserRef>=} userChoices
 * @property {Array<LabelDef>=} enumChoices
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
 * @property {Array<StatusDef>=} statusDefs
 * @property {Array<StatusRef>=} statusesOfferMerge
 * @property {Array<LabelDef>=} labelDefs
 * @property {Array<string>=} exclusiveLabelPrefixes
 * @property {Array<ComponentDef>=} componentDefs
 * @property {Array<FieldDef>=} fieldDefs
 * @property {Array<ApprovalDef>=} approvalDefs
 * @property {boolean=} restrictToKnown
 */


/**
 * A PresentationConfig Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} PresentationConfig
 * @property {String=} projectThumbnailUrl
 * @property {string=} projectSummary
 * @property {string=} customIssueEntryUrl
 * @property {string=} defaultQuery
 * @property {Array<SavedQuery>=} savedQueries
 * @property {string=} revisionUrlFormat
 * @property {string=} defaultColSpec
 * @property {string=} defaultSortSpec
 * @property {string=} defaultXAttr
 * @property {string=} defaultYAttr
 */

/**
 * A TemplateDef Object returned by the pRPC API project_objects.proto.
 *
 * @typedef {Object} TemplateDef
 * @property {string} templateName
 * @property {string=} content
 * @property {string=} summary
 * @property {boolean=} summaryMustBeEdited
 * @property {UserRef=} ownerRef
 * @property {StatusRef=} statusRef
 * @property {Array<LabelRef>=} labelRefs
 * @property {boolean=} membersOnly
 * @property {boolean=} ownerDefaultsToMember
 * @property {Array<UserRef>=} adminRefs
 * @property {Array<FieldValue>=} fieldValues
 * @property {Array<ComponentRef>=} componentRefs
 * @property {boolean=} componentRequired
 * @property {Array<Approval>=} approvalValues
 * @property {Array<PhaseDef>=} phases
 */


/**
 * Types defined in features_objects.proto.
 */

/**
 * A Hotlist Object returned by the pRPC API features_objects.proto.
 *
 * @typedef {Object} Hotlist
 * @property {UserRef=} ownerRef
 * @property {string=} name
 * @property {string=} summary
 * @property {string=} description
 * @property {string=} defaultColSpec
 * @property {boolean=} isPrivate
 */

/**
 * A HotlistItem Object returned by the pRPC API features_objects.proto.
 *
 * @typedef {Object} HotlistItem
 * @property {Issue=} issue
 * @property {number=} rank
 * @property {UserRef=} adderRef
 * @property {number=} addedTimestamp
 * @property {string=} note
 */

/**
 * Types defined in user_objects.proto.
 */

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
