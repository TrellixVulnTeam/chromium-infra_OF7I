# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

from waterfall import waterfall_config


def GetPostsubmitPlatformInfoMap(luci_project):
  """Returns a map of postsubmit platform information.

  The map contains per-luci_project platform information, and following is
  an example config:
  {
    'postsubmit_platform_info_map': {
      'chromium': {
        'linux': {
          'bucket': 'ci',
          'buider': 'linux-code-coverage',
          'coverage_tool': 'clang',
          'ui_name': 'Linux (C/C++)',
        }
      }
    }
  }
  """
  return waterfall_config.GetCodeCoverageSettings().get(
      'postsubmit_platform_info_map', {}).get(luci_project, {})
