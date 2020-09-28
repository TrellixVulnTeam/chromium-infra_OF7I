# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for prepareandsend.py"""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import mock
import unittest
import urllib
import urlparse

from features import send_notifications
from framework import urls
from tracker import tracker_bizobj


class SendNotificationTest(unittest.TestCase):

  def get_filtered_task_call_args(self, create_task_mock, relative_uri):
    return [
        (args, _kwargs)
        for (args, _kwargs) in create_task_mock.call_args_list
        if args[0]['app_engine_http_request']['relative_uri'].startswith(
            relative_uri)
    ]

  @mock.patch('framework.cloud_tasks_helpers.create_task')
  def testPrepareAndSendIssueChangeNotification(self, create_task_mock):
    send_notifications.PrepareAndSendIssueChangeNotification(
        issue_id=78901,
        hostport='testbed-test.appspotmail.com',
        commenter_id=1,
        old_owner_id=2,
        send_email=True)

    call_args_list = self.get_filtered_task_call_args(
        create_task_mock, urls.NOTIFY_ISSUE_CHANGE_TASK + '.do')
    self.assertEqual(1, len(call_args_list))

  @mock.patch('framework.cloud_tasks_helpers.create_task')
  def testPrepareAndSendIssueBlockingNotification(self, create_task_mock):
    send_notifications.PrepareAndSendIssueBlockingNotification(
        issue_id=78901,
        hostport='testbed-test.appspotmail.com',
        delta_blocker_iids=[],
        commenter_id=1,
        send_email=True)

    call_args_list = self.get_filtered_task_call_args(
        create_task_mock, urls.NOTIFY_BLOCKING_CHANGE_TASK + '.do')
    self.assertEqual(0, len(call_args_list))

    send_notifications.PrepareAndSendIssueBlockingNotification(
        issue_id=78901,
        hostport='testbed-test.appspotmail.com',
        delta_blocker_iids=[2],
        commenter_id=1,
        send_email=True)

    call_args_list = self.get_filtered_task_call_args(
        create_task_mock, urls.NOTIFY_BLOCKING_CHANGE_TASK + '.do')
    self.assertEqual(1, len(call_args_list))

  @mock.patch('framework.cloud_tasks_helpers.create_task')
  def testPrepareAndSendApprovalChangeNotification(self, create_task_mock):
    send_notifications.PrepareAndSendApprovalChangeNotification(
        78901, 3, 'testbed-test.appspotmail.com', 55)

    call_args_list = self.get_filtered_task_call_args(
        create_task_mock, urls.NOTIFY_APPROVAL_CHANGE_TASK + '.do')
    self.assertEqual(1, len(call_args_list))

  @mock.patch('framework.cloud_tasks_helpers.create_task')
  def testSendIssueBulkChangeNotification_CommentOnly(self, create_task_mock):
    send_notifications.SendIssueBulkChangeNotification(
        issue_ids=[78901],
        hostport='testbed-test.appspotmail.com',
        old_owner_ids=[2],
        comment_text='comment',
        commenter_id=1,
        amendments=[],
        send_email=True,
        users_by_id=2)

    call_args_list = self.get_filtered_task_call_args(
        create_task_mock, urls.NOTIFY_BULK_CHANGE_TASK + '.do')
    self.assertEqual(1, len(call_args_list))
    (args, _kwargs) = call_args_list[0]
    relative_uri = args[0]['app_engine_http_request']['relative_uri']
    parse_result = urlparse.urlparse(relative_uri)
    params = {
        k: v[0] for k, v in urlparse.parse_qs(parse_result.query, True).items()
    }
    self.assertEqual(params['comment_text'], 'comment')
    self.assertEqual(params['amendments'], '')

  @mock.patch('framework.cloud_tasks_helpers.create_task')
  def testSendIssueBulkChangeNotification_Normal(self, create_task_mock):
    send_notifications.SendIssueBulkChangeNotification(
        issue_ids=[78901],
        hostport='testbed-test.appspotmail.com',
        old_owner_ids=[2],
        comment_text='comment',
        commenter_id=1,
        amendments=[
            tracker_bizobj.MakeStatusAmendment('New', 'Old'),
            tracker_bizobj.MakeLabelsAmendment(['Added'], ['Removed']),
            tracker_bizobj.MakeStatusAmendment('New', 'Old'),
            ],
        send_email=True,
        users_by_id=2)

    call_args_list = self.get_filtered_task_call_args(
        create_task_mock, urls.NOTIFY_BULK_CHANGE_TASK + '.do')
    self.assertEqual(1, len(call_args_list))
    (args, _kwargs) = call_args_list[0]
    relative_uri = args[0]['app_engine_http_request']['relative_uri']
    parse_result = urlparse.urlparse(relative_uri)
    params = {k: v[0] for k, v in urlparse.parse_qs(parse_result.query).items()}
    self.assertEqual(params['comment_text'], 'comment')
    self.assertEqual(
        params['amendments'].split('\n'),
        ['    Status: New', '    Labels: -Removed Added'])

  @mock.patch('framework.cloud_tasks_helpers.create_task')
  def testPrepareAndSendDeletedFilterRulesNotifications(self, create_task_mock):
    filter_rule_strs = ['if yellow make orange', 'if orange make blue']
    send_notifications.PrepareAndSendDeletedFilterRulesNotification(
        789, 'testbed-test.appspotmail.com', filter_rule_strs)

    call_args_list = self.get_filtered_task_call_args(
        create_task_mock, urls.NOTIFY_RULES_DELETED_TASK + '.do')
    self.assertEqual(1, len(call_args_list))
    (args, _kwargs) = call_args_list[0]
    relative_uri = args[0]['app_engine_http_request']['relative_uri']
    parse_result = urlparse.urlparse(relative_uri)
    params = {k: v[0] for k, v in urlparse.parse_qs(parse_result.query).items()}
    self.assertEqual(params['project_id'], '789')
    self.assertEqual(
        params['filter_rules'], 'if yellow make orange,if orange make blue')
