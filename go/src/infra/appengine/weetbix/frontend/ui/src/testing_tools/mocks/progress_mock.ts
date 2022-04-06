// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import dayjs from 'dayjs';

import { ReclusteringProgress } from '../../services/progress';

export const createMockProgress = (progress: number): ReclusteringProgress => {
    return {
        progressPerMille: progress,
        latestAlgorithmsVersion: 7,
        latestConfigVersion: dayjs().toISOString(),
        next: {
            rulesVersion: dayjs().toISOString(),
            configVersion: dayjs().toISOString(),
            algorithmsVersion: 7
        },
        last: {
            rulesVersion: dayjs().subtract(2, 'minutes').toISOString(),
            configVersion: dayjs().subtract(2, 'minutes').toISOString(),
            algorithmsVersion: 6
        }
    };
};

export const createMockDoneProgress = (): ReclusteringProgress => {
    const currentDate = dayjs();
    return {
        progressPerMille: 1000,
        latestAlgorithmsVersion: 7,
        latestConfigVersion: currentDate.toISOString(),
        next: {
            rulesVersion: currentDate.toISOString(),
            configVersion: currentDate.toISOString(),
            algorithmsVersion: 7
        },
        last: {
            rulesVersion: currentDate.toISOString(),
            configVersion: currentDate.toISOString(),
            algorithmsVersion: 7
        }
    };
};