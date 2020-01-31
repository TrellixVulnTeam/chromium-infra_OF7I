# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

"""Tests for converting internal protorpc to external protoc."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from api.v1 import converters
from api.v1.api_proto import feature_objects_pb2
from api.v1.api_proto import issue_objects_pb2
from testing import fake
from services import service_manager

class ConverterFunctionsTest(unittest.TestCase):

  def testConvertHotlist(self):
    """We can convert a Hotlist."""
    hotlist = fake.Hotlist(
        'Hotlist-Name', 240, default_col_spec='chicken goose',
        is_private=False, owner_ids=[111], editor_ids=[222, 333],
        summary='Hotlist summary', description='Hotlist Description')
    expected_api_hotlist = feature_objects_pb2.Hotlist(
        name='hotlists/240',
        display_name=hotlist.name,
        owner='users/111',
        summary=hotlist.summary,
        description=hotlist.description,
        editors=['users/222', 'users/333'],
        hotlist_privacy=feature_objects_pb2.Hotlist.HotlistPrivacy.Value(
            'PUBLIC'),
        default_columns=[
            issue_objects_pb2.IssuesListColumn(column='chicken'),
            issue_objects_pb2.IssuesListColumn(column='goose')])
    self.assertEqual(converters.ConvertHotlist(hotlist), expected_api_hotlist)

  def testConvertHotlist_DefaultValues(self):
    """We can convert a Hotlist with some empty or default values."""
    hotlist = fake.Hotlist(
        'Hotlist-Name', 241, is_private=True, owner_ids=[111],
        summary='Hotlist summary', description='Hotlist Description',
        default_col_spec='')
    expected_api_hotlist = feature_objects_pb2.Hotlist(
        name='hotlists/241',
        display_name=hotlist.name,
        owner='users/111',
        summary=hotlist.summary,
        description=hotlist.description)
    self.assertEqual(converters.ConvertHotlist(hotlist), expected_api_hotlist)
