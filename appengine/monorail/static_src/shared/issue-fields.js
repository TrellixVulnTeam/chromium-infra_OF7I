// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {relativeTime} from
  'elements/chops/chops-timestamp/chops-timestamp-helpers.js';
import {labelRefsToStrings, issueRefsToStrings, componentRefsToStrings,
  userRefsToDisplayNames, statusRefsToStrings, labelNameToLabelPrefix,
} from './converters.js';
import {removePrefix} from './helpers.js';
import {STATUS_ENUM_TO_TEXT} from 'shared/approval-consts.js';
import {fieldValueMapKey} from 'shared/metadata-helpers.js';

// TODO(zhangtiff): Merge this file with metadata-helpers.js.


/** @enum {string} */
export const fieldTypes = Object.freeze({
  APPROVAL_TYPE: 'APPROVAL_TYPE',
  DATE_TYPE: 'DATE_TYPE',
  ENUM_TYPE: 'ENUM_TYPE',
  INT_TYPE: 'INT_TYPE',
  STR_TYPE: 'STR_TYPE',
  USER_TYPE: 'USER_TYPE',
  URL_TYPE: 'URL_TYPE',

  // Frontend types used to handle built in fields like BlockedOn.
  // Although these are not configurable custom field types on the
  // backend, hard-coding these fields types on the frontend allows
  // us to inter-op custom and baked in fields more seamlessly on
  // the frontend.
  ISSUE_TYPE: 'ISSUE_TYPE',
  TIME_TYPE: 'TIME_TYPE',
  COMPONENT_TYPE: 'COMPONENT_TYPE',
  STATUS_TYPE: 'STATUS_TYPE',
  LABEL_TYPE: 'LABEL_TYPE',
  PROJECT_TYPE: 'PROJECT_TYPE',
});

const SPEC_DELIMITER_REGEX = /[\s\+]+/;
export const SITEWIDE_DEFAULT_COLUMNS = ['ID', 'Type', 'Status',
  'Priority', 'Milestone', 'Owner', 'Summary'];

// When no default can is configured, projects use "Open issues".
export const SITEWIDE_DEFAULT_CAN = '2';

export const PHASE_FIELD_COL_DELIMITER_REGEX = /\./;

export const EMPTY_FIELD_VALUE = '----';

export const APPROVER_COL_SUFFIX_REGEX = /\-approver$/i;

/**
 * Parses colspec or groupbyspec values from user input such as form fields
 * or the URL.
 *
 * @param {string} spec a delimited string with spec values to parse.
 * @return {Array} list of spec values represented by the string.
 */
export function parseColSpec(spec = '') {
  return spec.split(SPEC_DELIMITER_REGEX).filter(Boolean);
}

/**
 * Finds the type for an issue based on the issue's custom fields
 * and labels. If there is a custom field named "Type", that field
 * is used, otherwise labels are used.
 * @param {!Array<FieldValue>} fieldValues
 * @param {!Array<LabelRef>} labelRefs
 * @return {string}
 */
export function extractTypeForIssue(fieldValues, labelRefs) {
  if (fieldValues) {
    // If there is a custom field for "Type", use that for type.
    const typeFieldValue = fieldValues.find(
        (f) => (f.fieldRef && f.fieldRef.fieldName.toLowerCase() === 'type'),
    );
    if (typeFieldValue) {
      return typeFieldValue.value;
    }
  }

  // Otherwise, search through labels for a "Type" label.
  if (labelRefs) {
    const typeLabel = labelRefs.find(
        (l) => l.label.toLowerCase().startsWith('type-'));
    if (typeLabel) {
    // Strip length of prefix.
      return typeLabel.label.substr(5);
    }
  }
  return;
}

// TODO(jojwang): monorail:6397, Refactor these specific map producers into
// selectors.
/**
 * Converts issue.fieldValues into a map where values can be looked up given
 * a field value key.
 *
 * @param {Array} fieldValues List of values with a fieldRef attached.
 * @return {Map} keys are a string constructed using fieldValueMapKey() and
 *   values are an Array of value strings.
 */
export function fieldValuesToMap(fieldValues) {
  if (!fieldValues) return new Map();
  const acc = new Map();
  for (const v of fieldValues) {
    if (!v || !v.fieldRef || !v.fieldRef.fieldName || !v.value) continue;
    const key = fieldValueMapKey(v.fieldRef.fieldName,
        v.phaseRef && v.phaseRef.phaseName);
    if (acc.has(key)) {
      acc.get(key).push(v.value);
    } else {
      acc.set(key, [v.value]);
    }
  }
  return acc;
}

/**
 * Converts issue.approvalValues into a map where values can be looked up given
  * a field value key.
  *
  * @param {Array} approvalValues list of approvals with a fieldRef attached.
  * @return {Map} keys are a string constructed using approvalValueFieldMapKey()
  *   and values are an Array of value strings.
  */
