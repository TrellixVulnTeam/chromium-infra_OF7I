// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';
import 'node-fetch';

import fetchMock from 'fetch-mock-jest';
import React from 'react';

import {
    fireEvent,
    screen,
    waitFor
} from '@testing-library/react';

import { Rule } from '../../../services/rules';
import { renderWithRouterAndClient } from '../../../testing_tools/libs/mock_router';
import { mockFetchAuthState } from '../../../testing_tools/mocks/authstate_mock';
import { createDefaultMockRule } from '../../../testing_tools/mocks/rule_mock';
import RuleInfo from './rule_info';

describe('Test RuleInfo component', () => {
    it('given a rule, then should display rule details', async () => {
        const mockRule = createDefaultMockRule();
        renderWithRouterAndClient(
            <RuleInfo
                project="chromium"
                rule={mockRule}
            />
        );

        await screen.findByText('Details');

        expect(screen.getByText(mockRule.ruleDefinition)).toBeInTheDocument();
        expect(screen.getByText(`${mockRule.sourceCluster.algorithm}/${mockRule.sourceCluster.id}`)).toBeInTheDocument();
        expect(screen.getByText('Archived')).toBeInTheDocument();
        expect(screen.getByText('No')).toBeInTheDocument();
    });

    it('when clicking on archived, then should show confirmation dialog', async () => {
        const mockRule = createDefaultMockRule();

        renderWithRouterAndClient(
            <RuleInfo
                project="chromium"
                rule={mockRule}
            />
        );
        await screen.findByText('Details');

        fireEvent.click(screen.getByText('Archive'));
        await screen.findByText('Are you sure?');

        expect(screen.getByText('Are you sure you want to archive this rule?')).toBeInTheDocument();
    });

    it('when confirming the archival, then should send archival request', async () => {
        mockFetchAuthState();
        const mockRule = createDefaultMockRule();
        renderWithRouterAndClient(
            <RuleInfo
                project="chromium"
                rule={mockRule}
            />
        );
        await screen.findByText('Details');

        fireEvent.click(screen.getByText('Archive'));
        await screen.findByText('Are you sure?');

        expect(screen.getByText('Are you sure you want to archive this rule?')).toBeInTheDocument();

        const updatedRule: Rule = {
            ...mockRule,
            isActive: false,
        };
        fetchMock.post('http://localhost/prpc/weetbix.v1.Rules/Update', {
            headers: {
                'X-Prpc-Grpc-Code': '0'
            },
            body: ')]}\''+JSON.stringify(updatedRule)
        });

        fireEvent.click(screen.getByText('Save'));
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        await waitFor(() => fetchMock.lastCall() !== undefined && fetchMock.lastCall()![0] === 'http://localhost/prpc/weetbix.v1.Rules/Update');

        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        expect(fetchMock.lastCall()![1]!.body).toEqual('{"rule":{"name":"projects/chromium/rules/ce83f8395178a0f2edad59fc1a167818",' +
        '"isActive":false},' +
        '"updateMask":"isActive","etag":"W/\\"2022-01-31T03:36:14.89643Z\\""' +
        '}');
    });
});