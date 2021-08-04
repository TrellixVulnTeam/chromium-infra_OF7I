# Copyright (c) 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Presubmit script for infra/services/android_docker/.

See http://dev.chromium.org/developers/how-tos/depottools/presubmit-scripts for
details on the presubmit API built into depot_tools.

Note: this needs its own PRESUBMIT.py since we can't use infra's root test.py
to provide coverage for this code given our dependency on linux-only libusb
wheels (that only a custom vpython spec can provide).
"""

import os

USE_PYTHON3 = True


def CommonChecks(input_api, output_api):
  results = []
  root_infra_path = os.path.join(input_api.PresubmitLocalPath(), '..', '..',
                                 '..')

  test_env = dict(input_api.environ)
  test_env.update({
      'PYTHONPATH': root_infra_path,
  })
  # pylint: disable=unused-variable
  tests = input_api.canned_checks.GetUnitTestsInDirectory(
      input_api,
      output_api,
      'test/',
      [r'^.+_test\.py$'],
      env=test_env,
      run_on_python2=True,
      # TODO(crbug.com/1236129): Enable py3.
      run_on_python3=False)

  # TODO(crbug.com/1236129): Actually run these tests after fixing them.
  #results += input_api.RunTests(tests)

  return results


def CheckChangeOnUpload(input_api, output_api):
  return CommonChecks(input_api, output_api)


def CheckChangeOnCommit(input_api, output_api):
  return CommonChecks(input_api, output_api)
