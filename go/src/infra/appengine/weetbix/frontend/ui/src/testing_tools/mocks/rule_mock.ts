// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import dayjs from 'dayjs';
import fetchMock from 'fetch-mock-jest';

import { Rule } from '../../services/rules';

export const createDefaultMockRule = (): Rule => {
    return {
        name: 'projects/chromium/rules/ce83f8395178a0f2edad59fc1a167818',
        project: 'chromium',
        ruleId: 'ce83f8395178a0f2edad59fc1a167818',
        ruleDefinition: 'test = "blink_lint_expectations"',
        bug: {
            system: 'monorail',
            id: 'chromium/920702',
            linkText: 'crbug.com/920702',
            url: 'https://monorail-staging.appspot.com/p/chromium/issues/detail?id=920702'
        },
        isActive: true,
        isManagingBug: true,
        sourceCluster: {
            algorithm: 'testname-v3',
            id: '78ff0812026b30570ca730b1541125ea'
        },
        createTime: dayjs().toISOString(),
        createUser: 'weetbix',
        lastUpdateTime: dayjs().toISOString(),
        lastUpdateUser: 'user@example.com',
        predicateLastUpdateTime: '2022-01-31T03:36:14.896430Z',
        etag: 'W/"2022-01-31T03:36:14.89643Z"'
    };
};

export const mockFetchRule = () => {
    fetchMock.post('http://localhost/prpc/weetbix.v1.Rules/Get', {
        headers: {
            'X-Prpc-Grpc-Code': '0'
        },
        body: ')]}\'' + JSON.stringify(createDefaultMockRule())
    });
};