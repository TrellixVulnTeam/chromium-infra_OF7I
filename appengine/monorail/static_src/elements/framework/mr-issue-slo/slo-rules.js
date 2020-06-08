// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Determining Issues' statuses relative to SLO rules.
 *
 * See go/monorail-slo-v0 for more info.
 */

/**
 * A rule determining the compliance of an issue with regard to an SLO.
 * @typedef {Object} SloRule
 * @property {function(Issue): SloStatus} statusFunction
 */

/**
 * Potential statuses of an issue relative to an SLO's completion criteria.
 * @enum {string}
 */
export const SloCompletionStatus = {
  /** The completion criteria for the SloRule have not been satisfied. */
  INCOMPLETE: 'INCOMPLETE',
  /** The completion criteria for the SloRule have been satisfied. */
  COMPLETE: 'COMPLETE',
};

/**
 * The status of an issue with regard to an SloRule.
 * @typedef {Object} SloStatus
 * @property {SloRule} rule The rule that generated this status.
 * @property {Date} target The time the Issue must move to completion, or null
 *     if the issue has already moved to completion.
 * @property {SloCompletionStatus} completion Issue's completion status.
 */

/**
 * Chrome OS Software's SLO for issue closure (go/chromeos-software-bug-slos).
 * @const {SloRule}
 * @private Only visible for testing.
 */
export const _CROS_CLOSURE_SLO = {
  // TODO(crbug.com/monorail/7740): Implement and test a real status function.
  statusFunction: (issue) => {
    if (!isCrosClosureEligible(issue)) {
      return null;
    }
    return {
      rule: _CROS_CLOSURE_SLO,
      target: null,
      completion: SloCompletionStatus.COMPLETE};
  },
};

/**
 * Determines if an issue is eligible for _CROS_CLOSURE_SLO.
 * @param {Issue} issue
 * @return {boolean}
 */
const isCrosClosureEligible = (issue) => {
  if (issue.localId === -1) {
    return true;
  }
  return false;
};

/**
 * Active SLO Rules.
 * @const {Array<SloRule>}
 */
const SLO_RULES = [_CROS_CLOSURE_SLO];

/**
 * Determines the SloStatus for the given issue.
 * @param {Issue} issue The issue to check.
 * @return {SloStatus} The status of the issue, or null if no rules apply.
 */
export const determineSloStatus = (issue) => {
  for (const rule of SLO_RULES) {
    const status = rule.statusFunction(issue);
    if (status) {
      return status;
    }
  }
  return null;
};
