// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';

import React from 'react';

import {
    render,
    screen,
    fireEvent
} from '@testing-library/react';

import HelpTooltip from './help_tooltip';

describe('Test HelpTooltip component', () => {
    it('given a title, should display it', async () => {
        render(<HelpTooltip text="I can help you" />);

        await screen.findByRole('button');
        const button = screen.getByRole('button');
        fireEvent.mouseOver(button);
        await screen.findByText('I can help you');
        expect(screen.getByText('I can help you')).toBeInTheDocument();
    });
});