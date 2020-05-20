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

const _SLO_EXPERIMENT_USER_DISPLAY_NAMES = new Set([
  'jessan@google.com',
]);

/**
 * Checks whether the current user is in given experiment.
 *
 * Note: although any user can currently be passed in, this is really only
 * intended to support checking if the current user is in the experiment. In the
 * future the user parameter may be removed.
 *
 * @param {Experiment} experiment The experiment to check.
 * @param {UserV0=} user The current user.
 * @return {boolean} Whether the experiment is enabled for the current user.
 */
export const isExperimentEnabled = (experiment, user) => {
  switch (experiment) {
    case SLO_EXPERIMENT:
      return !!user &&
        _SLO_EXPERIMENT_USER_DISPLAY_NAMES.has(user.displayName);
    default:
      throw Error('Unknown experiment provided');
  }
};
