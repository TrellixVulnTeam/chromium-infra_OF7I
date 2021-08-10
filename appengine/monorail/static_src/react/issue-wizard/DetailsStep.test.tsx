// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import {render} from '@testing-library/react';
import {assert} from 'chai';

import DetailsStep from './DetailsStep.tsx';

describe('DetailsStep', () => {
  it('renders', async () => {
    const {container} = render(<DetailsStep />);

    // this is checking for the first question
    const input = container.querySelector('input');
    assert.isNotNull(input)

    // this is checking for the rest
    const count = document.querySelectorAll('textarea').length;
    assert.equal(count, 3)
  });
});