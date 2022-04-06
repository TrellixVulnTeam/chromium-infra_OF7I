// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';

import React from 'react';

import {
    render,
    screen
} from '@testing-library/react';

import GridLabel from './grid_label';

describe('Test GridLabel component', () => {
    it('given text only, then should display it', async () => {
        render(
            <GridLabel
                text="Test text"
            />
        );
        await screen.findByText('Test text');
        expect(screen.getByText('Test text')).toBeInTheDocument();
    });

    it('given text and children, then should display them', async () => {
        render(
            <GridLabel
                text="Test text">
                <p>I am a child</p>
            </GridLabel>
        );
        await screen.findByText('Test text');
        expect(screen.getByText('Test text')).toBeInTheDocument();
        expect(screen.getByText('I am a child')).toBeInTheDocument();
    });
});