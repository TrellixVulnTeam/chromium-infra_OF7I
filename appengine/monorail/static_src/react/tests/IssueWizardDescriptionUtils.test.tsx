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
      'test',
      [],
    )
    assert.equal(expandLabels.length, 0);
    expect(expandDescription).to.contain("Example URL: ");
  });
});
