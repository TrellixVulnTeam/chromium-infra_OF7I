# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from google.appengine.ext import ndb


class TestLocation(ndb.Model):
  """The location of a test in the source tree"""
  file_path = ndb.StringProperty(required=True)
  line_number = ndb.IntegerProperty(required=True)
