# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
"""Tests for converting permission strings to API permissions enums."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from api.v1 import permission_converters as pc
from api.v1.api_proto import permission_objects_pb2
from framework import exceptions
from framework import permissions


class ConverterFunctionsTest(unittest.TestCase):

  def testConvertHotlistPermissions(self):
    api_perms = pc.ConvertHotlistPermissions(
        [permissions.ADMINISTER_HOTLIST, permissions.EDIT_HOTLIST])
    expected_perms = [
        permission_objects_pb2.Permission.Value('HOTLIST_ADMINISTER'),
        permission_objects_pb2.Permission.Value('HOTLIST_EDIT')
    ]
    self.assertEqual(api_perms, expected_perms)

  def testConvertHotlistPermissions_InvalidPermission(self):
    with self.assertRaises(exceptions.InputException):
      pc.ConvertHotlistPermissions(['EatHotlist'])

  def testConvertFieldDefPermissions(self):
    api_perms = pc.ConvertFieldDefPermissions(
        [permissions.EDIT_FIELD_DEF_VALUE, permissions.EDIT_FIELD_DEF])
    expected_perms = [
        permission_objects_pb2.Permission.Value('FIELD_DEF_VALUE_EDIT'),
        permission_objects_pb2.Permission.Value('FIELD_DEF_EDIT')
    ]
    self.assertEqual(api_perms, expected_perms)

  def testConvertFieldDefPermissions_InvalidPermission(self):
    with self.assertRaises(exceptions.InputException):
      pc.ConvertFieldDefPermissions(['EatFieldDef'])
