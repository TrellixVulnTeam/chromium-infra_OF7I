// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * @fileoverview Manages the current user's participation in experiments (e.g.
 * phased rollouts).
 *
 * This file is an early prototype serving the needs of go/monorail-slo-v0.
 *
 * The more mature design is under discussion:
 * http://doc/1rtYXq68WSlTNCzVJiSttLWF14CiK5sOlEef2JWAgheg
 */

/**
 * An Enum representing known expreriments.
 *
 * @typedef {string} Experiment
 */

/**
 * @type {Experiment}
 */
export const SLO_EXPERIMENT = 'slo';

const EXPERIMENT_QUERY_PARAM = 'e';

const DISABLED_STR = '-';

const _SLO_EXPERIMENT_USER_DISPLAY_NAMES = new Set([
  'jessan@google.com',
]);

/**
 * Checks whether the current user is in given experiment.
 *
 * @param {Experiment} experiment The experiment to check.
 * @param {UserV0=} user The current user. Although any user can currently
 *     be passed in, we only intend to support checking if the current user is
 *     in the experiment. In the future the user parameter may be removed.
 * @param {Object} queryParams The current query parameters, parsed by qs.
 *     We support a string like 'e=-exp1,-exp2...' for disabling experiments.
 *
 *     We allow disabling so that a user in the fishfood group can work around
 *     any bugs or undesired behaviors the experiment may introduce for them.
 *
 *     As of now, we don't allow enabling experiments by override params.
 *     We may not want access shared beyond the fishfood group (e.g. if it is a
 *     feature we are likely to change dramatically or take away).
 * @return {boolean} Whether the experiment is enabled for the current user.
 */
export const isExperimentEnabled = (experiment, user, queryParams) => {
  const experimentOverrides = parseExperimentParam(
      queryParams[EXPERIMENT_QUERY_PARAM]);
  if (experimentOverrides[experiment] === false) {
    return false;
  }
  switch (experiment) {
    case SLO_EXPERIMENT:
      return !!user &&
        _SLO_EXPERIMENT_USER_DISPLAY_NAMES.has(user.displayName);
    default:
      throw Error('Unknown experiment provided');
  }
};

/**
 * Parses a comma separated list of experiments from the query string.
 * Experiment strings preceded by DISABLED_STR are overrode to be disabled,
 * otherwise they are to be enabled.
 *
 * Does not do any validation of the experiment string provided.
 *
 * @param {string?} experimentParam comma separated experiements.
 * @return {Object} Maps experiment name to whether enabled or
 *    disabled boolean. May include invalid experiment names.
 */
const parseExperimentParam = (experimentParam) => {
  const experimentOverrides = {};
  if (experimentParam) {
    for (const experimentOverride of experimentParam.split(',')) {
      if (experimentOverride.startsWith(DISABLED_STR)) {
        const experiment = experimentOverride.substr(DISABLED_STR.length);
        experimentOverrides[experiment] = false;
      } else {
        experimentOverrides[experimentOverride] = true;
      }
    }
  }
  return experimentOverrides;
};
