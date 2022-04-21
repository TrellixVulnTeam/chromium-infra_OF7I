// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert, expect} from 'chai';
import {expandDescriptions} from 'react/issue-wizard/IssueWizardDescriptionsUtils.tsx';

describe('IssueWizardDescriptionsUtils', () => {
  it('get expandDescription and labels', () => {
    const {expandDescription, expandLabels} = expandDescriptions(
      'Network / Downloading',
      ['test url'],
      false,
      'test',
      [],
    )
    assert.equal(expandLabels.length, 1);
    expect(expandDescription).to.contain("test url");
  });

  it('get proper component value base on user answer', () => {
    const {expandDescription, expandLabels, compVal} = expandDescriptions(
      'Content',
      ['test url', 'LABELS: Yes - this is'],
      false,
      'test',
      [],
    )
    assert.equal(expandLabels.length, 1);
    assert.equal(expandLabels[0].label, 'Type-Bug');
    assert.equal(compVal, 'Blink');
    expect(expandDescription).to.contain("test url");
  });
});
