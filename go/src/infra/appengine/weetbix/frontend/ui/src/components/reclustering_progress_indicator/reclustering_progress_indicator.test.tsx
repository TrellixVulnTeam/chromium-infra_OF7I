/* eslint-disable @typescript-eslint/no-empty-function */
// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';
import 'node-fetch';

import dayjs from 'dayjs';
import fetchMock from 'fetch-mock-jest';
import React from 'react';

import {
    screen,
    waitFor
} from '@testing-library/react';

import { renderWithClient } from '../../testing_tools/libs/mock_rquery';
import {
    createMockDoneProgress,
    createMockProgress
} from '../../testing_tools/mocks/progress_mock';
import ReclusteringProgressIndicator from './reclustering_progress_indicator';

describe('Test ReclusteringProgressIndicator component', () => {

    afterEach(() => {
        fetchMock.mockClear();
        fetchMock.reset();
    });

    it('given an finished progress, then should not display', async () => {
        fetchMock.get('/api/projects/chromium/reclusteringProgress', createMockDoneProgress());
        renderWithClient(
            <ReclusteringProgressIndicator
                project='chromium'
                hasRule
                rulePredicateLastUpdated={dayjs().subtract(5, 'minutes').toISOString()}
            />
        );

        expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    });

    it('given a progress, then should display percentage', async () => {
        fetchMock.get('/api/projects/chromium/reclusteringProgress', createMockProgress(800));
        renderWithClient(
            <ReclusteringProgressIndicator
                project='chromium'
                hasRule
                rulePredicateLastUpdated={dayjs().subtract(5, 'minutes').toISOString()}
            />
        );

        await screen.findByRole('alert');
        await screen.findByText('80%');

        expect(screen.getByText('80%')).toBeInTheDocument();
    });

    it('when progress is done after being on screen, then should display button to refresh analysis', async () => {
        fetchMock.getOnce('/api/projects/chromium/reclusteringProgress', createMockProgress(800));
        renderWithClient(
            <ReclusteringProgressIndicator
                project='chromium'
                hasRule
                rulePredicateLastUpdated={dayjs().subtract(5, 'minutes').toISOString()}
            />
        );
        await screen.findByRole('alert');
        await screen.findByText('80%');

        fetchMock.getOnce('/api/projects/chromium/reclusteringProgress', createMockDoneProgress(), {
            overwriteRoutes: false
        });

        await waitFor(() => fetchMock.calls.length == 2);

        await screen.findByRole('button');
        expect(screen.getByText('View updated impact')).toBeInTheDocument();
    });
});