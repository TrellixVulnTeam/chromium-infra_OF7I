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
 *
 * Implementation based on the queries defined in Sheriffbot
 * https://team.git.corp.google.com/clusterfuzz-team/sheriffbot/+/refs/heads/master/src/sheriffbot/bug_slo_daily_queries.py
 *
 * @const {SloRule}
 * @private Only visible for testing.
 */
export const _CROS_CLOSURE_SLO = {
  statusFunction: (issue) => {
    if (!_isCrosClosureEligible(issue)) {
      return null;
    }

    const pri = getPriFromIssue(issue);
    const daysToClose = _CROS_CLOSURE_SLO_DAYS_BY_PRIORITY[pri];

    if (!daysToClose) {
      // No applicable SLO found issues with this priority.
      return null;
    }
    // Return a complete status for closed issues.
    if (issue.statusRef && !issue.statusRef.meansOpen) {
      return {
        rule: _CROS_CLOSURE_SLO,
        target: null,
        completion: SloCompletionStatus.COMPLETE};
    }

    // Set the target based on the opening and the daysToClose.
    const target = new Date(issue.openedTimestamp * 1000);
    target.setDate(target.getDate() + daysToClose);
    return {
      rule: _CROS_CLOSURE_SLO,
      target: target,
      completion: SloCompletionStatus.INCOMPLETE};
  },
};

/**
 * @param {Issue} issue
 * @return {string?} the pri's value, if found.
 */
const getPriFromIssue = (issue) => {
  for (const fv of issue.fieldValues) {
    if (fv.fieldRef.fieldName === 'Pri') {
      return fv.value;
    }
  }
};

/**
 * The number of days (since the issue was opened) allowed for it to be fixed.
 * @private Only visible for testing.
 */
export const _CROS_CLOSURE_SLO_DAYS_BY_PRIORITY = Object.freeze({
  '1': 42,
});

// https://team.git.corp.google.com/clusterfuzz-team/sheriffbot/+/refs/heads/master/src/sheriffbot/bug_slo_daily_queries.py#97
const CROS_ELIGIBLE_COMPONENT_PATHS = new Set([
  'OS>Systems>CrashReporting',
  'OS>Systems>Displays',
  'OS>Systems>Feedback',
  'OS>Systems>HaTS',
  'OS>Systems>Input',
  'OS>Systems>Input>Keyboard',
  'OS>Systems>Input>Mouse',
  'OS>Systems>Input>Shortcuts',
  'OS>Systems>Input>Touch',
  'OS>Systems>Metrics',
  'OS>Systems>Multidevice',
  'OS>Systems>Multidevice>Messages',
  'OS>Systems>Multidevice>SmartLock',
  'OS>Systems>Multidevice>Tethering',
  'OS>Systems>Network>Bluetooth',
  'OS>Systems>Network>Cellular',
  'OS>Systems>Network>VPN',
  'OS>Systems>Network>WiFi',
  'OS>Systems>Printing',
  'OS>Systems>Settings',
  'OS>Systems>Spellcheck',
  'OS>Systems>Update',
  'OS>Systems>Wallpaper',
  'OS>Systems>WirelessCharging',
  'Platform>Apps>Feedback',
  'UI>Shell>Networking',
]);

/**
 * Determines if an issue is eligible for _CROS_CLOSURE_SLO.
 * @param {Issue} issue
 * @return {boolean}
 * @private Only visible for testing.
 */
export const _isCrosClosureEligible = (issue) => {
  // If at least one component applies, continue.
  const hasEligibleComponent = issue.componentRefs.some(
      (component) => CROS_ELIGIBLE_COMPONENT_PATHS.has(component.path));
  if (!hasEligibleComponent) {
    return false;
  }

  let priority = null;
  let hasMilestone = false;
  for (const fv of issue.fieldValues) {
    if (fv.fieldRef.fieldName === 'Type') {
      // These types don't apply.
      if (fv.value === 'Feature' || fv.value === 'FLT-Launch' ||
      fv.value === 'Postmortem-Followup' || fv.value === 'Design-Review') {
        return false;
      }
    }
    if (fv.fieldRef.fieldName === 'Pri') {
      priority = fv.value;
    }
    if (fv.fieldRef.fieldName === 'M') {
      hasMilestone = true;
    }
  }
  // P1 issues with milestones don't apply.
  if (priority === '1' && hasMilestone) {
    return false;
  }
  // Issues with the ChromeOS_No_SLO label don't apply.
  for (const labelRef of issue.labelRefs) {
    if (labelRef.label === 'ChromeOS_No_SLO') {
      return false;
    }
  }
  return true;
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
  try {
    for (const rule of SLO_RULES) {
      const status = rule.statusFunction(issue);
      if (status) {
        return status;
      }
    }
  } catch (error) {
    // Don't bubble up any errors in SLO_RULES functions, which might sometimes
    // be written/updated by client teams.
  }
  return null;
};
