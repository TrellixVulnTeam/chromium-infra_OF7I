// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import { ClusterFailure } from '../../shared_elements/failure_table';

export const getMockFailures = (): Array<ClusterFailure> => {
    return [{
        'realm': 'chromeos:test_runner',
        'testId': 'health.DiagnosticsRun.cpu_cache',
        'variant': [{
            'key': 'test_config',
            'value': 'volteer-manatee-cq.hw.bvt-tast-cq'
        }, {
            'key': 'board',
            'value': 'volteer'
        }, {
            'key': 'build_target',
            'value': 'volteer'
        }],
        'presubmitRunId': null,
        'presubmitRunOwner': '',
        'presubmitRunCl': null,
        'partitionTime': '2022-03-30T04:25:26.835039Z',
        'exonerationStatus': 'NOT_EXONERATED',
        'ingestedInvocationId': 'build-8818296299583177985',
        'isIngestedInvocationBlocked': true,
        'testRunIds': ['build-8818296299583177985'],
        'isTestRunBlocked': true,
        'count': 3
    }, {
        'realm': 'chromeos:test_runner',
        'testId': 'health.DiagnosticsRun.cpu_cache',
        'variant': [{
            'key': 'build_target',
            'value': 'volteer'
        }, {
            'key': 'test_config',
            'value': 'volteer-manatee-cq.hw.bvt-tast-cq'
        }, {
            'key': 'board',
            'value': 'volteer'
        }],
        'presubmitRunId': null,
        'presubmitRunOwner': '',
        'presubmitRunCl': null,
        'partitionTime': '2022-03-30T04:23:52.105273Z',
        'exonerationStatus': 'NOT_EXONERATED',
        'ingestedInvocationId': 'build-8818296398914149361',
        'isIngestedInvocationBlocked': true,
        'testRunIds': ['build-8818296398914149361'],
        'isTestRunBlocked': true,
        'count': 3
    }, {
        'realm': 'chromeos:testplatform',
        'testId': 'health.DiagnosticsRun.cpu_cache',
        'variant': [{
            'key': 'board',
            'value': 'volteer'
        }, {
            'key': 'build_target',
            'value': 'volteer'
        }, {
            'key': 'test_config',
            'value': 'volteer-manatee-cq.hw.bvt-tast-cq'
        }],
        'presubmitRunId': null,
        'presubmitRunOwner': '',
        'presubmitRunCl': null,
        'partitionTime': '2022-03-30T04:02:25.466308Z',
        'exonerationStatus': 'NOT_EXONERATED',
        'ingestedInvocationId': 'build-8818297748053790929',
        'isIngestedInvocationBlocked': true,
        'testRunIds': ['build-8818297302370459617'],
        'isTestRunBlocked': true,
        'count': 3
    }];
};