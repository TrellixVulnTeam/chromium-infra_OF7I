#!/usr/bin/env vpython
# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import ntpath
import os
import sys
import unittest

import logging
import mock

from services.code_coverage import summary_coverage_aggregator
from waterfall.test.wf_testcase import WaterfallTestCase


class SummaryCoverageAggregatorTest(WaterfallTestCase):

  def testBasic(self):
    file_coverage_data = {
        'path': '//dir/file.cc',
        'lines': [{
            'first': 1,
            'last': 1,
            'count': 1,
        }],
        'summaries': [{
            'name': 'line',
            'covered': 1,
            'total': 1,
        }],
    }

    aggregator = summary_coverage_aggregator.SummaryCoverageAggregator(
        metrics=['line'])
    aggregator.consume_file_coverage(file_coverage_data)
    per_directory_data = aggregator.produce_summary_coverage()

    expected_per_directory_data = {
        '//': {
            'dirs': [{
                'name': 'dir/',
                'path': '//dir/',
                'summaries': [{
                    'name': 'line',
                    'covered': 1,
                    'total': 1,
                }]
            }],
            'files': [],
            'path': '//',
            'summaries': [{
                'name': 'line',
                'covered': 1,
                'total': 1,
            }]
        },
        '//dir/': {
            'dirs': [],
            'files': [{
                'name': 'file.cc',
                'path': '//dir/file.cc',
                'summaries': [{
                    'name': 'line',
                    'covered': 1,
                    'total': 1,
                }]
            }],
            'path': '//dir/',
            'summaries': [{
                'name': 'line',
                'covered': 1,
                'total': 1,
            }]
        }
    }

    self.assertDictEqual(expected_per_directory_data, per_directory_data)

  def testAvoidComponentDoubleCounting(self):
    files_coverage_data = [
        {
            'path': '//dir/file1.cc',
            'lines': [{
                'first': 1,
                'last': 1,
                'count': 1,
            }],
            'summaries': [{
                'name': 'line',
                'covered': 1,
                'total': 1,
            }],
        },
        {
            'path': '//dir/subdir/file2.cc',
            'lines': [{
                'first': 1,
                'last': 2,
                'count': 1,
            }],
            'summaries': [{
                'name': 'line',
                'covered': 2,
                'total': 2,
            }],
        },
    ]

    aggregator = summary_coverage_aggregator.SummaryCoverageAggregator(
        metrics=['line'])
    aggregator.consume_file_coverage(files_coverage_data[0])
    aggregator.consume_file_coverage(files_coverage_data[1])
    per_directory_data = aggregator.produce_summary_coverage()
    logging.info(per_directory_data)

    expected_per_directory_data = {
        '//': {
            'dirs': [{
                'name': 'dir/',
                'path': '//dir/',
                'summaries': [{
                    'name': 'line',
                    'covered': 3,
                    'total': 3
                }]
            }],
            'files': [],
            'path': '//',
            'summaries': [{
                'name': 'line',
                'covered': 3,
                'total': 3
            }]
        },
        '//dir/': {
            'dirs': [{
                'name': 'subdir/',
                'path': '//dir/subdir/',
                'summaries': [{
                    'name': 'line',
                    'covered': 2,
                    'total': 2
                }]
            }],
            'files': [{
                'name': 'file1.cc',
                'path': '//dir/file1.cc',
                'summaries': [{
                    'name': 'line',
                    'covered': 1,
                    'total': 1
                }]
            }],
            'path': '//dir/',
            'summaries': [{
                'name': 'line',
                'covered': 3,
                'total': 3
            }]
        },
        '//dir/subdir/': {
            'dirs': [],
            'files': [{
                'name': 'file2.cc',
                'path': '//dir/subdir/file2.cc',
                'summaries': [{
                    'name': 'line',
                    'covered': 2,
                    'total': 2
                }]
            }],
            'path': '//dir/subdir/',
            'summaries': [{
                'name': 'line',
                'covered': 2,
                'total': 2
            }]
        }
    }

    self.assertDictEqual(expected_per_directory_data, per_directory_data)
