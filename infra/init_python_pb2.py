# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Ensures protos can be imported from python_pb2 dir.

Use:
  >>> from infra import init_python_pb2  # pylint: disable=unused-import
  >>> from go.chromium.org.luci.buildbucket.proto import common_pb2
"""

import os
import sys

pb2_dir = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'python_pb2')
if pb2_dir not in sys.path:  # pragma: no cover
  sys.path.append(pb2_dir)
