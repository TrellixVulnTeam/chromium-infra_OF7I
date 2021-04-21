// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {assert} from 'chai';
import {render} from '@testing-library/react';

import {IssueWizard} from './IssueWizard.tsx';

describe('IssueWizard', () => {
  it('renders', async () => {
    const {container} = render(<IssueWizard />);

    const paragraph = container.querySelector('p');

    assert.include(paragraph?.innerText, 'Welcome to the new issue wizard');
  });
});
