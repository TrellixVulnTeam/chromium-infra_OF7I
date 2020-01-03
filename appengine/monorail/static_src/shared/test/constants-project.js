// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {fieldTypes} from 'shared/issue-fields.js';
import {USER_REF} from './constants-user.js';
import 'shared/typedef.js';

/** @type {string} */
export const PROJECT_NAME = 'project-name';

/** @type {FieldDef} */
export const FIELD_DEF_INT = Object.freeze({
  fieldRef: Object.freeze({
    fieldId: 123,
    fieldName: 'field-name',
    type: fieldTypes.INT_TYPE,
  }),
});

/** @type {FieldDef} */
export const FIELD_DEF_ENUM = Object.freeze({
  fieldRef: Object.freeze({
    fieldId: 456,
    fieldName: 'enum',
    type: fieldTypes.ENUM_TYPE,
  }),
});

/** @type {Array<FieldDef>} */
export const FIELD_DEFS = [
  FIELD_DEF_INT,
  FIELD_DEF_ENUM,
];

/** @type {Config} */
export const CONFIG = Object.freeze({
  projectName: PROJECT_NAME,
  fieldDefs: FIELD_DEFS,
  labelDefs: [
    {label: 'One'},
    {label: 'EnUm'},
    {label: 'eNuM-Options'},
    {label: 'hello-world', docstring: 'hmmm'},
    {label: 'hello-me', docstring: 'hmmm'},
  ],
});

/** @type {string} */
export const DEFAULT_QUERY = 'owner:me';

/** @type {PresentationConfig} */
export const PRESENTATION_CONFIG = Object.freeze({
  projectThumbnailUrl: 'test.png',
  defaultColSpec: 'ID+Summary+AllLabels',
  defaultQuery: DEFAULT_QUERY,
});

/** @type {{userRefs: Array<UserRef>, groupRefs: Array<UserRef>}} */
export const VISIBLE_MEMBERS = Object.freeze({
  userRefs: [USER_REF],
  groupRefs: [],
});

/** @type {TemplateDef} */
export const TEMPLATE_DEF = Object.freeze({
  templateName: 'Template Name',
});

export const STATE = Object.freeze({project: {
  name: PROJECT_NAME,
  configs: {[PROJECT_NAME]: CONFIG},
  presentationConfigs: {[PROJECT_NAME]: PRESENTATION_CONFIG},
  visibleMembers: {[PROJECT_NAME]: VISIBLE_MEMBERS},
  templates: {[PROJECT_NAME]: TEMPLATE_DEF},
}});
