// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import {MrIssueSlo} from './mr-issue-slo.js';


let element;

describe('mr-issue-slo', () => {
  beforeEach(() => {
    element = document.createElement('mr-issue-slo');
    document.body.appendChild(element);
  });

  afterEach(() => {
    document.body.removeChild(element);
  });

  it('initializes', () => {
    assert.instanceOf(element, MrIssueSlo);
  });

  it('handles ineligible issues', async () => {
    element._determineSloStatus = () => {
      return null;
    };
    element.issue = {};
    await element.updateComplete;
    assert.equal(element.shadowRoot.textContent, 'N/A');
  });

  it('handles issues that have completed the SLO criteria', async () => {
    element._determineSloStatus = () => {
      return {target: null};
    };
    element.issue = {};
    await element.updateComplete;
    assert.equal(element.shadowRoot.textContent, 'Done');
  });

  it('handles issues that have not completed the SLO criteria', async () => {
    element._determineSloStatus = () => {
      return {target: 1234};
    };
    element.issue = {};
    await element.updateComplete;
    const timestampElement =
        element.shadowRoot.querySelector('chops-timestamp');

    assert.equal(timestampElement.timestamp, 1234);
  });
});