export function approvalValuesToMap(approvalValues) {
  if (!approvalValues) return new Map();
  const approvalKeysToValues = new Map();
  for (const av of approvalValues) {
    if (!av || !av.fieldRef || !av.fieldRef.fieldName) continue;
    const key = fieldValueMapKey(av.fieldRef.fieldName);
    // If there is not status for this approval, the value should show NOT_SET.
    approvalKeysToValues.set(key, [STATUS_ENUM_TO_TEXT[av.status || '']]);
  }
  return approvalKeysToValues;
}

/**
 * Converts issue.approvalValues into a map where the approvers can be looked
 * up given a field value key.
 *
 * @param {Array} approvalValues list of approvals with a fieldRef attached.
 * @return {Map} keys are a string constructed using fieldValueMapKey() and
 *   values are an Array of
 */
export function approvalApproversToMap(approvalValues) {
  if (!approvalValues) return new Map();
  const approvalKeysToApprovers = new Map();
  for (const av of approvalValues) {
    if (!av || !av.fieldRef || !av.fieldRef.fieldName ||
        !av.approverRefs) continue;
    const key = fieldValueMapKey(av.fieldRef.fieldName);
    const approvers = av.approverRefs.map((ref) => ref.displayName);
    approvalKeysToApprovers.set(key, approvers);
  }
  return approvalKeysToApprovers;
}


// Helper function used for fields with only one value that can be unset.
const wrapValueIfExists = (value) => value ? [value] : [];


/**
 * @typedef DefaultIssueField
 * @property {string} fieldName
 * @property {fieldTypes} type
 * @property {function(*): Array<string>} extractor
*/
// TODO(zhangtiff): Merge this functionality with extract-grid-data.js
// TODO(zhangtiff): Combine this functionality with mr-metadata and
// mr-edit-metadata to allow more expressive representation of built in fields.
/**
 * @const {Array<DefaultIssueField>}
 */
const defaultIssueFields = Object.freeze([
  {
    fieldName: 'ID',
    type: fieldTypes.ISSUE_TYPE,
    extractor: ({localId, projectName}) => [{localId, projectName}],
  }, {
    fieldName: 'Project',
    type: fieldTypes.PROJECT_TYPE,
    extractor: (issue) => [issue.projectName],
  }, {
    fieldName: 'Attachments',
    type: fieldTypes.INT_TYPE,
    extractor: (issue) => [issue.attachmentCount || 0],
  }, {
    fieldName: 'AllLabels',
    type: fieldTypes.LABEL_TYPE,
    extractor: (issue) => issue.labelRefs || [],
  }, {
    fieldName: 'Blocked',
    type: fieldTypes.STR_TYPE,
    extractor: (issue) => {
      if (issue.blockedOnIssueRefs && issue.blockedOnIssueRefs.length) {
        return ['Yes'];
      }
      return ['No'];
    },
  }, {
    fieldName: 'BlockedOn',
    type: fieldTypes.ISSUE_TYPE,
    extractor: (issue) => issue.blockedOnIssueRefs || [],
  }, {
    fieldName: 'Blocking',
    type: fieldTypes.ISSUE_TYPE,
    extractor: (issue) => issue.blockingIssueRefs || [],
  }, {
    fieldName: 'CC',
    type: fieldTypes.USER_TYPE,
    extractor: (issue) => issue.ccRefs || [],
  }, {
    fieldName: 'Closed',
    type: fieldTypes.TIME_TYPE,
    extractor: (issue) => wrapValueIfExists(issue.closedTimestamp),
  }, {
    fieldName: 'Component',
    type: fieldTypes.COMPONENT_TYPE,
    extractor: (issue) => issue.componentRefs || [],
  }, {
    fieldName: 'ComponentModified',
    type: fieldTypes.TIME_TYPE,
    extractor: (issue) => [issue.componentModifiedTimestamp],
  }, {
    fieldName: 'MergedInto',
    type: fieldTypes.ISSUE_TYPE,
    extractor: (issue) => wrapValueIfExists(issue.mergedIntoIssueRef),
  }, {
    fieldName: 'Modified',
    type: fieldTypes.TIME_TYPE,
    extractor: (issue) => wrapValueIfExists(issue.modifiedTimestamp),
  }, {
    fieldName: 'Reporter',
    type: fieldTypes.USER_TYPE,
    extractor: (issue) => [issue.reporterRef],
  }, {
    fieldName: 'Stars',
    type: fieldTypes.INT_TYPE,
    extractor: (issue) => [issue.starCount || 0],
  }, {
    fieldName: 'Status',
    type: fieldTypes.STATUS_TYPE,
    extractor: (issue) => wrapValueIfExists(issue.statusRef),
  }, {
    fieldName: 'StatusModified',
    type: fieldTypes.TIME_TYPE,
    extractor: (issue) => [issue.statusModifiedTimestamp],
  }, {
    fieldName: 'Summary',
    type: fieldTypes.STR_TYPE,
    extractor: (issue) => [issue.summary],
  }, {
    fieldName: 'Type',
    type: fieldTypes.ENUM_TYPE,
    extractor: (issue) => wrapValueIfExists(extractTypeForIssue(
        issue.fieldValues, issue.labelRefs)),
  }, {
    fieldName: 'Owner',
    type: fieldTypes.USER_TYPE,
    extractor: (issue) => wrapValueIfExists(issue.ownerRef),
  }, {
    // TODO(zhangtiff): Call ListReferencedUsers if the user views the
    // OwnerLastVisit column to get the timestamps for each owner's last visit.
    fieldName: 'OwnerLastVisit',
    type: fieldTypes.TIME_TYPE,
    extractor: (issue) => [],
  }, {
    fieldName: 'OwnerModified',
    type: fieldTypes.TIME_TYPE,
    extractor: (issue) => [issue.ownerModifiedTimestamp],
  }, {
    fieldName: 'Opened',
    type: fieldTypes.TIME_TYPE,
    extractor: (issue) => [issue.openedTimestamp],
  },
]);

