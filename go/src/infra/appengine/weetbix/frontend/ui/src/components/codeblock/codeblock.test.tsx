// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import React from 'react';
import '@testing-library/jest-dom';

import {
    render,
    screen,
    waitFor
} from '@testing-library/react';

import CodeBlock from './codeblock';

describe('Test CodeBlock component', () => {
    it('given a piece of code, then should display it in a block', async () => {
        render(<CodeBlock code='int x = 0'/>);
        await waitFor(() => screen.getByTestId('codeblock'));
        expect(screen.getByTestId('codeblock')).toHaveTextContent('int x = 0');
    });
});