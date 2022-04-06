// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';

import React from 'react';

import {
    render,
    screen
} from '@testing-library/react';

import {
    Snack,
    SnackbarContext
} from '../../context/snackbar_context';
import { identityFunction } from '../../testing_tools/functions';
import FeedbackSnackbar from './feedback_snackbar';

describe('Test ErrorSnackbar component', () => {
    it('given an error text, should display in a snackbar', async () => {
        const snack: Snack = {
            open: true,
            message: 'Failed to load issue',
            severity: 'error'
        };
        render(
            <SnackbarContext.Provider value={{
                snack: snack,
                setSnack: identityFunction
            }}>
                <FeedbackSnackbar />
            </SnackbarContext.Provider>
        );

        await screen.findByTestId('snackbar');

        expect(screen.getByText('Failed to load issue')).toBeInTheDocument();
    });
});