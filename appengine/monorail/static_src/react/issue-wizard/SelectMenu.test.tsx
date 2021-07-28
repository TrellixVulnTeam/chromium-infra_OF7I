// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {render} from '@testing-library/react';
import userEvent from '@testing-library/user-event'
import {assert} from 'chai';

import SelectMenu from './SelectMenu.tsx';

describe.only('SelectMenu', () => {
  it('renders', async () => {
    const {container} = render(<SelectMenu />);

    const form = container.querySelector('form');
    assert.isNotNull(form)
  });

  it('renders options on click', async () => {
    const {container} = render(<SelectMenu />);

    const input = document.getElementById("outlined-select-category")
    if (!input) {
      throw new Error('Input is undefined');
    }

    userEvent.click(input)

    // 14 is the current number of options in the select menu
    const count = document.querySelectorAll('li').length;
    assert.equal(count, 14)
  });
});