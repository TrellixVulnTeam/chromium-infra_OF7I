# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
"""Tests for the users servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import unittest

from google.protobuf import empty_pb2

from api import resource_name_converters as rnc
from api.v3 import users_servicer
from api.v3 import converters
from api.v3.api_proto import users_pb2
from api.v3.api_proto import user_objects_pb2
from framework import exceptions
from framework import monorailcontext
from framework import permissions
from testing import fake
from testing import testing_helpers
from services import features_svc
from services import user_svc
from services import service_manager


class UsersServicerTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        user=fake.UserService(),
        usergroup=fake.UserGroupService(),
        project=fake.ProjectService(),
        project_star=fake.ProjectStarService())
    self.users_svcr = users_servicer.UsersServicer(
        self.services, make_rate_limiter=False)

    self.user_1 = self.services.user.TestAddUser('user_111@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('user_222@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('user_333@example.com', 333)

    self.project_1 = self.services.project.TestAddProject(
        'proj', project_id=789)

    self.converter = None

  def CallWrapped(self, wrapped_handler, mc, *args, **kwargs):
    self.converter = converters.Converter(mc, self.services)
    self.users_svcr.converter = self.converter
    return wrapped_handler.wrapped(self.users_svcr, mc, *args, **kwargs)

  def testGetUser(self):
    request = users_pb2.GetUserRequest(name='users/222')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(self.users_svcr.GetUser, mc, request)
    expected_response = user_objects_pb2.User(
        name='users/222',
        display_name=testing_helpers.ObscuredEmail(self.user_2.email),
        availability_message='User never visited')
    self.assertEqual(response, expected_response)

  def testBatchGetUsers(self):
    request = users_pb2.BatchGetUsersRequest(
        names=['users/222', 'users/333'])
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(self.users_svcr.BatchGetUsers, mc, request)
    expected_users = [
        user_objects_pb2.User(
            name='users/222',
            display_name=testing_helpers.ObscuredEmail(self.user_2.email),
            availability_message='User never visited'),
        user_objects_pb2.User(
            name='users/333',
            display_name=testing_helpers.ObscuredEmail(self.user_3.email),
            availability_message='User never visited')
    ]
    self.assertEqual(
        response, users_pb2.BatchGetUsersResponse(users=expected_users))

  def testStarProject(self):
    request = users_pb2.StarProjectRequest(project='projects/proj')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(self.users_svcr.StarProject, mc, request)
    expected_name = 'users/111/projectStars/proj'

    self.assertEqual(response, user_objects_pb2.ProjectStar(name=expected_name))

  def testUnStarProject(self):
    request = users_pb2.UnStarProjectRequest(project='projects/proj')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(self.users_svcr.UnStarProject, mc, request)

    self.assertEqual(response, empty_pb2.Empty())

    is_starred = self.services.project_star.IsItemStarredBy(self.cnxn, 789, 111)
    self.assertFalse(is_starred)

  def testListProjectStars(self):
    request = users_pb2.ListProjectStarsRequest(parent='users/111')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user_1.email)
    mc.LookupLoggedInUserPerms(None)

    self.services.project_star.SetStar(
        self.cnxn, self.project_1.project_id, self.user_1.user_id, True)

    response = self.CallWrapped(self.users_svcr.ListProjectStars, mc, request)

    expected_response = users_pb2.ListProjectStarsResponse(
        project_stars=[
            user_objects_pb2.ProjectStar(name='users/111/projectStars/proj')
        ])
    self.assertEqual(response, expected_response)
