// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {EMPTY_FIELD_VALUE,
  stringValuesForIssueField} from 'shared/issue-fields.js';
import {getTypeForFieldName, fieldTypes} from 'shared/issue-fields';


const DEFAULT_HEADER_VALUE = 'All';

// A list of the valid default field names available in an issue grid.
// High cardinality fields must be excluded, so the grid only includes a subset
// of AVAILABLE FIELDS.
export const DEFAULT_GRID_FIELD_NAMES = [
  'Project',
  'Attachments',
  'Blocked',
  'BlockedOn',
  'Blocking',
  'Component',
  'MergedInto',
  'Reporter',
  'Stars',
  'Status',
  'Type',
  'Owner',
];

const GROUPABLE_FIELD_TYPES = new Set([
  fieldTypes.DATE_TYPE,
  fieldTypes.ENUM_TYPE,
  fieldTypes.USER_TYPE,
  fieldTypes.INT_TYPE,
]);

/**
 * Returns the fields available given these fieldDefs and labelFields.
 *
 * A special value of 'None' will always be prepended to the otherwise sorted
 * list returned.
 *
 * @param {Iterable<FieldDef>=} fieldDefs
 * @param {Iterable<string>=} labelFields
 * @return {Array<string>}
 */
export function getAvailableGridFields(fieldDefs = [], labelFields = []) {
  // TODO(jessan): Consider whether the deduplication is needed.
  const gridFieldSet = new Set([...DEFAULT_GRID_FIELD_NAMES, ...labelFields]);
  for (const fd of fieldDefs) {
    if (GROUPABLE_FIELD_TYPES.has(fd.fieldRef.type)) {
      gridFieldSet.add(fd.fieldRef.fieldName);
    }
  };

  const gridFieldList = [...gridFieldSet];
  gridFieldList.sort();
  gridFieldList.unshift('None');
  return gridFieldList;
};

// Sort headings functions
// TODO(zhangtiff): Find some way to restructure this code to allow
// sorting functions to sort with raw types instead of stringified values.

/**
 * Used as an optional 'compareFunction' for Array.sort().
 * @param {string} strA
 * @param {string} strB
 * @return {number}
 */
function intStrComparator(strA, strB) {
  return parseInt(strA) - parseInt(strB);
}

/**
 * Used as an optional 'compareFunction' for Array.sort()
 * @param {string} issueRefStrA
 * @param {string} issueRefStrB
 * @return {number}
 */
function issueRefComparator(issueRefStrA, issueRefStrB) {
  const issueRefA = issueRefStrA.split(':');
  const issueRefB = issueRefStrB.split(':');
  if (issueRefA[0] != issueRefB[0]) {
    return issueRefStrA.localeCompare(issueRefStrB);
  } else {
    return parseInt(issueRefA[1]) - parseInt(issueRefB[1]);
  }
}

/**
 * Returns a comparator for strings representing statuses using the ordering
 * provided in statusDefs.
 * Any status not found in statusDefs will be sorted to the end.
 * @param {!Array<StatusDef>} statusDefs
 * @return {function(string, string): number}
 */
function getStatusDefComparator(statusDefs) {
  return (statusStrA, statusStrB) => {
    // Traverse statusDefs to determine which status is first.
    for (const statusDef of statusDefs) {
      if (statusDef.status == statusStrA) {
        return -1;
      } else if (statusDef.status == statusStrB) {
        return 1;
      }
    }
    return 0;
  };
}

/**
 * @param {!Set<string>} headingSet The headers found for the field.
 * @param {string} fieldName The field on which we're sorting.
 * @param {!Map} fieldDefMap
 * @param {!Array<StatusDef>} statusDefs
 * @return {!Array<string>}
 */
