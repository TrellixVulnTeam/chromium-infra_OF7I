// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {isExperimentEnabled, SLO_EXPERIMENT} from './experiments.js';


describe('isExperimentEnabled', () => {
  it('throws error for unknown experiment', () => {
    assert.throws(() =>
      isExperimentEnabled('unknown-exp', {displayName: 'jessan@google.com'}));
  });

  it('returns false if user not in experiment', () => {
    const ineligibleUser = {displayName: 'example@example.com'};
    assert.isFalse(isExperimentEnabled(SLO_EXPERIMENT, ineligibleUser));
  });

  it('returns false if no user provided', () => {
    assert.isFalse(isExperimentEnabled(SLO_EXPERIMENT, undefined));
  });

  it('returns true if user in experiment', () => {
    const eligibleUser = {displayName: 'jessan@google.com'};
    assert.isTrue(isExperimentEnabled(SLO_EXPERIMENT, eligibleUser));
  });
});
