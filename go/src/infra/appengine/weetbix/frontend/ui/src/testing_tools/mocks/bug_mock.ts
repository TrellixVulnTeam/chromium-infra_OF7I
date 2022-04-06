// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import dayjs from 'dayjs';

import { Issue } from '../../services/monorail';

export const createMockBug = (): Issue => {
    return {
        name: 'bug for rule',
        summary: 'a bug for a rule',
        status: {
            status: 'accepted',
            derivation: '',
        },
        reporter: 'user@example.com',
        modifyTime: dayjs().toISOString(),
    };
};