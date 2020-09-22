# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the ban spammer feature."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import json
import mock
import os
import unittest
import urllib
import webapp2

import settings
from features import banspammer
from framework import framework_views
from framework import permissions
from framework import urls
from proto import tracker_pb2
from services import service_manager
from testing import fake
from testing import testing_helpers

class BanSpammerTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = 'fake cnxn'
    self.mr = testing_helpers.MakeMonorailRequest()
    self.services = service_manager.Services(
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        spam=fake.SpamService(),
        user=fake.UserService())
    self.servlet = banspammer.BanSpammer('req', 'res', services=self.services)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def testProcessFormData_noPermission(self, get_client_mock):
    self.servlet.services.user.TestAddUser('member', 222)
    self.servlet.services.user.TestAddUser('spammer@domain.com', 111)
    mr = testing_helpers.MakeMonorailRequest(
        path='/u/spammer@domain.com/banSpammer.do',
        perms=permissions.GetPermissions(None, {}, None))
    mr.viewed_user_auth.user_view = framework_views.MakeUserView(mr.cnxn,
        self.servlet.services.user, 111)
    mr.auth.user_id = 222
    self.assertRaises(permissions.PermissionException,
        self.servlet.AssertBasePermission, mr)
    try:
      self.servlet.ProcessFormData(mr, {})
    except permissions.PermissionException:
      pass
    self.assertEqual(get_client_mock().queue_path.call_count, 0)
    self.assertEqual(get_client_mock().create_task.call_count, 0)

  @mock.patch('framework.cloud_tasks_helpers._get_client')
  def testProcessFormData_ok(self, get_client_mock):
    self.servlet.services.user.TestAddUser('owner', 222)
    self.servlet.services.user.TestAddUser('spammer@domain.com', 111)
    mr = testing_helpers.MakeMonorailRequest(
        path='/u/spammer@domain.com/banSpammer.do',
        perms=permissions.ADMIN_PERMISSIONSET)
    mr.viewed_user_auth.user_view = framework_views.MakeUserView(mr.cnxn,
        self.servlet.services.user, 111)
    mr.viewed_user_auth.user_pb.user_id = 111
    mr.auth.user_id = 222
    self.servlet.ProcessFormData(mr, {'banned': 'non-empty'})

    params = {'spammer_id': 111, 'reporter_id': 222, 'is_spammer': True}
    task = {
        'app_engine_http_request':
            {
                'relative_uri':
                    urls.BAN_SPAMMER_TASK + '.do?' + urllib.urlencode(params)
            }
    }
    get_client_mock().queue_path.assert_called_with(
        settings.app_id, settings.CLOUD_TASKS_REGION, 'default')
    get_client_mock().create_task.assert_called_once()
    ((_parent, called_task), _kwargs) = get_client_mock().create_task.call_args
    self.assertEqual(called_task, task)


class BanSpammerTaskTest(unittest.TestCase):
  def setUp(self):
    self.services = service_manager.Services(
        issue=fake.IssueService(),
        spam=fake.SpamService())
    self.res = webapp2.Response()
    self.servlet = banspammer.BanSpammerTask('req', self.res,
        services=self.services)

  def testProcessFormData_okNoIssues(self):
    mr = testing_helpers.MakeMonorailRequest(
        path=urls.BAN_SPAMMER_TASK + '.do', method='POST',
        params={'spammer_id': 111, 'reporter_id': 222})

    self.servlet.HandleRequest(mr)
    self.assertEqual(self.res.body, json.dumps({'comments': 0, 'issues': 0}))

  def testProcessFormData_okSomeIssues(self):
    mr = testing_helpers.MakeMonorailRequest(
        path=urls.BAN_SPAMMER_TASK + '.do', method='POST',
        params={'spammer_id': 111, 'reporter_id': 222})

    for i in range(0, 10):
      issue = fake.MakeTestIssue(
          1, i, 'issue_summary', 'New', 111, project_name='project-name')
      self.servlet.services.issue.TestAddIssue(issue)

    self.servlet.HandleRequest(mr)
    self.assertEqual(self.res.body, json.dumps({'comments': 0, 'issues': 10}))

  def testProcessFormData_okSomeCommentsAndIssues(self):
    mr = testing_helpers.MakeMonorailRequest(
        path=urls.BAN_SPAMMER_TASK + '.do', method='POST',
        params={'spammer_id': 111, 'reporter_id': 222})

    for i in range(0, 12):
      issue = fake.MakeTestIssue(
          1, i, 'issue_summary', 'New', 111, project_name='project-name')
      self.servlet.services.issue.TestAddIssue(issue)

    for i in range(10, 20):
      issue = fake.MakeTestIssue(
          1, i, 'issue_summary', 'New', 222, project_name='project-name')
      self.servlet.services.issue.TestAddIssue(issue)
      for _ in range(0, 5):
        comment = tracker_pb2.IssueComment()
        comment.project_id = 1
        comment.user_id = 111
        comment.issue_id = issue.issue_id
        self.servlet.services.issue.TestAddComment(comment, issue.local_id)
    self.servlet.HandleRequest(mr)
    self.assertEqual(self.res.body, json.dumps({'comments': 50, 'issues': 10}))
