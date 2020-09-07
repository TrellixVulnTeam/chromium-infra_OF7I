# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest

import httplib2
import json
import logging
import unittest

from deepdiff import DeepDiff
from apiclient import errors
from chromeperf.services import issue_tracker_service


@pytest.fixture(autouse=True)
def setup_mock_discovery_build(mocker):
  mocker.patch('chromeperf.services.issue_tracker_service.discovery.build',
               mocker.MagicMock())


def test_IssueTracker_AddBugComment_Basic(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._MakeCommentRequest = mocker.Mock()
  assert service.AddBugComment(12345, 'The comment')
  assert service._MakeCommentRequest.call_count == 1
  service._MakeCommentRequest.assert_called_with(
      12345, {
          'updates': {},
          'content': 'The comment'
      },
      project='chromium',
      send_email=True)


def test_IssueTracker_AddBugComment_Basic_EmptyProject(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._MakeCommentRequest = mocker.Mock()
  assert service.AddBugComment(12345, 'The comment', project='')
  assert service._MakeCommentRequest.call_count == 1
  service._MakeCommentRequest.assert_called_with(
      12345, {
          'updates': {},
          'content': 'The comment'
      },
      project='chromium',
      send_email=True)


def test_IssueTracker_AddBugComment_Basic_ProjectIsNone(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._MakeCommentRequest = mocker.Mock()
  assert service.AddBugComment(12345, 'The comment', project=None)
  assert service._MakeCommentRequest.call_count == 1
  service._MakeCommentRequest.assert_called_with(
      12345, {
          'updates': {},
          'content': 'The comment'
      },
      project='chromium',
      send_email=True)


def test_IssueTracker_AddBugComment_WithNoBug_ReturnsFalse(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._MakeCommentRequest = mocker.Mock()
  assert not service.AddBugComment(None, 'Some comment')
  assert not service.AddBugComment(-1, 'Some comment')


def test_IssueTracker_AddBugComment_WithOptionalParameters(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._MakeCommentRequest = mocker.Mock()
  assert service.AddBugComment(
      12345,
      'Some other comment',
      status='Fixed',
      labels=['Foo'],
      cc_list=['someone@chromium.org'])
  assert service._MakeCommentRequest.call_count == 1
  service._MakeCommentRequest.assert_called_with(
      12345, {
          'updates': {
              'status': 'Fixed',
              'cc': ['someone@chromium.org'],
              'labels': ['Foo'],
          },
          'content': 'Some other comment'
      },
      project='chromium',
      send_email=True)


def test_IssueTracker_AddBugComment_MergeBug(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._MakeCommentRequest = mocker.Mock()
  assert service.AddBugComment(12345, 'Dupe', merge_issue=54321)
  assert service._MakeCommentRequest.call_count == 1
  service._MakeCommentRequest.assert_called_with(
      12345, {
          'updates': {
              'status': 'Duplicate',
              'mergedInto': 'chromium:54321',
          },
          'content': 'Dupe'
      },
      project='chromium',
      send_email=True)


def test_IssueTracker_AddBugComment_Error(mocker):
  mocker.patch('logging.error', mocker.MagicMock())
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._ExecuteRequest = mocker.Mock(return_value=None)
  assert not service.AddBugComment(12345, 'My bug comment')
  assert service._ExecuteRequest.call_count == 1
  assert logging.error.call_count == 1


def test_IssueTracker_NewBug_Success_NewBugReturnsId(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._ExecuteRequest = mocker.Mock(return_value={'id': 333})
  response = service.NewBug('Bug title', 'body', owner='someone@chromium.org')
  bug_id = response['bug_id']
  assert service._ExecuteRequest.call_count == 1
  assert bug_id == 333


def test_IssueTracker_NewBug_Success_SupportNonChromium(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._ExecuteRequest = mocker.Mock(return_value={
      'id': 333,
      'projectId': 'non-chromium'
  })
  response = service.NewBug(
      'Bug title', 'body', owner='someone@example.com', project='non-chromium')
  bug_id = response['bug_id']
  project_id = response['project_id']
  assert service._ExecuteRequest.call_count == 1
  assert bug_id == 333
  assert project_id == 'non-chromium'


def test_IssueTracker_NewBug_Success_ProjectIsEmpty(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._ExecuteRequest = mocker.Mock(return_value={
      'id': 333,
      'projectId': 'chromium'
  })
  response = service.NewBug(
      'Bug title', 'body', owner='someone@example.com', project='')
  bug_id = response['bug_id']
  project_id = response['project_id']
  assert service._ExecuteRequest.call_count == 1
  assert bug_id == 333
  assert project_id == 'chromium'


def test_IssueTracker_NewBug_Success_ProjectIsNone(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._ExecuteRequest = mocker.Mock(return_value={
      'id': 333,
      'projectId': 'chromium'
  })
  response = service.NewBug(
      'Bug title', 'body', owner='someone@example.com', project=None)
  bug_id = response['bug_id']
  project_id = response['project_id']
  assert service._ExecuteRequest.call_count == 1
  assert bug_id == 333
  assert project_id == 'chromium'


def test_IssueTracker_NewBug_Failure_HTTPException(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._ExecuteRequest = mocker.Mock(
      side_effect=httplib2.HttpLib2Error('reason'))
  response = service.NewBug('Bug title', 'body', owner='someone@chromium.org')
  assert service._ExecuteRequest.call_count == 1
  assert 'error' in response


def test_IssueTracker_NewBug_Failure_NewBugReturnsError(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._ExecuteRequest = mocker.Mock(return_value={})
  response = service.NewBug('Bug title', 'body', owner='someone@chromium.org')
  assert service._ExecuteRequest.call_count == 1
  assert 'error' in response


def test_IssueTracker_NewBug_HttpError_NewBugReturnsError(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  error_content = {
      'error': {
          'message': 'The user does not exist: test@chromium.org',
          'code': 404
      }
  }
  service._ExecuteRequest = mocker.Mock(
      side_effect=errors.HttpError(
          mocker.Mock(return_value={'status': 404}),
          bytes(json.dumps(error_content), encoding='utf-8')))
  response = service.NewBug('Bug title', 'body', owner='someone@chromium.org')
  assert service._ExecuteRequest.call_count == 1
  assert 'error' in response


def test_IssueTracker_NewBug_UsesExpectedParams(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._MakeCreateRequest = mocker.Mock()
  service.NewBug(
      'Bug title',
      'body',
      owner='someone@chromium.org',
      cc=['somebody@chromium.org', 'nobody@chromium.org'])
  service._MakeCreateRequest.assert_called_with(
      {
          'title': 'Bug title',
          'summary': 'Bug title',
          'description': 'body',
          'labels': [],
          'components': [],
          'status': 'Assigned',
          'owner': {
              'name': 'someone@chromium.org'
          },
          'cc': mocker.ANY,
          'projectId': 'chromium',
      }, 'chromium')
  assert not DeepDiff(
      service._MakeCreateRequest.call_args[0][0].get('cc'), [
          {
              'name': 'somebody@chromium.org'
          },
          {
              'name': 'nobody@chromium.org'
          },
      ],
      ignore_order=True)


def test_IssueTracker_NewBug_UsesExpectedParamsSansOwner(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  service._MakeCreateRequest = mocker.Mock()
  service.NewBug(
      'Bug title', 'body', cc=['somebody@chromium.org', 'nobody@chromium.org'])
  service._MakeCreateRequest.assert_called_with(
      {
          'title': 'Bug title',
          'summary': 'Bug title',
          'description': 'body',
          'labels': [],
          'components': [],
          'status': 'Unconfirmed',
          'cc': mocker.ANY,
          'projectId': 'chromium',
      }, 'chromium')
  assert not DeepDiff(
      service._MakeCreateRequest.call_args[0][0].get('cc'), [
          {
              'name': 'somebody@chromium.org'
          },
          {
              'name': 'nobody@chromium.org'
          },
      ],
      ignore_order=True)


def test_IssueTracker_MakeCommentRequest_UserCantOwn_RetryComment(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  error_content = {
      'error': {
          'message': 'Issue owner must be a project member',
          'code': 400
      }
  }
  service._ExecuteRequest = mocker.Mock(
      side_effect=errors.HttpError(
          mocker.Mock(return_value={'status': 404}),
          bytes(json.dumps(error_content), encoding='utf-8')))
  service.AddBugComment(12345, 'The comment', owner=['test@chromium.org'])
  assert service._ExecuteRequest.call_count == 2


def test_IssueTracker_MakeCommentRequest_UserDoesNotExist_RetryComment(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  error_content = {
      'error': {
          'message': 'The user does not exist: test@chromium.org',
          'code': 404
      }
  }
  service._ExecuteRequest = mocker.Mock(
      side_effect=errors.HttpError(
          mocker.Mock(return_value={'status': 404}),
          bytes(json.dumps(error_content), encoding='utf-8')))
  service.AddBugComment(
      12345,
      'The comment',
      cc_list=['test@chromium.org'],
      owner=['test@chromium.org'])
  assert service._ExecuteRequest.call_count == 2


def test_IssueTracker_MakeCommentRequest_IssueDeleted_ReturnsTrue(mocker):
  service = issue_tracker_service.IssueTrackerService(mocker.MagicMock())
  error_content = {
      'error': {
          'message': 'User is not allowed to view this issue 12345',
          'code': 403
      }
  }
  service._ExecuteRequest = mocker.Mock(
      side_effect=errors.HttpError(
          mocker.Mock(return_value={'status': 403}),
          bytes(json.dumps(error_content), encoding='utf-8')))
  comment_posted = service.AddBugComment(
      12345, 'The comment', owner='test@chromium.org')
  assert service._ExecuteRequest.call_count == 1
  assert comment_posted
