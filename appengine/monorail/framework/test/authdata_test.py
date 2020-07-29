# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Unit tests for the authdata module."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import mock
import unittest

from google.appengine.api import users

from framework import authdata
from services import service_manager
from testing import fake


class AuthDataTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        user=fake.UserService(),
        usergroup=fake.UserGroupService())
    self.user_1 = self.services.user.TestAddUser('test@example.com', 111)

  def testFromRequest(self):

    class FakeUser(object):
      email = lambda _: self.user_1.email

    with mock.patch.object(users, 'get_current_user',
                           autospec=True) as mock_get_current_user:
      mock_get_current_user.return_value = FakeUser()
      auth = authdata.AuthData.FromRequest(self.cnxn, self.services)
    self.assertEqual(auth.user_id, 111)

  def testFromEmail(self):
    auth = authdata.AuthData.FromEmail(
        self.cnxn, self.user_1.email, self.services)
    self.assertEqual(auth.user_id, 111)
    self.assertEqual(auth.user_pb.email, self.user_1.email)

  def testFromuserId(self):
    auth = authdata.AuthData.FromUserID(self.cnxn, 111, self.services)
    self.assertEqual(auth.user_id, 111)
    self.assertEqual(auth.user_pb.email, self.user_1.email)

  def testFromUser(self):
    auth = authdata.AuthData.FromUser(self.cnxn, self.user_1, self.services)
    self.assertEqual(auth.user_id, 111)
    self.assertEqual(auth.user_pb.email, self.user_1.email)
