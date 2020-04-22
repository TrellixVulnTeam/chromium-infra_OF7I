# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
"""Tests for the permissions servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from api.v1 import permission_converters as pc
from api.v1 import permissions_servicer
from api.v1.api_proto import permissions_pb2
from api.v1.api_proto import permission_objects_pb2
from framework import exceptions
from framework import monorailcontext
from framework import permissions
from testing import fake
from services import features_svc
from services import service_manager


class PermissionsServicerTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        features=fake.FeaturesService(),
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        config=fake.ConfigService(),
        user=fake.UserService(),
        usergroup=fake.UserGroupService())
    self.project = self.services.project.TestAddProject(
        'proj', project_id=789, committer_ids=[111])
    self.permissions_svcr = permissions_servicer.PermissionsServicer(
        self.services, make_rate_limiter=False)
    self.user_1 = self.services.user.TestAddUser('goose_1@example.com', 111)
    self.hotlist_1 = self.services.features.TestAddHotlist(
        'ThingsToBreak', owner_ids=[self.user_1.user_id])
    self.services.config.CreateFieldDef(
        self.cnxn, self.project.project_id, 'Field_1', 'STR_TYPE', None, None,
        None, None, None, None, None, None, None, None, None, None, None, None,
        [], [])
    self.config = self.services.config.GetProjectConfig(
        self.cnxn, self.project.project_id)

  def CallWrapped(self, wrapped_handler, *args, **kwargs):
    return wrapped_handler.wrapped(self.permissions_svcr, *args, **kwargs)

  def testBatchGetPermissionSets_Hotlist(self):
    """We can batch get PermissionSets for hotlists."""
    hotlist_1_name = 'hotlists/%s' % self.hotlist_1.hotlist_id
    request = permissions_pb2.BatchGetPermissionSetsRequest(
        names=[hotlist_1_name])
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(
        self.permissions_svcr.BatchGetPermissionSets, mc, request)

    expected_permission_sets = [
        permission_objects_pb2.PermissionSet(
            resource=hotlist_1_name,
            permissions=[
                permission_objects_pb2.Permission.Value('HOTLIST_ADMINISTER'),
                permission_objects_pb2.Permission.Value('HOTLIST_EDIT'),
            ])
    ]
    self.assertEqual(
        response,
        permissions_pb2.BatchGetPermissionSetsResponse(
            permission_sets=expected_permission_sets))

  def testBatchGetPermissionSets_FieldDef(self):
    """We can batch get PermissionSets for fields."""
    field = self.config.field_defs[0]
    field_1_name = 'projects/%s/fieldDefs/%s' % (
        self.project.project_name, field.field_name)
    request = permissions_pb2.BatchGetPermissionSetsRequest(
        names=[field_1_name])
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(self.project)
    response = self.CallWrapped(
        self.permissions_svcr.BatchGetPermissionSets, mc, request)

    expected_permission_sets = [
        permission_objects_pb2.PermissionSet(
            resource=field_1_name,
            permissions=[
                permission_objects_pb2.Permission.Value('FIELD_DEF_VALUE_EDIT'),
            ])
    ]
    self.assertEqual(
        response,
        permissions_pb2.BatchGetPermissionSetsResponse(
            permission_sets=expected_permission_sets))

  # Each case of recognized resource name is tested in testBatchGetPermissions.
  def testGetPermissionSet_InvalidName(self):
    """We raise exception when the resource name is unrecognized."""
    we = None
    with self.assertRaises(exceptions.InputException):
      self.permissions_svcr._GetPermissionSet(self.cnxn, we, 'goose/honk')
