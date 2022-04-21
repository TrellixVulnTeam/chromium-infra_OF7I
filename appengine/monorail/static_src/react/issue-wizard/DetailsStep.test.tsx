// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {render, cleanup} from '@testing-library/react';
import {assert} from 'chai';

import DetailsStep from './DetailsStep.tsx';

describe('DetailsStep', () => {
  afterEach(cleanup);

  it('renders', async () => {
    const textFiled = {
      oneLineSummary: '',
      stepsToReproduce: '',
      describeProblem: '',
    };
    const {container} = render(<DetailsStep textValues={textFiled} setIsRegression={() => {}}/>);

    // this is checking for the first question
    const input = container.querySelector('input');
    assert.isNotNull(input)

    // this is checking for the rest
    const count = document.querySelectorAll('textarea').length;
    assert.equal(count, 4)
  });

  it('renders category in title', async () => {
    const textFiled = {
      oneLineSummary: '',
      stepsToReproduce: '',
      describeProblem: '',
    };

    const {container} = render(<DetailsStep category='UI' textValues={textFiled} setIsRegression={() => {}}/>);

    // this is checking the title contains our category
    const title = container.querySelector('h2');
    assert.include(title?.innerText, 'Details for problems with UI');
  });

});
