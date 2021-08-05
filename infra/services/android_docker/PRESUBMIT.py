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
import sys

USE_PYTHON3 = True


def CommonChecks(input_api, output_api):
  results = []
  root_infra_path = os.path.join(input_api.PresubmitLocalPath(), '..', '..',
                                 '..')

  # Given the use of USB subsystems, the code here has a hard-dependency on
  # linux.
  if not sys.platform.startswith('linux'):
    return []

  test_env = dict(input_api.environ)
  test_env.update({
      'PYTHONPATH': root_infra_path,
  })
  tests = input_api.canned_checks.GetUnitTestsInDirectory(
      input_api,
      output_api,
      'test/', [r'^.+_test\.py$'],
      env=test_env,
      skip_shebang_check=True,
      run_on_python2=False,
      run_on_python3=True)

  results += input_api.RunTests(tests)
  return results


def CheckChangeOnUpload(input_api, output_api):
  return CommonChecks(input_api, output_api)


def CheckChangeOnCommit(input_api, output_api):
  return CommonChecks(input_api, output_api)
