// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {EMPTY_FIELD_VALUE, fieldTypes} from 'shared/issue-fields.js';
import 'shared/typedef.js';


const DEFAULT_HEADER_VALUE = 'All';

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
 * @param {!Array<StatusDef>=} statusDefs
 * @return {function(string, string): number}
 */
function getStatusDefComparator(statusDefs = []) {
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
 * @param {function(string): string=} extractTypeForFieldName
 * @param {!Array<StatusDef>=} statusDefs
 * @return {!Array<string>}
 */
function sortHeadings(headingSet, fieldName, extractTypeForFieldName,
    statusDefs = []) {
  let sorter;
  if (extractTypeForFieldName) {
    const type = extractTypeForFieldName(fieldName);
    if (type === fieldTypes.ISSUE_TYPE) {
      sorter = issueRefComparator;
    } else if (type === fieldTypes.INT_TYPE) {
      sorter = intStrComparator;
    } else if (type === fieldTypes.STATUS_TYPE) {
      sorter = getStatusDefComparator(statusDefs);
    }
  }

  // Track whether EMPTY_FIELD_VALUE is present, and ensure that
  // it is sorted to the first position of custom fields.
  // TODO(jessan): although convenient, it is bad practice to mutate parameters.
  const hasEmptyFieldValue = headingSet.delete(EMPTY_FIELD_VALUE);
  const headingsList = [...headingSet];

  headingsList.sort(sorter);

  if (hasEmptyFieldValue) {
    headingsList.unshift(EMPTY_FIELD_VALUE);
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
 * @param {function(Issue, string): Array<string>} extractFieldValuesFromIssue
 * @return {!Array<string>} The headings the issue should be grouped into.
 */
function prepareHeadings(
    issue, fieldName, extractFieldValuesFromIssue) {
  const values = extractFieldValuesFromIssue(issue, fieldName);

  return values.length == 0 ?
     [EMPTY_FIELD_VALUE] :
     values;
}

/**
 * Groups issues by their values for the given fields.
 * @param {Array<Issue>} required.issues The issues we are grouping
 * @param {function(Issue, string): Array<string>}
 *     required.extractFieldValuesFromIssue
 * @param {string=} options.xFieldName name of the field for grouping columns
 * @param {string=} options.yFieldName name of the field for grouping rows
 * @param {function(string): string=} options.extractTypeForFieldName
 * @param {Array=} options.statusDefs
 * @param {Map=} options.labelPrefixValueMap
 * @return {!Object} Grid data
 *   - groupedIssues: A map of issues grouped by thir xField and yField values.
 *   - xHeadings: sorted headings for columns.
 *   - yHeadings: sorted headings for rows.
 */
export function extractGridData({issues, extractFieldValuesFromIssue}, {
  xFieldName = '',
  yFieldName = '',
  extractTypeForFieldName = undefined,
  statusDefs = [],
  labelPrefixValueMap = new Map(),
} = {}) {
  const xHeadingsPredefinedSet = new Set();
  const xHeadingsAdHocSet = new Set();
  const yHeadingsSet = new Set();
  const groupedIssues = new Map();
  for (const issue of issues) {
    const xHeadings = !xFieldName ?
        [DEFAULT_HEADER_VALUE] :
        prepareHeadings(
            issue, xFieldName, extractFieldValuesFromIssue);
    const yHeadings = !yFieldName ?
        [DEFAULT_HEADER_VALUE] :
        prepareHeadings(
            issue, yFieldName, extractFieldValuesFromIssue);

    // Find every combo of 'xValue yValue' that the issue belongs to
    // and add it into that cell. Also record each header used.
    for (const xHeading of xHeadings) {
      if (labelPrefixValueMap.has(xFieldName) &&
          labelPrefixValueMap.get(xFieldName).has(xHeading)) {
        xHeadingsPredefinedSet.add(xHeading);
      } else {
        xHeadingsAdHocSet.add(xHeading);
      }
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

  // Predefined labels to be ordered in front of ad hoc labels
  const xHeadings = [
    ...sortHeadings(
        xHeadingsPredefinedSet,
        xFieldName,
        extractTypeForFieldName,
        statusDefs,
    ),
    ...sortHeadings(
        xHeadingsAdHocSet,
        xFieldName,
        extractTypeForFieldName,
        statusDefs,
    ),
  ];

  return {
    groupedIssues,
    xHeadings,
    yHeadings: sortHeadings(yHeadingsSet, yFieldName, extractTypeForFieldName,
        statusDefs),
  };
}
