# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is govered by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for features.pubsub."""

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import os
import unittest

from mock import Mock

from google.appengine.ext import testbed

from features import pubsub
from services import service_manager
from testing import fake
from testing import testing_helpers


class PublishPubsubIssueChangeTaskTest(unittest.TestCase):

  def setUp(self):
    self.testbed = testbed.Testbed()
    self.testbed.activate()
    self.testbed.init_memcache_stub()
    self.testbed.init_taskqueue_stub()
    self.taskqueue_stub = self.testbed.get_stub(testbed.TASKQUEUE_SERVICE_NAME)
    self.taskqueue_stub._root_path = os.path.dirname(
        os.path.dirname(os.path.dirname(__file__)))
    self.services = service_manager.Services(
        user=fake.UserService(),
        project=fake.ProjectService(),
        config=fake.ConfigService(),
        issue=fake.IssueService(),
        features=fake.FeaturesService())
    self.services.project.TestAddProject(
        'test-project', owner_ids=[1, 3],
        project_id=12345)

    # Stub the pubsub API (there is no pubsub testbed stub).
    self.pubsub_client_mock = Mock()
    pubsub.set_up_pubsub_api = Mock(return_value=self.pubsub_client_mock)

  def tearDown(self):
    self.testbed.deactivate()

  def testPublishPubsubIssueChangeTask_NoIssueIdParam(self):
    """Test case when issue_id param is not passed."""
    task = pubsub.PublishPubsubIssueChangeTask(
        request=None, response=None, services=self.services)
    mr = testing_helpers.MakeMonorailRequest(
        user_info={'user_id': 1},
        params={},
        method='POST',
        services=self.services)
    result = task.HandleRequest(mr)
    expected_body = {
      'error': 'Cannot proceed without a valid issue ID.',
    }
    self.assertEqual(result, expected_body)

  def testPublishPubsubIssueChangeTask_PubSubAPIInitFailure(self):
    """Test case when pub/sub API fails to init."""
    pubsub.set_up_pubsub_api = Mock(return_value=None)
    task = pubsub.PublishPubsubIssueChangeTask(
        request=None, response=None, services=self.services)
    mr = testing_helpers.MakeMonorailRequest(
        user_info={'user_id': 1},
        params={},
        method='POST',
        services=self.services)
    result = task.HandleRequest(mr)
    expected_body = {
      'error': 'Pub/Sub API init failure.',
    }
    self.assertEqual(result, expected_body)

  def testPublishPubsubIssueChangeTask_Normal(self):
    """Test normal happy-path case."""
    task = pubsub.PublishPubsubIssueChangeTask(
        request=None, response=None, services=self.services)
    params = {'issue_id': 12345001}
    mr = testing_helpers.MakeMonorailRequest(
        user_info={'user_id': 1},
        params=params,
        method='POST',
        services=self.services)
    result = task.HandleRequest(mr)

    self.pubsub_client_mock.projects().topics().publish.assert_called_once_with(
      topic='projects/testing-app/topics/issue-updates',
      body={
        'messages': [{
          'attributes': {
            'issue_id': '12345001',
          },
        }],
      }
    )
    self.pubsub_client_mock.projects().topics().publish() \
        .execute.assert_called_once()
    self.assertEqual(result, {})
