# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json

import endpoints
from google.appengine.api import taskqueue

from testing_utils import testing

from common.waterfall import failure_type
import findit_api
from findit_api import FindItApi
from model.wf_analysis import WfAnalysis
from model.wf_try_job import WfTryJob
from model import analysis_status
from waterfall import waterfall_config


class FinditApiTest(testing.EndpointsTestCase):
  api_service_cls = FindItApi

  def setUp(self):
    super(FinditApiTest, self).setUp()
    self.taskqueue_requests = []
    def Mocked_taskqueue_add(**kwargs):
      self.taskqueue_requests.append(kwargs)
    self.mock(taskqueue, 'add', Mocked_taskqueue_add)

  def _MockMasterIsSupported(self, supported):
    def MockMasterIsSupported(*_):
      return supported
    self.mock(waterfall_config, 'MasterIsSupported',
              MockMasterIsSupported)

  def testUnrecognizedMasterUrl(self):
    builds = {
        'builds': [
            {
                'master_url': 'https://not a master url',
                'builder_name': 'a',
                'build_number': 1
            }
        ]
    }
    expected_results = []

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_results, response.json_body.get('results', []))

  def testMasterIsNotSupported(self):
    builds = {
        'builds': [
            {
                'master_url': 'https://build.chromium.org/p/a',
                'builder_name': 'a',
                'build_number': 1
            }
        ]
    }
    expected_results = []

    self._MockMasterIsSupported(supported=False)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_results, response.json_body.get('results', []))

  def testNothingIsReturnedWhenNoAnalysisWasRun(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 5

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number
            }
        ]
    }

    expected_result = []

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_result, response.json_body.get('results', []))

  def testFailedAnalysisIsNotReturnedEvenWhenItHasResults(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 5

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number
            }
        ]
    }

    analysis = WfAnalysis.Create(master_name, builder_name, build_number)
    analysis.status = analysis_status.ERROR
    analysis.result = {
        'failures': [
            {
                'step_name': 'test',
                'first_failure': 3,
                'last_pass': 1,
                'suspected_cls': [
                    {
                        'repo_name': 'chromium',
                        'revision': 'git_hash',
                        'commit_position': 123,
                    }
                ]
            }
        ]
    }
    analysis.put()

    expected_result = []

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_result, response.json_body.get('results', []))

  def testNoResultIsReturnedWhenNoAnalysisIsCompleted(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 5

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number
            }
        ]
    }

    analysis = WfAnalysis.Create(master_name, builder_name, build_number)
    analysis.status = analysis_status.RUNNING
    analysis.result = None
    analysis.put()

    expected_result = []

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_result, response.json_body.get('results', []))

  def testPreviousAnalysisResultIsReturnedWhileANewAnalysisIsRunning(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 1

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number,
                'failed_steps': ['a', 'b']
            }
        ]
    }

    self._MockMasterIsSupported(supported=True)

    analysis_result = {
        'failures': [
            {
                'step_name': 'a',
                'first_failure': 23,
                'last_pass': 22,
                'suspected_cls': [
                    {
                        'repo_name': 'chromium',
                        'revision': 'git_hash',
                        'commit_position': 123,
                    }
                ]
            }
        ]
    }
    expected_results = [
        {
            'master_url': master_url,
            'builder_name': builder_name,
            'build_number': build_number,
            'step_name': 'a',
            'is_sub_test': False,
            'first_known_failed_build_number': 23,
            'suspected_cls': [
                {
                    'repo_name': 'chromium',
                    'revision': 'git_hash',
                    'commit_position': 123,
                }
            ],
            'analysis_approach': 'HEURISTIC',
        },
    ]

    analysis = WfAnalysis.Create(master_name, builder_name, build_number)
    analysis.status = analysis_status.RUNNING
    analysis.result = analysis_result
    analysis.put()

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_results, response.json_body['results'])

  def testAnalysisFindingNoSuspectedCLsIsNotReturned(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 5

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number
            }
        ]
    }

    analysis = WfAnalysis.Create(master_name, builder_name, build_number)
    analysis.status = analysis_status.COMPLETED
    analysis.result = {
        'failures': [
            {
                'step_name': 'test',
                'first_failure': 3,
                'last_pass': 1,
                'suspected_cls': []
            }
        ]
    }
    analysis.put()

    expected_result = []

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_result, response.json_body.get('results', []))

  def testAnalysisFindingSuspectedCLsIsReturned(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 5

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number
            }
        ]
    }

    analysis = WfAnalysis.Create(master_name, builder_name, build_number)
    analysis.status = analysis_status.COMPLETED
    analysis.result = {
        'failures': [
            {
                'step_name': 'test',
                'first_failure': 3,
                'last_pass': 1,
                'suspected_cls': [
                    {
                        'build_number': 2,
                        'repo_name': 'chromium',
                        'revision': 'git_hash1',
                        'commit_position': 234,
                        'score': 11,
                        'hints': {
                            'add a/b/x.cc': 5,
                            'delete a/b/y.cc': 5,
                            'modify e/f/z.cc': 1,
                        }
                    },
                    {
                        'build_number': 3,
                        'repo_name': 'chromium',
                        'revision': 'git_hash2',
                        'commit_position': 288,
                        'score': 1,
                        'hints': {
                            'modify d/e/f.cc': 1,
                        }
                    }
                ]
            }
        ]
    }
    analysis.put()

    expected_results = [
        {
            'master_url': master_url,
            'builder_name': builder_name,
            'build_number': build_number,
            'step_name': 'test',
            'is_sub_test': False,
            'first_known_failed_build_number': 3,
            'suspected_cls': [
                {
                    'repo_name': 'chromium',
                    'revision': 'git_hash1',
                    'commit_position': 234,
                },
                {
                    'repo_name': 'chromium',
                    'revision': 'git_hash2',
                    'commit_position': 288,
                }
            ],
            'analysis_approach': 'HEURISTIC',
        }
    ]

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_results, response.json_body.get('results'))

  def testTryJobResultReturnedForCompileFailure(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 5

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number
            }
        ]
    }

    try_job = WfTryJob.Create(master_name, builder_name, 3)
    try_job.status = analysis_status.COMPLETED
    try_job.compile_results = [
        {
            'culprit': {
                'compile': {
                      'repo_name': 'chromium',
                      'revision': 'r3',
                      'commit_position': 3,
                      'url': None,
                },
            },
        }
    ]
    try_job.put()

    analysis = WfAnalysis.Create(master_name, builder_name, build_number)
    analysis.status = analysis_status.COMPLETED
    analysis.build_failure_type = failure_type.COMPILE
    analysis.failure_result_map = {
        'compile': '/'.join([master_name, builder_name, '3']),
    }
    analysis.result = {
        'failures': [
            {
                'step_name': 'compile',
                'first_failure': 3,
                'last_pass': 1,
                'suspected_cls': [
                    {
                        'build_number': 3,
                        'repo_name': 'chromium',
                        'revision': 'git_hash2',
                        'commit_position': 288,
                        'score': 1,
                        'hints': {
                            'modify d/e/f.cc': 1,
                        }
                    }
                ]
            }
        ]
    }
    analysis.put()

    expected_results = [
        {
            'master_url': master_url,
            'builder_name': builder_name,
            'build_number': build_number,
            'step_name': 'compile',
            'is_sub_test': False,
            'first_known_failed_build_number': 3,
            'suspected_cls': [
                {
                    'repo_name': 'chromium',
                    'revision': 'r3',
                    'commit_position': 3,
                },
            ],
            'analysis_approach': 'TRY_JOB',
        }
    ]

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_results, response.json_body.get('results'))

  def testTestLevelResultIsReturned(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 5

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number
            }
        ]
    }

    try_job = WfTryJob.Create(master_name, builder_name, 4)
    try_job.status = analysis_status.COMPLETED
    try_job.test_results = [
        {
            'culprit': {
                'a': {
                      'repo_name': 'chromium',
                      'revision': 'r4_2',
                      'commit_position': 42,
                      'url': None,
                },
                'b': {
                    'tests': {
                        'Unittest3.Subtest1': {
                            'repo_name': 'chromium',
                            'revision': 'r4_10',
                            'commit_position': 410,
                            'url': None,
                        },
                    }
                }
            },
        }
    ]
    try_job.put()

    analysis = WfAnalysis.Create(master_name, builder_name, build_number)
    analysis.status = analysis_status.COMPLETED
    analysis.failure_result_map = {
        'a': '/'.join([master_name, builder_name, '4']),
        'b': {
            'Unittest1.Subtest1': '/'.join([master_name, builder_name, '3']),
            'Unittest3.Subtest1': '/'.join([master_name, builder_name, '4']),
        },
    }
    analysis.result = {
        'failures': [
            {
                'step_name': 'a',
                'first_failure': 4,
                'last_pass': 3,
                'suspected_cls': [
                    {
                        'build_number': 4,
                        'repo_name': 'chromium',
                        'revision': 'r4_2_failed',
                        'commit_position': None,
                        'url': None,
                        'score': 2,
                        'hints': {
                            'modified f4_2.cc (and it was in log)': 2,
                        },
                    }
                ],
            },
            {
                'step_name': 'b',
                'first_failure': 3,
                'last_pass': 2,
                'suspected_cls': [
                    {
                        'build_number': 3,
                        'repo_name': 'chromium',
                        'revision': 'r3_1',
                        'commit_position': None,
                        'url': None,
                        'score': 5,
                        'hints': {
                            'added x/y/f3_1.cc (and it was in log)': 5,
                        },
                    },
                    {
                        'build_number': 4,
                        'repo_name': 'chromium',
                        'revision': 'r4_1',
                        'commit_position': None,
                        'url': None,
                        'score': 2,
                        'hints': {
                            'modified f4.cc (and it was in log)': 2,
                        },
                    }
                ],
                'tests': [
                    {
                        'test_name': 'Unittest1.Subtest1',
                        'first_failure': 3,
                        'last_pass': 2,
                        'suspected_cls': [
                            {
                                'build_number': 2,
                                'repo_name': 'chromium',
                                'revision': 'r2_1',
                                'commit_position': None,
                                'url': None,
                                'score': 5,
                                'hints': {
                                    'added x/y/f99_1.cc (and it was in log)': 5,
                                },
                            }
                        ]
                    },
                    {
                        'test_name': 'Unittest2.Subtest1',
                        'first_failure': 4,
                        'last_pass': 2,
                        'suspected_cls': [
                            {
                                'build_number': 2,
                                'repo_name': 'chromium',
                                'revision': 'r2_1',
                                'commit_position': None,
                                'url': None,
                                'score': 5,
                                'hints': {
                                    'added x/y/f99_1.cc (and it was in log)': 5,
                                },
                            }
                        ]
                    },
                    {
                        'test_name': 'Unittest3.Subtest1',
                        'first_failure': 4,
                        'last_pass': 2,
                        'suspected_cls': []
                    }
                ]
            }
        ]
    }
    analysis.put()

    expected_results = [
        {
            'master_url': master_url,
            'builder_name': builder_name,
            'build_number': build_number,
            'step_name': 'a',
            'is_sub_test': False,
            'first_known_failed_build_number': 4,
            'suspected_cls': [
                {
                    'repo_name': 'chromium',
                    'revision': 'r4_2',
                    'commit_position': 42,
                }
            ],
            'analysis_approach': 'TRY_JOB',
        },
        {
            'master_url': master_url,
            'builder_name': builder_name,
            'build_number': build_number,
            'step_name': 'b',
            'is_sub_test': True,
            'test_name': 'Unittest1.Subtest1',
            'first_known_failed_build_number': 3,
            'suspected_cls': [
                {
                    'repo_name': 'chromium',
                    'revision': 'r2_1',
                }
            ],
            'analysis_approach': 'HEURISTIC',
        },
        {
            'master_url': master_url,
            'builder_name': builder_name,
            'build_number': build_number,
            'step_name': 'b',
            'is_sub_test': True,
            'test_name': 'Unittest2.Subtest1',
            'first_known_failed_build_number': 4,
            'suspected_cls': [
                {
                    'repo_name': 'chromium',
                    'revision': 'r2_1',
                }
            ],
            'analysis_approach': 'HEURISTIC',
        },
        {
            'master_url': master_url,
            'builder_name': builder_name,
            'build_number': build_number,
            'step_name': 'b',
            'is_sub_test': True,
            'test_name': 'Unittest3.Subtest1',
            'first_known_failed_build_number': 4,
            'suspected_cls': [
                {
                    'repo_name': 'chromium',
                    'revision': 'r4_10',
                    'commit_position': 410,
                }
            ],
            'analysis_approach': 'TRY_JOB',
        }
    ]

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_results, response.json_body.get('results'))

  def testAnalysisRequestQueuedAsExpected(self):
    master_name = 'm'
    builder_name = 'b'
    build_number = 5

    master_url = 'https://build.chromium.org/p/%s' % master_name
    builds = {
        'builds': [
            {
                'master_url': master_url,
                'builder_name': builder_name,
                'build_number': build_number
            }
        ]
    }

    expected_result = []

    self._MockMasterIsSupported(supported=True)

    response = self.call_api('AnalyzeBuildFailures', body=builds)
    self.assertEqual(200, response.status_int)
    self.assertEqual(expected_result, response.json_body.get('results', []))
    self.assertEqual(1, len(self.taskqueue_requests))

    expected_payload_json = {
        'builds': [
            {
                'master_name': master_name,
                'builder_name': builder_name,
                'build_number': build_number,
                'failed_steps': [],
            },
        ]
    }
    self.assertEqual(
        expected_payload_json,
        json.loads(self.taskqueue_requests[0].get('payload')))
