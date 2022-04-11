/* eslint-disable @typescript-eslint/no-non-null-assertion */
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
import { identityFunction } from '../../../testing_tools/functions';
import { renderWithClient } from '../../../testing_tools/libs/mock_rquery';
import { mockFetchAuthState } from '../../../testing_tools/mocks/authstate_mock';
import { createDefaultMockRule } from '../../../testing_tools/mocks/rule_mock';
import RuleEditDialog from './rule_edit_dialog';

describe('Test RuleEditDialog component', () => {

    afterEach(() => {
        fetchMock.mockClear();
        fetchMock.reset();
    });

    it('given a rule, should display the rule\'s current text', async () => {
        const mockRule = createDefaultMockRule();

        renderWithClient(
            <RuleEditDialog
                open
                rule={mockRule}
                setOpen={identityFunction}
            />
        );
        await screen.findByTestId('rule-input');

        expect(screen.getByText(mockRule.ruleDefinition)).toBeInTheDocument();
    });

    it('when modifying the rule\'s text, then should update the rule', async () => {
        const mockRule = createDefaultMockRule();
        mockFetchAuthState();

        renderWithClient(
            <RuleEditDialog
                open
                rule={mockRule}
                setOpen={identityFunction}
            />
        );

        await screen.findByTestId('rule-input');

        fireEvent.change(screen.getByTestId('rule-input'), { target: { value: 'new rule definition' } });

        const updatedRule: Rule = {
            ...mockRule,
            ruleDefinition: 'new rule definition'
        };
        fetchMock.post('http://localhost/prpc/weetbix.v1.Rules/Update', {
            headers: {
                'X-Prpc-Grpc-Code': '0'
            },
            body: ')]}\''+JSON.stringify(updatedRule)
        });

        fireEvent.click(screen.getByText('Save'));
        await waitFor(() => fetchMock.lastCall() !== undefined && fetchMock.lastCall()![0] === 'http://localhost/prpc/weetbix.v1.Rules/Update');

        expect(fetchMock.lastCall()![1]!.body).toEqual('{"rule":{"name":"projects/chromium/rules/ce83f8395178a0f2edad59fc1a167818",' +
        '"ruleDefinition":"new rule definition"},' +
        '"updateMask":"ruleDefinition","etag":"W/\\"2022-01-31T03:36:14.89643Z\\""' +
        '}');
    });

    it('when canceling the changes, then should revert', async () => {
        const mockRule = createDefaultMockRule();

        renderWithClient(
            <RuleEditDialog
                open
                rule={mockRule}
                setOpen={identityFunction}
            />
        );
        await screen.findByTestId('rule-input');

        fireEvent.change(screen.getByTestId('rule-input'), { target: { value: 'new rule definition' } });

        fireEvent.click(screen.getByText('Cancel'));

        expect(screen.getByTestId('rule-input')).toHaveValue('test = "blink_lint_expectations"');
    });
});