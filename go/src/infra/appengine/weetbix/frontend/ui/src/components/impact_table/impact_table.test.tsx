// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import '@testing-library/jest-dom';

import React from 'react';

import {
    render,
    screen
} from '@testing-library/react';

import ImpactTable from './impact_table';
import { getMockCluster } from '../../testing_tools/mocks/cluster_mock';

describe('Test ImpactTable component', () => {
    it('given a cluster, should display it', async () => {
        const cluster = getMockCluster('1234567890abcdef1234567890abcdef');
        render(<ImpactTable cluster={cluster} />);

        await screen.findByText('User Cls Failed Presubmit');
        // Check for 7d unexpected failures total.
        expect(screen.getByText('15810')).toBeInTheDocument();
    });
});