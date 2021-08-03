// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {assert} from 'chai';

import DotMobileStepper from './DotMobileStepper.tsx';

describe('DotMobileStepper', () => {
  beforeEach(() => {
    render(<DotMobileStepper />);
  });

  it('renders', () => {
    // this is checking the buttons for the stepper rendered
      const count = document.querySelectorAll('button').length;
      assert.equal(count, 2)
  });

  it('back button disabled on first step', () => {
    // Finds a button on the page with "back" as text using React testing library.
    const backButton = screen.getByRole('button', {name: /backButton/i});

    // Back button is disabled on the first step.
    assert.isNotNull(backButton.getAttribute('disabled'));

    // Click the next button to move to the second step.
    const nextButton = screen.getByRole('button', {name: /nextButton/i});

    userEvent.click(nextButton);

    // Back button is not disabled on the second step.
    assert.isNull(backButton.getAttribute('disabled'));
  });

  it('next button disabled on last step', () => {
    // Finds a button on the page with "next" as text using React testing library.
    const nextButton = screen.getByRole('button', {name: /next/i});

    // Next button is available on the first step.
    assert.isNull(nextButton.getAttribute('disabled'));

    // Click the next button twice to go to the third step.
    userEvent.click(nextButton);

    // Now the next button should be disabled.
    assert.isNotNull(nextButton.getAttribute('disabled'));
  });
});