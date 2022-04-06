// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';
import 'node-fetch';

import fetchMock from 'fetch-mock-jest';
import React from 'react';

import {
    fireEvent,
    screen
} from '@testing-library/react';

import { Issue } from '../../../services/monorail';
import { Rule } from '../../../services/rules';
import { renderWithRouterAndClient } from '../../../testing_tools/libs/mock_router';
import { mockFetchAuthState } from '../../../testing_tools/mocks/authstate_mock';
import { createMockBug } from '../../../testing_tools/mocks/bug_mock';
import { mockFetchProjectConfig } from '../../../testing_tools/mocks/project_config_mock';
import { createDefaultMockRule, mockFetchRule } from '../../../testing_tools/mocks/rule_mock';
import BugInfo from './bug_info';

describe('Test BugInfo component', () => {

    let mockRule!: Rule;
    let mockIssue!: Issue;

    beforeEach(() => {
        mockFetchAuthState();
        mockRule = createDefaultMockRule();
        mockIssue = createMockBug();
        fetchMock.post('https://api-dot-crbug.com/prpc/monorail.v3.Issues/GetIssue', {
            headers: {
                'X-Prpc-Grpc-Code': '0'
            },
            body: ')]}\'' + JSON.stringify(mockIssue)
        });
        mockFetchRule();
    });

    afterEach(() => {
        fetchMock.mockClear();
        fetchMock.reset();
    });

    it('given a rule, should fetch and display bug info', async () => {
        renderWithRouterAndClient(
            <BugInfo
                rule={mockRule}
            />
        );

        await screen.findByText('Bug details');

        expect(screen.getByText(mockRule.bug.linkText)).toBeInTheDocument();
        expect(screen.getByText(mockIssue.summary)).toBeInTheDocument();
        expect(screen.getByText(mockIssue.status.status)).toBeInTheDocument();
    });

    it('when clicking edit, should open dialog', async () => {
        mockFetchProjectConfig();
        renderWithRouterAndClient(
            <BugInfo
                rule={mockRule}
            />,
            '/p/chromium/rules/123456',
            '/p/:project/rules/:id'
        );

        await screen.findByText('Bug details');

        fireEvent.click(screen.getByLabelText('edit'));

        expect(screen.getByText('Edit bug')).toBeInTheDocument();
    });
});