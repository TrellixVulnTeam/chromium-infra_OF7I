// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';

import React from 'react';

import {
    render,
    screen
} from '@testing-library/react';

import { identityFunction } from '../../testing_tools/functions';
import ConfirmDialog from './confirm_dialog';

describe('Test ConfirmDialog component', () => {
    it('Given no message, should display title only', async () => {
        render(<ConfirmDialog
            onCancel={identityFunction}
            onConfirm={identityFunction}
            open
        />);
        await screen.findByText('Are you sure?');
        expect(screen.getByText('Are you sure?')).toBeInTheDocument();
    });

    it('Given a message, then should display it', async () => {
        render(<ConfirmDialog
            onCancel={identityFunction}
            onConfirm={identityFunction}
            message="Test message"
            open
        />);
        await screen.findByText('Test message');
        expect(screen.getByText('Test message')).toBeInTheDocument();
    });
});