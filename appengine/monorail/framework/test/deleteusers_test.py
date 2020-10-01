# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Unit tests for deleteusers classes."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import logging
import mock
import unittest
import urllib

from framework import cloud_tasks_helpers
from framework import deleteusers
from framework import framework_constants
from framework import urls
from services import service_manager
from testing import fake
from testing import testing_helpers

class TestWipeoutSyncCron(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(user=fake.UserService())
    self.task = deleteusers.WipeoutSyncCron(
        request=None, response=None, services=self.services)
    self.user_1 = self.services.user.TestAddUser('user1@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('user2@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('user3@example.com', 333)

  def generate_simple_task(self, url, body):
    return {
        'app_engine_http_request':
            {
                'relative_uri': url,
                'body': body,
                'headers': {
                    'Content-type': 'application/x-www-form-urlencoded'
                }
            }
    }

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def testHandleRequest(self, get_client_mock):
    mr = testing_helpers.MakeMonorailRequest(
        path='url/url?batchsize=2',
        services=self.services)
    self.task.HandleRequest(mr)

    self.assertEqual(get_client_mock().create_task.call_count, 3)

    expected_task = self.generate_simple_task(
        urls.SEND_WIPEOUT_USER_LISTS_TASK + '.do', 'limit=2&offset=0')
    get_client_mock().create_task.assert_any_call(
        get_client_mock().queue_path(),
        expected_task,
        retry=cloud_tasks_helpers._DEFAULT_RETRY)

    expected_task = self.generate_simple_task(
        urls.SEND_WIPEOUT_USER_LISTS_TASK + '.do', 'limit=2&offset=2')
    get_client_mock().create_task.assert_any_call(
        get_client_mock().queue_path(),
        expected_task,
        retry=cloud_tasks_helpers._DEFAULT_RETRY)

    expected_task = self.generate_simple_task(
        urls.DELETE_WIPEOUT_USERS_TASK + '.do', '')
    get_client_mock().create_task.assert_any_call(
        get_client_mock().queue_path(),
        expected_task,
        retry=cloud_tasks_helpers._DEFAULT_RETRY)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def testHandleRequest_NoBatchSizeParam(self, get_client_mock):
    mr = testing_helpers.MakeMonorailRequest(services=self.services)
    self.task.HandleRequest(mr)

    expected_task = self.generate_simple_task(
        urls.SEND_WIPEOUT_USER_LISTS_TASK + '.do',
        'limit={}&offset=0'.format(deleteusers.MAX_BATCH_SIZE))
    get_client_mock().create_task.assert_any_call(
        get_client_mock().queue_path(),
        expected_task,
        retry=cloud_tasks_helpers._DEFAULT_RETRY)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def testHandleRequest_NoUsers(self, get_client_mock):
    mr = testing_helpers.MakeMonorailRequest()
    self.services.user.users_by_id = {}
    self.task.HandleRequest(mr)

    calls = get_client_mock().create_task.call_args_list
    self.assertEqual(len(calls), 0)


class SendWipeoutUserListsTaskTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(user=fake.UserService())
    self.task = deleteusers.SendWipeoutUserListsTask(
        request=None, response=None, services=self.services)
    self.task.sendUserLists = mock.Mock()
    deleteusers.authorize = mock.Mock(return_value='service')
    self.user_1 = self.services.user.TestAddUser('user1@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('user2@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('user3@example.com', 333)

  def testHandleRequest_NoBatchSizeParam(self):
    mr = testing_helpers.MakeMonorailRequest(path='url/url?limit=2&offset=1')
    self.task.HandleRequest(mr)
    deleteusers.authorize.assert_called_once_with()
    self.task.sendUserLists.assert_called_once_with(
        'service', [
            {'id': self.user_2.email},
            {'id': self.user_3.email}])

  def testHandleRequest_NoLimit(self):
    mr = testing_helpers.MakeMonorailRequest()
    self.services.user.users_by_id = {}
    with self.assertRaisesRegexp(AssertionError, 'Missing param limit'):
      self.task.HandleRequest(mr)

  def testHandleRequest_NoOffset(self):
    mr = testing_helpers.MakeMonorailRequest(path='url/url?limit=3')
    self.services.user.users_by_id = {}
    with self.assertRaisesRegexp(AssertionError, 'Missing param offset'):
      self.task.HandleRequest(mr)

  def testHandleRequest_ZeroOffset(self):
    mr = testing_helpers.MakeMonorailRequest(path='url/url?limit=2&offset=0')
    self.task.HandleRequest(mr)
    self.task.sendUserLists.assert_called_once_with(
        'service', [
            {'id': self.user_1.email},
            {'id': self.user_2.email}])


class DeleteWipeoutUsersTaskTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services()
    deleteusers.authorize = mock.Mock(return_value='service')
    self.task = deleteusers.DeleteWipeoutUsersTask(
        request=None, response=None, services=self.services)
    deleted_users = [
        {'id': 'user1@gmail.com'}, {'id': 'user2@gmail.com'},
        {'id': 'user3@gmail.com'}, {'id': 'user4@gmail.com'}]
    self.task.fetchDeletedUsers = mock.Mock(return_value=deleted_users)

  def generate_simple_task(self, url, body):
    return {
        'app_engine_http_request':
            {
                'relative_uri': url,
                'body': body,
                'headers': {
                    'Content-type': 'application/x-www-form-urlencoded'
                }
            }
    }

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def testHandleRequest(self, get_client_mock):
    mr = testing_helpers.MakeMonorailRequest(path='url/url?limit=3')
    self.task.HandleRequest(mr)

    deleteusers.authorize.assert_called_once_with()
    self.task.fetchDeletedUsers.assert_called_once_with('service')
    ((_app_id, _region, queue),
     _kwargs) = get_client_mock().queue_path.call_args
    self.assertEqual(queue, framework_constants.QUEUE_DELETE_USERS)

    self.assertEqual(get_client_mock().create_task.call_count, 2)

    query = urllib.urlencode(
        {'emails': 'user1@gmail.com,user2@gmail.com,user3@gmail.com'})
    expected_task = self.generate_simple_task(
        urls.DELETE_USERS_TASK + '.do', query)

    get_client_mock().create_task.assert_any_call(
        get_client_mock().queue_path(),
        expected_task,
        retry=cloud_tasks_helpers._DEFAULT_RETRY)

    query = urllib.urlencode({'emails': 'user4@gmail.com'})
    expected_task = self.generate_simple_task(
        urls.DELETE_USERS_TASK + '.do', query)

    get_client_mock().create_task.assert_any_call(
        get_client_mock().queue_path(),
        expected_task,
        retry=cloud_tasks_helpers._DEFAULT_RETRY)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def testHandleRequest_DefaultMax(self, get_client_mock):
    mr = testing_helpers.MakeMonorailRequest(path='url/url')
    self.task.HandleRequest(mr)

    deleteusers.authorize.assert_called_once_with()
    self.task.fetchDeletedUsers.assert_called_once_with('service')
    self.assertEqual(get_client_mock().create_task.call_count, 1)

    emails = 'user1@gmail.com,user2@gmail.com,user3@gmail.com,user4@gmail.com'
    query = urllib.urlencode({'emails': emails})
    expected_task = self.generate_simple_task(
        urls.DELETE_USERS_TASK + '.do', query)

    get_client_mock().create_task.assert_any_call(
        get_client_mock().queue_path(),
        expected_task,
        retry=cloud_tasks_helpers._DEFAULT_RETRY)
