// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {assert} from 'chai';
import {render} from '@testing-library/react';

import {IssueWizard} from './IssueWizard.tsx';

describe('IssueWizard', () => {
  it('renders', async () => {
    render(<IssueWizard loginUrl="login" userDisplayName="user"/>);

    const stepper = document.getElementById("mobile-stepper")

    assert.isNotNull(stepper);
  });
});