function sortHeadings(headingSet, fieldName, fieldDefMap, statusDefs) {
  let sorter;
  const type = getTypeForFieldName(fieldName, fieldDefMap);
  if (type === fieldTypes.ISSUE_TYPE) {
    sorter = issueRefComparator;
  } else if (type === fieldTypes.INT_TYPE) {
    sorter = intStrComparator;
  } else if (type === fieldTypes.STATUS_TYPE) {
    sorter = getStatusDefComparator(statusDefs);
  }


  // Track whether EMPTY_FIELD_VALUE is present, and ensure that
  // it is sorted to the last position even for custom fields.
  // TODO(jessan): although convenient, it is bad practice to mutate parameters.
  const hasEmptyFieldValue = headingSet.delete(EMPTY_FIELD_VALUE);
  const headingsList = [...headingSet];

  headingsList.sort(sorter);

  if (hasEmptyFieldValue) {
    headingsList.push(EMPTY_FIELD_VALUE);
  }
  return headingsList;
}

/**
 * @param {string} x Header value.
 * @param {string} y Header value.
 * @return {string} The key for the groupedIssue map.
 * TODO(jessan): Make a GridData class, which avoids exposing this logic.
 */
export function makeGridCellKey(x, y) {
  // Note: Some possible x and y values contain ':', '-', and other
  // non-word characters making delimiter options limited.
  return x + ' + ' + y;
}

/**
 * @param {Issue} issue The issue for which we're preparing grid headings.
 * @param {string} fieldName The field on which we're grouping.
 * @param {string} projectName
 * @param {!Map} fieldDefMap
 * @param {!Set} labelPrefixSet
 * @return {!Array<string>} The headings the issue should be grouped into.
 */
function prepareHeadings(
    issue, fieldName, projectName, fieldDefMap, labelPrefixSet) {
  const values = stringValuesForIssueField(
      issue, fieldName, projectName, fieldDefMap, labelPrefixSet);

  return values.length == 0 ?
     [EMPTY_FIELD_VALUE] :
     values;
}

/**
 * Groups issues by their values for the given fields.
 *
 * @param {Array<Issue>} issues The issues we are grouping.
 * @param {string=} xFieldName name of the field for grouping columns.
 * @param {string=} yFieldName name of the field for grouping rows.
 * @param {string=} projectName
 * @param {Map=} fieldDefMap
 * @param {Set=} labelPrefixSet
 * @param {Array<StatusDef>=} statusDefs
 * @return {!Object} Grid data
 *   - groupedIssues: A map of issues grouped by thir xField and yField values.
 *   - xHeadings: sorted headings for columns.
 *   - yHeadings: sorted headings for rows.
 */
export function extractGridData(
    issues, xFieldName = '', yFieldName = '', projectName = '',
    fieldDefMap = new Map(), labelPrefixSet = new Set(), statusDefs = []) {
  const xHeadingsSet = new Set();
  const yHeadingsSet = new Set();
  const groupedIssues = new Map();
  for (const issue of issues) {
    const xHeadings = !xFieldName ?
        [DEFAULT_HEADER_VALUE] :
        prepareHeadings(
            issue, xFieldName, projectName, fieldDefMap, labelPrefixSet);
    const yHeadings = !yFieldName ?
        [DEFAULT_HEADER_VALUE] :
        prepareHeadings(
            issue, yFieldName, projectName, fieldDefMap, labelPrefixSet);

    // Find every combo of 'xValue yValue' that the issue belongs to
    // and add it into that cell. Also record each header used.
    for (const xHeading of xHeadings) {
      xHeadingsSet.add(xHeading);
      for (const yHeading of yHeadings) {
        yHeadingsSet.add(yHeading);
        const cellKey = makeGridCellKey(xHeading, yHeading);
        if (groupedIssues.has(cellKey)) {
          groupedIssues.get(cellKey).push(issue);
        } else {
          groupedIssues.set(cellKey, [issue]);
        }
      }
    }
  }

  return {
    groupedIssues,
    xHeadings: sortHeadings(xHeadingsSet, xFieldName, fieldDefMap, statusDefs),
    yHeadings: sortHeadings(yHeadingsSet, yFieldName, fieldDefMap, statusDefs),
  };
}