/**
 * Lowercase field name -> field object. This uses an Object instead of a Map
 * so that it can be frozen.
 * @type {Object.<string, DefaultIssueField>}
 */
export const defaultIssueFieldMap = Object.freeze(
    defaultIssueFields.reduce((acc, field) => {
      acc[field.fieldName.toLowerCase()] = field;
      return acc;
    }, {}),
);

export const DEFAULT_ISSUE_FIELD_LIST = defaultIssueFields.map(
    (field) => field.fieldName);

export const stringValuesForIssueField = (issue, fieldName, projectName,
    fieldDefMap = new Map(), labelPrefixSet = new Set()) => {
  const fieldKey = fieldName.toLowerCase();

  // Look at whether the field is a built in field first.
  if (defaultIssueFieldMap.hasOwnProperty(fieldKey)) {
    const bakedFieldDef = defaultIssueFieldMap[fieldKey];
    const values = bakedFieldDef.extractor(issue);
    switch (bakedFieldDef.type) {
      case fieldTypes.ISSUE_TYPE:
        return issueRefsToStrings(values, projectName);
      case fieldTypes.COMPONENT_TYPE:
        return componentRefsToStrings(values);
      case fieldTypes.LABEL_TYPE:
        return labelRefsToStrings(values);
      case fieldTypes.USER_TYPE:
        return userRefsToDisplayNames(values);
      case fieldTypes.STATUS_TYPE:
        return statusRefsToStrings(values);
      case fieldTypes.TIME_TYPE:
        // TODO(zhangtiff): Find a way to dynamically update displayed
        // time without page reloads.
        return values.map((time) => relativeTime(new Date(time * 1000)));
    }
    return values.map((value) => `${value}`);
  }

  // Handle custom approval field approver columns.
  const found = fieldKey.match(APPROVER_COL_SUFFIX_REGEX);
  if (found) {
    const approvalName = fieldKey.slice(0, -found[0].length);
    const approvalFieldKey = fieldValueMapKey(approvalName);
    if (fieldDefMap.has(approvalFieldKey)) {
      const approvalApproversMap = approvalApproversToMap(issue.approvalValues);
      if (approvalApproversMap.has(approvalFieldKey)) {
        return approvalApproversMap.get(approvalFieldKey);
      }
    }
  }

  // Handle custom approval field columns.
  if (fieldDefMap.has(fieldKey) && fieldDefMap.get(fieldKey).fieldRef &&
      fieldDefMap.get(fieldKey).fieldRef.type == fieldTypes.APPROVAL_TYPE) {
    const approvalValuesMap = approvalValuesToMap(issue.approvalValues);
    if (approvalValuesMap.has(fieldKey)) {
      return approvalValuesMap.get(fieldKey);
    }
  }

  // Handle custom fields.
  let fieldValueKey = fieldKey;
  let fieldNameKey = fieldKey;
  if (fieldKey.match(PHASE_FIELD_COL_DELIMITER_REGEX)) {
    let phaseName;
    [phaseName, fieldNameKey] = fieldKey.split(
        PHASE_FIELD_COL_DELIMITER_REGEX);
    // key for fieldValues Map contain the phaseName, if any.
    fieldValueKey = fieldValueMapKey(fieldNameKey, phaseName);
  }
  if (fieldDefMap.has(fieldNameKey)) {
    const fieldValuesMap = fieldValuesToMap(issue.fieldValues);
    if (fieldValuesMap.has(fieldValueKey)) {
      return fieldValuesMap.get(fieldValueKey);
    }
  }

  // Label options are last in precedence.
  if (labelPrefixSet.has(fieldKey)) {
    const matchingLabels = (issue.labelRefs || []).filter((labelRef) => {
      const labelPrefixKey = labelNameToLabelPrefix(
          labelRef.label).toLowerCase();
      return fieldKey === labelPrefixKey;
    });
    const labelPrefix = fieldKey + '-';
    return matchingLabels.map(
        (labelRef) => removePrefix(labelRef.label, labelPrefix));
  }

  return [];
};

// TODO(zhangtiff): Implement hotlist specific fields: Rank, Added, Adder.
