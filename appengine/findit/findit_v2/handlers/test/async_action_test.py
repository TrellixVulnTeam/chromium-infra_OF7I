# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import mock

import webapp2

from testing_utils import testing

from findit_v2.handlers import async_action
from findit_v2.model.culprit_action import CulpritAction
from findit_v2.model.gitiles_commit import Culprit


class AsyncActionTest(testing.AppengineTestCase):
  app_module = webapp2.WSGIApplication([
      ('/url', async_action.AsyncAction),
  ],
                                       debug=True)

  def testBadRequest(self):
    headers = {'X-AppEngine-QueueName': 'task_queue'}
    response = self.test_app.post('/url', '{}', headers=headers)
    # Expect 200 because we don't want to retry a malformed request.
    self.assertEquals(200, response.status_int)

  @mock.patch('findit_v2.services.gerrit_actions.GerritActions.NotifyCulprit')
  def testNotify(self, mocked_func):
    culprit = Culprit.GetOrCreate(
        'host', 'project', 'ref', 'id10', commit_position=10)
    payload = {
        'project': 'chromium',
        'action': 'NotifyCulprit',
        'culprit_key': culprit.key.urlsafe(),
        'message': 'message',
        'silent_notification': True,
    }
    error = async_action.AsyncAction()._RunAction(payload)
    self.assertIsNone(error)
    mocked_func.assert_called_once_with(
        culprit, 'message', silent_notification=True)

  @mock.patch('findit_v2.services.gerrit_actions.GerritActions.CommitRevert')
  @mock.patch('findit_v2.services.gerrit_actions.GerritActions.CreateRevert')
  def testCommitRevert(self, mock_create, mock_submit):
    revert_id = 678
    culprit = Culprit.GetOrCreate(
        'host', 'project', 'ref', 'id11', commit_position=11)
    CulpritAction.Create(culprit, CulpritAction.REVERT).put()
    payload = {
        'project': 'chromium',
        'action': 'CommitRevert',
        'culprit_key': culprit.key.urlsafe(),
        'request_confirmation_message': 'message',
        'revert_description': 'revert description',
    }
    revert_info = {'id': revert_id}
    mock_create.return_value = revert_info
    error = async_action.AsyncAction()._RunAction(payload)
    self.assertIsNone(error)

    action = CulpritAction.CreateKey(culprit).get()
    self.assertEquals(action.revert_change, revert_id)
    self.assertTrue(action.revert_committed)
    mock_create.assert_called_once_with(culprit, 'revert description')
    mock_submit.assert_called_once_with(revert_info, 'message')

  @mock.patch('findit_v2.services.gerrit_actions.GerritActions.RequestReview')
  @mock.patch('findit_v2.services.gerrit_actions.GerritActions.CreateRevert')
  def testRequestReview(self, mock_create, mock_request_review):
    revert_id = 679
    culprit = Culprit.GetOrCreate(
        'host', 'project', 'ref', 'id12', commit_position=12)
    CulpritAction.Create(culprit, CulpritAction.REVERT).put()
    payload = {
        'project': 'chromium',
        'action': 'RequestReview',
        'culprit_key': culprit.key.urlsafe(),
        'request_review_message': 'message',
        'revert_description': 'revert description',
    }
    revert_info = {'id': revert_id}
    mock_create.return_value = revert_info
    error = async_action.AsyncAction()._RunAction(payload)
    self.assertIsNone(error)

    action = CulpritAction.CreateKey(culprit).get()
    self.assertEquals(action.revert_change, revert_id)
    self.assertFalse(action.revert_committed)
    mock_create.assert_called_once_with(culprit, 'revert description')
    mock_request_review.assert_called_once_with(revert_info, 'message')
