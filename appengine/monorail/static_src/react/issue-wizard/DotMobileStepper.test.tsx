// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {render, screen, cleanup} from '@testing-library/react';
import {assert} from 'chai';

import DotMobileStepper from './DotMobileStepper.tsx';

describe('DotMobileStepper', () => {
  let container: HTMLElement;

  afterEach(cleanup);

  it('renders', () => {
    container = render(<DotMobileStepper activeStep={0} nextEnabled={true}/>).container;

    // this is checking the buttons for the stepper rendered
      const count = document.querySelectorAll('button').length;
      assert.equal(count, 1)
  });

  it('back button not avlialbe on first step', () => {
    render(<DotMobileStepper activeStep={0} nextEnabled={true}/>).container;

    // Finds a button on the page with "back" as text using React testing library.
    const backButton = document.querySelector('[aria-label="backButton"]');

    // Back button is not avliable on the first step.
    assert.notExists(backButton);
  });

  it('both buttons enabled on second step', () => {
    render(<DotMobileStepper activeStep={1} nextEnabled={true}/>).container;

    // Finds a button on the page with "back" as text using React testing library.
    const backButton = screen.getByRole('button', {name: /backButton/i}) as HTMLButtonElement;

    // Finds a button on the page with "next" as text using React testing library.
    const nextButton = screen.getByRole('button', {name: /nextButton/i}) as HTMLButtonElement;

    // Back button is not disabled on the second step.
    assert.isFalse(backButton.disabled);

    // Next button is not disabled on the second step.
    assert.isFalse(nextButton.disabled);
  });
});
