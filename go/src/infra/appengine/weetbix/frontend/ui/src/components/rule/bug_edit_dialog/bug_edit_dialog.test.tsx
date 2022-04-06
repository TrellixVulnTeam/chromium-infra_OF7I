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
import { noopStateChanger } from '../../../testing_tools/functions';
import { renderWithRouterAndClient } from '../../../testing_tools/libs/mock_router';
import { mockFetchAuthState } from '../../../testing_tools/mocks/authstate_mock';
import { mockFetchProjectConfig } from '../../../testing_tools/mocks/project_config_mock';
import {
    createDefaultMockRule,
    mockFetchRule
} from '../../../testing_tools/mocks/rule_mock';
import BugEditDialog from './bug_edit_dialog';

describe('Test BugEditDialog component', () => {

    beforeEach(() => {
        mockFetchProjectConfig();
        mockFetchAuthState();
        mockFetchRule();
    });

    afterEach(() => {
        fetchMock.mockClear();
        fetchMock.reset();
    });

    it('given a bug, then should display details', async () => {
        renderWithRouterAndClient(
            <BugEditDialog
                open
                setOpen={noopStateChanger}
            />,
            '/p/chromium/rules/1234567',
            '/p/:project/rules/:id',
        );

        await screen.findByText('Save');

        expect(screen.getByText('Save')).toBeInTheDocument();
        expect(screen.getByText('Cancel')).toBeInTheDocument();
        expect(screen.getByText('Bug number')).toBeInTheDocument();
    });

    it('when cancelled, then should revert changes made', async () => {
        renderWithRouterAndClient(
            <BugEditDialog
                open
                setOpen={noopStateChanger}
            />,
            '/p/chromium/rules/1234567',
            '/p/:project/rules/:id',
        );

        await screen.findByText('Save');
        fireEvent.change(screen.getByTestId('bug-number'), { target: { value: '6789' } });

        expect(screen.getByTestId('bug-number')).toHaveValue('6789');

        fireEvent.click(screen.getByText('Cancel'));

        await waitFor(() => expect(screen.getByTestId('bug-number')).toHaveValue('920702'));
    });

    it('when changing bug details, then should update rule', async () => {
        renderWithRouterAndClient(
            <BugEditDialog
                open
                setOpen={noopStateChanger}
            />,
            '/p/chromium/rules/1234567',
            '/p/:project/rules/:id',
        );

        await screen.findByText('Save');
        fireEvent.change(screen.getByTestId('bug-number'), { target: { value: '6789' } });

        const updatedRule: Rule = {
            ...createDefaultMockRule(),
            bug: {
                id: 'chromium/6789',
                linkText: 'new-bug',
                system: 'monorail',
                url: 'http://linktobug',
            }
        };
        fetchMock.post('http://localhost/prpc/weetbix.v1.Rules/Update', {
            headers: {
                'X-Prpc-Grpc-Code': '0'
            },
            body: ')]}\'' + JSON.stringify(updatedRule)
        });
        fireEvent.click(screen.getByText('Save'));
        await waitFor(() => fetchMock.lastCall() !== undefined && fetchMock.lastCall()![0] === 'http://localhost/prpc/weetbix.v1.Rules/Update');
        expect(fetchMock.lastCall()![1]!.body).toEqual('{"rule":'+
        '{"name":"projects/chromium/rules/ce83f8395178a0f2edad59fc1a167818",'+
        '"bug":{"system":"monorail","id":"chromium/6789"'+
        '}},'+
        '"updateMask":"bug","etag":"W/\\"2022-01-31T03:36:14.89643Z\\""}');
        expect(screen.getByTestId('bug-number')).toHaveValue('6789');
    });
});