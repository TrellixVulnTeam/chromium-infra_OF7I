# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

PRESUBMIT_VERSION = '2.0.0'


def CheckPanicUtil(input_api, output_api):
  d = (
      input_api.os_path.dirname(input_api.PresubmitLocalPath()) +
      input_api.os_path.sep)

  def panic_utils_not_used(file_ext, line):
    del file_ext
    violation = 'util.PanicOnError' in line or 'util.PanicIf' in line
    return not violation

  def source_file_filter(affected_file):
    f = affected_file.AbsoluteLocalPath()
    if not f.endswith('.go'):
      return False
    if not f.startswith(d):
      return False
    f = f[len(d):].replace(input_api.os_path.sep, '/')
    return not (f.startswith('bootstrapper/fakes/') or
                f.endswith('/test_utils.go') or f.endswith('_test.go'))

  def error_formatter(filename, line_number, line):
    return '* {}:{}\n{}'.format(filename, line_number, line)

  violations = input_api.canned_checks._FindNewViolationsOfRule(
      panic_utils_not_used,
      input_api,
      source_file_filter=source_file_filter,
      error_formatter=error_formatter,
  )

  if violations:
    message = [
        'Found new uses of util.PanicIf and/or util.PanicOnError '
        'used outside of test or fakes code',
    ]
    return [output_api.PresubmitError('\n'.join(message + violations))]
  return []
