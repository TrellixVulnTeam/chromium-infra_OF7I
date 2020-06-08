// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {_CROS_CLOSURE_SLO, determineSloStatus, SloCompletionStatus}
  from './slo-rules.js';

describe('determineSloStatus', () => {
  it('returns null for ineligible issues', () => {
    const ineligibleIssue = {localId: 1, projectName: 'x'};
    assert.isNull(determineSloStatus(ineligibleIssue));
  });

  it('returns SloStatus for eligible issues', () => {
    const eligibleIssue = {localId: -1, projectName: 'x'};
    const status = determineSloStatus(eligibleIssue);
    assert.isNull(status.target);
    assert.equal(status.completion, SloCompletionStatus.COMPLETE);
    assert.equal(status.rule, _CROS_CLOSURE_SLO);
  });
});
