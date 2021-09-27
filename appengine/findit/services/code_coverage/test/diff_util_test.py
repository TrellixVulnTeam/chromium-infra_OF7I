# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import base64
import json
import logging
import mock
import textwrap

from google.appengine.ext import ndb

from model.code_coverage import CoveragePercentage
from services.code_coverage import diff_util
from waterfall.test.wf_testcase import WaterfallTestCase


class DiffUtilTest(WaterfallTestCase):

  # This tests following modification
  #
  # Line number |   Old Content    |   New Content
  #             |                  |
  #   1         |line1             |line2 modified
  #   2         |line2             |line3
  #   3         |line3             |line4
  def testParseAddedLineNumFromUnifiedDiff(self):
    diff_lines = [
        'diff --git a/myfile b/myfile',
        'index f2c7de5b55..e64015c0c3 100644',
        '--- a/myfile',
        '+++ b/myfile',
        '@@ -1,3 +1,3 @@',
        '-line1',
        '-line2',
        '-line3',
        '+line2 modified',
        '+line3',
        '+line4',
    ]
    response = diff_util.parse_added_line_num_from_unified_diff(diff_lines)
    self.assertEqual(len(response), 1)
    self.assertEqual(response['myfile'], set([1, 2, 3]))
