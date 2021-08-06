# Copyright (c) 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Presubmit script for infra/services/.

See http://dev.chromium.org/developers/how-tos/depottools/presubmit-scripts for
details on the presubmit API built into depot_tools.

Note: this file is mostly used for py3 tests since we can't use infra's root
test.py for python3.
"""

import os
import sys

USE_PYTHON3 = True


def CommonChecks(input_api, output_api):
  results = []
  tests = []
  root_infra_path = os.path.join(input_api.PresubmitLocalPath(), '..', '..')

  affected_files = [x.LocalPath() for x in input_api.AffectedFiles()]
  docker_services = ['android_docker', 'cros_docker', 'swarm_docker']
  include_docker_service_tests = False
  for f in affected_files:
    if any(service in f.split(os.sep) for service in docker_services):
      include_docker_service_tests = True
      break

  if include_docker_service_tests and sys.platform.startswith('linux'):
    test_env = dict(input_api.environ)
    test_env.update({
        'PYTHONPATH': root_infra_path,
    })
    for service in docker_services:
      tests.extend(
          input_api.canned_checks.GetUnitTestsInDirectory(
              input_api,
              output_api,
              os.path.join(service, 'test'), [r'^.+_test\.py$'],
              env=test_env,
              skip_shebang_check=True,
              run_on_python2=False,
              run_on_python3=True))

  results += input_api.RunTests(tests)
  return results


def CheckChangeOnUpload(input_api, output_api):
  return CommonChecks(input_api, output_api)


def CheckChangeOnCommit(input_api, output_api):
  return CommonChecks(input_api, output_api)
