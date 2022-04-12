// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

export const getMockCluster = (id: string) => {
    return {
        'clusterId': {
            'algorithm': 'rules-v1',
            'id': id
        },
        'presubmitRejects1d': {
            'nominal': 97,
            'preWeetbix': 97,
            'preExoneration': 98,
            'residual': 97,
            'residualPreWeetbix': 97,
            'residualPreExoneration': 98
        },
        'presubmitRejects3d': {
            'nominal': 157,
            'preWeetbix': 157,
            'preExoneration': 158,
            'residual': 157,
            'residualPreWeetbix': 157,
            'residualPreExoneration': 158
        },
        'presubmitRejects7d': {
            'nominal': 163,
            'preWeetbix': 163,
            'preExoneration': 167,
            'residual': 163,
            'residualPreWeetbix': 163,
            'residualPreExoneration': 167
        },
        'testRunFailures1d': {
            'nominal': 2425,
            'preWeetbix': 2425,
            'preExoneration': 2527,
            'residual': 2425,
            'residualPreWeetbix': 2425,
            'residualPreExoneration': 2527
        },
        'testRunFailures3d': {
            'nominal': 4494,
            'preWeetbix': 4494,
            'preExoneration': 4716,
            'residual': 4494,
            'residualPreWeetbix': 4494,
            'residualPreExoneration': 4716
        },
        'testRunFailures7d': {
            'nominal': 4662,
            'preWeetbix': 4662,
            'preExoneration': 4938,
            'residual': 4662,
            'residualPreWeetbix': 4662,
            'residualPreExoneration': 4938
        },
        'failures1d': {
            'nominal': 7319,
            'preWeetbix': 7319,
            'preExoneration': 7625,
            'residual': 7319,
            'residualPreWeetbix': 7319,
            'residualPreExoneration': 7625
        },
        'failures3d': {
            'nominal': 15221,
            'preWeetbix': 15221,
            'preExoneration': 16052,
            'residual': 15221,
            'residualPreWeetbix': 15231,
            'residualPreExoneration': 16052
        },
        'failures7d': {
            'nominal': 15800,
            'preWeetbix': 15810,
            'preExoneration': 16792,
            'residual': 15780,
            'residualPreWeetbix': 15790,
            'residualPreExoneration': 16792
        },
        'title': '',
        'failureAssociationRule': ''
    };
};