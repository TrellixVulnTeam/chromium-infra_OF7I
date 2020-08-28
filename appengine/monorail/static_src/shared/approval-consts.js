// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview A file containing common constants used in the Approvals
 * feature.
 */

// Only approvers are allowed to set an approval state to one of these states.
export const APPROVER_RESTRICTED_STATUSES = new Set(
    ['NA', 'Approved', 'NotApproved']);

// Map the internal enum names used in Monorail's backend from approval
// statuses to user friendly names.
export const STATUS_ENUM_TO_TEXT = {
  '': 'NotSet',
  'NEEDS_REVIEW': 'NeedsReview',
  'NA': 'NA',
  'REVIEW_REQUESTED': 'ReviewRequested',
  'REVIEW_STARTED': 'ReviewStarted',
  'NEED_INFO': 'NeedInfo',
  'APPROVED': 'Approved',
  'NOT_APPROVED': 'NotApproved',
};

// Reverse mapping of user friendly names to internal enum names.
// Note that NotSet -> NOT_SET maps differently in reverse because
// the backend sends an empty message to communicate NOT_SET.
export const TEXT_TO_STATUS_ENUM = {
  'NotSet': 'NOT_SET',
  'NeedsReview': 'NEEDS_REVIEW',
  'NA': 'NA',
  'ReviewRequested': 'REVIEW_REQUESTED',
  'ReviewStarted': 'REVIEW_STARTED',
  'NeedInfo': 'NEED_INFO',
  'Approved': 'APPROVED',
  'NotApproved': 'NOT_APPROVED',
};

// Statuses mapped to CSS classes used to apply custom styles per
// status like background colors.
export const STATUS_CLASS_MAP = {
  'NotSet': 'status-notset',
  'NeedsReview': 'status-notset',
  'NA': 'status-na',
  'ReviewRequested': 'status-pending',
  'ReviewStarted': 'status-pending',
  'NeedInfo': 'status-pending',
  'Approved': 'status-approved',
  'NotApproved': 'status-rejected',
};

// Hardcoded frontent documentation for each approval status.
export const STATUS_DOCSTRING_MAP = {
  'NotSet': '',
  'NeedsReview': 'Approval gate needs work',
  'NA': 'Approval gate not required',
  'ReviewRequested': 'Approval requested',
  'ReviewStarted': 'Approval in progress',
  'NeedInfo': 'Approval review needs more information',
  'Approved': 'Approved for Launch',
  'NotApproved': 'Not Approved for Launch',
};

// The Material Design icon names that are attached to each
// CSS class.
export const CLASS_ICON_MAP = {
  'status-na': 'remove',
  'status-notset': 'warning',
  'status-pending': 'autorenew',
  'status-approved': 'done',
  'status-rejected': 'close',
};

// Statuses formated as an Array rather than an Object for ease of use
// by components.
export const APPROVAL_STATUSES = Object.keys(STATUS_CLASS_MAP).map(
    (status) => ({status, docstring: STATUS_DOCSTRING_MAP[status], rank: 1}));
