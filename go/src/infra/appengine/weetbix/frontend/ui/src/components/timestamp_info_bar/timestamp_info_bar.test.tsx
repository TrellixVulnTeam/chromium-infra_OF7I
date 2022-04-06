// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';

import React from 'react';

import {
    render,
    screen,
    waitFor
} from '@testing-library/react';

import { createDefaultMockRule } from '../../testing_tools/mocks/rule_mock';
import TimestampInfoBar from './timestamp_info_bar';

describe('Test TimestampInfoBar component', () => {
    it('when proved with rule, then should render username and timestamps', async () => {
        const rule = createDefaultMockRule();
        render(<TimestampInfoBar
            createUsername={rule.createUser}
            createTime={rule.createTime}
            updateUsername={rule.lastUpdateUser}
            updateTime={rule.lastUpdateTime}
        />);
        await waitFor(() => screen.getByTestId('timestamp-info-bar-create'));

        expect(screen.getByTestId('timestamp-info-bar-create'))
            .toHaveTextContent('Created by Weetbix a few seconds ago');
        expect(screen.getByTestId('timestamp-info-bar-update'))
            .toHaveTextContent('Last modified by user@example.com a few seconds ago');
    });

    it('when provided with a google account, then should display name only', async () => {
        const rule = createDefaultMockRule();
        rule.createUser = 'googler@google.com';
        render(<TimestampInfoBar
            createUsername={rule.createUser}
            createTime={rule.createTime}
            updateUsername={rule.lastUpdateUser}
            updateTime={rule.lastUpdateTime}
        />);
        await waitFor(() => screen.getByTestId('timestamp-info-bar-create'));

        expect(screen.getByTestId('timestamp-info-bar-create'))
            .toHaveTextContent('Created by googler a few seconds ago');
        expect(screen.getByTestId('timestamp-info-bar-update'))
            .toHaveTextContent('Last modified by user@example.com a few seconds ago');
    });

    it('when provided with an external user, then should use full username', async () => {
        const rule = createDefaultMockRule();
        rule.createUser = 'user@example.com';
        render(<TimestampInfoBar
            createUsername={rule.createUser}
            createTime={rule.createTime}
            updateUsername={rule.lastUpdateUser}
            updateTime={rule.lastUpdateTime}
        />);
        await waitFor(() => screen.getByTestId('timestamp-info-bar-create'));

        expect(screen.getByTestId('timestamp-info-bar-create'))
            .toHaveTextContent('Created by user@example.com a few seconds ago');
        expect(screen.getByTestId('timestamp-info-bar-update'))
            .toHaveTextContent('Last modified by user@example.com a few seconds ago');
    });
});