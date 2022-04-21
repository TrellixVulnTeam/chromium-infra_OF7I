// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {assert} from 'chai';
import {cleanup, render} from '@testing-library/react';
import {ConfirmBackModal} from 'react/issue-wizard/ConfirmBackModal.tsx';

describe('IssueWizard confirm back modal', () => {

  afterEach(cleanup);

  it('render', () => {
    render(<ConfirmBackModal enable={true} setEnable={()=>{}} confirmBack={()=>{}}/>);
    const buttons = document.querySelectorAll('Button');
    assert.equal(2, buttons.length);
  });
});
