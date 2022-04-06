// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';

import React from 'react';

import {
    render,
    screen
} from '@testing-library/react';

import ErrorAlert from './error_alert';

describe('Test ErrorAlert component', () => {
    it('given a title and text, then should display them.', async () => {
        render(<ErrorAlert
            errorTitle="Test error title"
            errorText="Test error text"
            showError={true}
        />);
        await screen.findByText('Test error title');
        expect(screen.getByText('Test error title')).toBeInTheDocument();
        expect(screen.getByText('Test error text')).toBeInTheDocument();
    });
});