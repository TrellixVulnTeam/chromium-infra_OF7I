# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""infra.tools specific presubmit.
"""

USE_PYTHON3 = True


def CheckChangeOnUpload(input_api, output_api):
  output = []
  output.extend(
      input_api.RunTests([
          input_api.Command(
              name='check dockerbuild wheel dump',
              cmd=['vpython3', '-m', 'dockerbuild', 'wheel-dump', '--check'],
              kwargs={},
              message=output_api.PresubmitError)
      ]))
  return output
