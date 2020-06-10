// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {_CROS_CLOSURE_SLO, _CROS_CLOSURE_SLO_DAYS_BY_PRIORITY,
  _isCrosClosureEligible, SloCompletionStatus, determineSloStatus}
  from './slo-rules.js';

const P1_FIELD_VALUE = Object.freeze({
  fieldRef: {
    fieldId: 1,
    fieldName: 'Pri',
    type: 'ENUM_TYPE',
  },
  value: '1'});

// TODO(crbug.com/monorail/7843): Separate testing of determineSloStatus from
// testing of specific SLO Rules. Add testing for a rule that throws an error.
describe('determineSloStatus', () => {
  it('returns null for ineligible issues', () => {
    const ineligibleIssue = {
      componentRefs: [{path: 'Some>Other>Component'}],
      fieldValues: [P1_FIELD_VALUE],
      labelRefs: [],
      localId: 1,
      projectName: 'x',
    };
    assert.isNull(determineSloStatus(ineligibleIssue));
  });

  it('returns null for eligible issues without defined priority', () => {
    const ineligibleIssue = {
      componentRefs: [{path: 'OS>Systems>CrashReporting'}],
      fieldValues: [],
      labelRefs: [],
      localId: 1,
      projectName: 'x',
    };
    assert.isNull(determineSloStatus(ineligibleIssue));
  });

  it('returns SloStatus with target for incomplete eligible issues', () => {
    const openedTimestamp = 1412362587;
    const eligibleIssue = {
      componentRefs: [{path: 'OS>Systems>CrashReporting'}],
      fieldValues: [P1_FIELD_VALUE],
      labelRefs: [],
      localId: 1,
      openedTimestamp: openedTimestamp,
      projectName: 'x',
    };
    const status = determineSloStatus(eligibleIssue);

    const expectedTarget = new Date(openedTimestamp * 1000);
    expectedTarget.setDate(
        expectedTarget.getDate() + _CROS_CLOSURE_SLO_DAYS_BY_PRIORITY['1']);

    assert.equal(status.target.valueOf(), expectedTarget.valueOf());
    assert.equal(status.completion, SloCompletionStatus.INCOMPLETE);
    assert.equal(status.rule, _CROS_CLOSURE_SLO);
  });

  it('returns SloStatus without target for complete eligible issues', () => {
    const eligibleIssue = {
      componentRefs: [{path: 'OS>Systems>CrashReporting'}],
      fieldValues: [P1_FIELD_VALUE],
      labelRefs: [],
      localId: 1,
      projectName: 'x',
      statusRef: {status: 'Closed', meansOpen: false},
    };
    const status = determineSloStatus(eligibleIssue);
    assert.isNull(status.target);
    assert.equal(status.completion, SloCompletionStatus.COMPLETE);
    assert.equal(status.rule, _CROS_CLOSURE_SLO);
  });
});

describe('_isCrosClosureEligible', () => {
  let crosIssue;
  beforeEach(() => {
    crosIssue = {
      componentRefs: [{path: 'OS>Systems>CrashReporting'}],
      fieldValues: [],
      labelRefs: [],
      localId: 1,
      projectName: 'x',
    };
  });

  it('returns true when eligible', () => {
    assert.isTrue(_isCrosClosureEligible(crosIssue));
  });

  it('returns true if at least one eligible component', () => {
    crosIssue.componentRefs.push({path: 'Some>Other>Component'});
    assert.isTrue(_isCrosClosureEligible(crosIssue));
  });

  it('returns false for issues in wrong component', () => {
    crosIssue.componentRefs = [{path: 'Some>Other>Component'}];
    assert.isFalse(_isCrosClosureEligible(crosIssue));
  });

  it('returns false for Feature', () => {
    crosIssue.fieldValues.push(
        {fieldRef: {fieldName: 'Type'}, value: 'Feature'});
    assert.isFalse(_isCrosClosureEligible(crosIssue));
  });

  it('returns false for FLT-Launch', () => {
    crosIssue.fieldValues.push(
        {fieldRef: {fieldName: 'Type'}, value: 'FLT-Launch'});
    assert.isFalse(_isCrosClosureEligible(crosIssue));
  });

  it('returns false for Postmortem-Followup', () => {
    crosIssue.fieldValues.push(
        {fieldRef: {fieldName: 'Type'}, value: 'Postmortem-Followup'});
    assert.isFalse(_isCrosClosureEligible(crosIssue));
  });

  it('returns false for Design-Review', () => {
    crosIssue.fieldValues.push(
        {fieldRef: {fieldName: 'Type'}, value: 'Design-Review'});
    assert.isFalse(_isCrosClosureEligible(crosIssue));
  });

  it('returns true for other types', () => {
    crosIssue.fieldValues.push(
        {fieldRef: {fieldName: 'type'}, value: 'Any-Other-Type'});
    assert.isTrue(_isCrosClosureEligible(crosIssue));
  });

  it('returns false for p1 with milestone', () => {
    crosIssue.fieldValues.push(P1_FIELD_VALUE);
    crosIssue.fieldValues.push({fieldRef: {fieldName: 'M'}, value: 'any'});
    assert.isFalse(_isCrosClosureEligible(crosIssue));
  });

  it('returns true for p1 without milestone', () => {
    crosIssue.fieldValues.push(P1_FIELD_VALUE);
    crosIssue.fieldValues.push({fieldRef: {fieldName: 'Other'}, value: 'any'});
    assert.isTrue(_isCrosClosureEligible(crosIssue));
  });

  it('returns false for ChromeOS_No_SLO label', () => {
    crosIssue.labelRefs.push({label: 'ChromeOS_No_SLO'});
    assert.isFalse(_isCrosClosureEligible(crosIssue));
  });
});
