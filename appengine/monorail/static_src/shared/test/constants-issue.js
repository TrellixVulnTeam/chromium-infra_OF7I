// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import 'shared/typedef.js';

export const NAME = 'projects/project-name/issues/1234';

export const ISSUE_REF_STRING = 'project-name:1234';

/** @type {IssueRef} */
export const ISSUE_REF = Object.freeze({
  projectName: 'project-name',
  localId: 1234,
});

/** @type {Issue} */
export const ISSUE = Object.freeze({
  projectName: 'project-name',
  localId: 1234,
  statusRef: {status: 'Available', meansOpen: true},
});

export const NAME_OTHER_PROJECT = 'projects/other-project-name/issues/1234';

export const ISSUE_OTHER_PROJECT_REF_STRING = 'other-project-name:1234';

/** @type {Issue} */
export const ISSUE_OTHER_PROJECT = Object.freeze({
  projectName: 'other-project-name',
  localId: 1234,
  statusRef: {status: 'Available', meansOpen: true},
});

export const NAME_CLOSED = 'projects/project-name/issues/5678';

/** @type {Issue} */
export const ISSUE_CLOSED = Object.freeze({
  projectName: 'project-name',
  localId: 5678,
  statusRef: {status: 'Fixed', meansOpen: false},
});
