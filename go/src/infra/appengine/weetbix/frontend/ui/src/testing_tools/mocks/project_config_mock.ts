// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import fetchMock from 'fetch-mock-jest';

import { ProjectConfig } from '../../services/config';

export const createMockProjectConfig = (): ProjectConfig => {
    return {
        project: 'chromium',
        monorail: {
            project: 'chromium',
            displayPrefix: 'crbug.com'
        },
        paths: []
    };
};

export const mockFetchProjectConfig = () => {
    fetchMock.get('/api/projects/chromium/config', createMockProjectConfig());
};