// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';
import 'node-fetch';

import fetchMock from 'fetch-mock-jest';
import React from 'react';

import { screen } from '@testing-library/react';

import { renderWithRouterAndClient } from '../../../testing_tools/libs/mock_router';
import { mockFetchAuthState } from '../../../testing_tools/mocks/authstate_mock';
import { createMockBug } from '../../../testing_tools/mocks/bug_mock';
import { mockFetchProjectConfig } from '../../../testing_tools/mocks/project_config_mock';
import { createDefaultMockRule } from '../../../testing_tools/mocks/rule_mock';
import RuleTopPanel from './rule_top_panel';

describe('Test RuleTopPanel component', () => {
    it('given a rule, should display rule and bug details', async () => {
        mockFetchProjectConfig();
        mockFetchAuthState();
        const mockRule = createDefaultMockRule();
        fetchMock.post('https://api-dot-crbug.com/prpc/monorail.v3.Issues/GetIssue', {
            headers: {
                'X-Prpc-Grpc-Code': '0'
            },
            body: ')]}\'' + JSON.stringify(createMockBug())
        });
        fetchMock.post('http://localhost/prpc/weetbix.v1.Rules/Get', {
            headers: {
                'X-Prpc-Grpc-Code': '0'
            },
            body: ')]}\''+JSON.stringify(mockRule)
        });

        renderWithRouterAndClient(
            <RuleTopPanel 
                project="chromium"
                ruleId='12345'
            />,
            '/p/chromium/rules/12345',
            '/p/:project/rules/:id'
        );
        await screen.findByText('Bug details');

        expect(screen.getByText('Details')).toBeInTheDocument();
        expect(screen.getByText('Bug details')).toBeInTheDocument();
    });
});