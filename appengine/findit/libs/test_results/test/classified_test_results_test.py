# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from libs.test_results.classified_test_results import ClassifiedTestResults
from waterfall.test import wf_testcase


class ClassifiedTestResultsTest(wf_testcase.WaterfallTestCase):

  def testClassifiedTestResults(self):
    test_result_dict = {
        'Unittest1.Subtest1': {
            'total_run': 2,
            'num_expected_results': 1,
            'num_unexpected_results': 1,
            'results': {
                'passes': {
                    'SUCCESS': 1
                },
                'failures': {},
                'skips': {},
                'unknowns': {
                    'UNKNOWN': 1
                }
            }
        },
        'Unittest1.Subtest2': {
            'total_run': 3,
            'num_expected_results': 1,
            'num_unexpected_results': 2,
            'results': {
                'passes': {
                    'SUCCESS': 1
                },
                'failures': {
                    'FAILURE': 2
                },
                'skips': {},
                'unknowns': {}
            }
        },
        'Unittest2.Subtest1': {
            'total_run': 4,
            'num_expected_results': 1,
            'num_unexpected_results': 3,
            'results': {
                'passes': {
                    'SUCCESS': 1,
                },
                'failures': {
                    'FAILURE': 2
                },
                'skips': {
                    'SKIPPED': 1
                },
                'unknowns': {}
            }
        }
    }
    test_result_object = ClassifiedTestResults.FromDict(test_result_dict)
    self.assertEqual(test_result_dict, test_result_object.ToDict())
