# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Functions that prepare and send email notifications of issue changes."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import


import logging

from features import features_constants
from framework import cloud_tasks_helpers
from framework import framework_constants
from framework import urls
from tracker import tracker_bizobj


def PrepareAndSendIssueChangeNotification(
    issue_id, hostport, commenter_id, send_email=True,
    old_owner_id=framework_constants.NO_USER_SPECIFIED, comment_id=None):
  """Create a task to notify users that an issue has changed.

  Args:
    issue_id: int ID of the issue that was changed.
    hostport: string domain name and port number from the HTTP request.
    commenter_id: int user ID of the user who made the comment.
    send_email: True if email notifications should be sent.
    old_owner_id: optional user ID of owner before the current change took
      effect. They will also be notified.
    comment_id: int Comment ID of the comment that was entered.

  Returns nothing.
  """
  if old_owner_id is None:
    old_owner_id = framework_constants.NO_USER_SPECIFIED
  params = dict(
      issue_id=issue_id, commenter_id=commenter_id, comment_id=comment_id,
      hostport=hostport, old_owner_id=old_owner_id, send_email=int(send_email))
  task = cloud_tasks_helpers.generate_simple_task(
      urls.NOTIFY_ISSUE_CHANGE_TASK + '.do', params)
  cloud_tasks_helpers.create_task(
      task, queue=features_constants.QUEUE_NOTIFICATIONS)

  task = cloud_tasks_helpers.generate_simple_task(
      urls.PUBLISH_PUBSUB_ISSUE_CHANGE_TASK + '.do', params)
  cloud_tasks_helpers.create_task(task, queue=features_constants.QUEUE_PUBSUB)


def PrepareAndSendIssueBlockingNotification(
    issue_id, hostport, delta_blocker_iids, commenter_id, send_email=True):
  """Create a task to follow up on an issue blocked_on change."""
  if not delta_blocker_iids:
    return  # No notification is needed

  params = dict(
      issue_id=issue_id, commenter_id=commenter_id, hostport=hostport,
      send_email=int(send_email),
      delta_blocker_iids=','.join(str(iid) for iid in delta_blocker_iids))

  task = cloud_tasks_helpers.generate_simple_task(
      urls.NOTIFY_BLOCKING_CHANGE_TASK + '.do', params)
  cloud_tasks_helpers.create_task(
      task, queue=features_constants.QUEUE_NOTIFICATIONS)


def PrepareAndSendApprovalChangeNotification(
    issue_id, approval_id, hostport, comment_id, send_email=True):
  """Create a task to follow up on an approval change."""

  params = dict(
      issue_id=issue_id, approval_id=approval_id, hostport=hostport,
      comment_id=comment_id, send_email=int(send_email))

  task = cloud_tasks_helpers.generate_simple_task(
      urls.NOTIFY_APPROVAL_CHANGE_TASK + '.do', params)
  cloud_tasks_helpers.create_task(
      task, queue=features_constants.QUEUE_NOTIFICATIONS)


def SendIssueBulkChangeNotification(
    issue_ids, hostport, old_owner_ids, comment_text, commenter_id,
    amendments, send_email, users_by_id):
  """Create a task to follow up on an issue blocked_on change."""
  amendment_lines = []
  for up in amendments:
    line = '    %s: %s' % (
        tracker_bizobj.GetAmendmentFieldName(up),
        tracker_bizobj.AmendmentString(up, users_by_id))
    if line not in amendment_lines:
      amendment_lines.append(line)

  params = dict(
      issue_ids=','.join(str(iid) for iid in issue_ids),
      commenter_id=commenter_id, hostport=hostport, send_email=int(send_email),
      old_owner_ids=','.join(str(uid) for uid in old_owner_ids),
      comment_text=comment_text, amendments='\n'.join(amendment_lines))

  task = cloud_tasks_helpers.generate_simple_task(
      urls.NOTIFY_BULK_CHANGE_TASK + '.do', params)
  cloud_tasks_helpers.create_task(
      task, queue=features_constants.QUEUE_NOTIFICATIONS)


def PrepareAndSendDeletedFilterRulesNotification(
    project_id, hostport, filter_rule_strs):
  """Create a task to notify project owners of deleted filter rules."""

  params = dict(
      project_id=project_id, filter_rules=','.join(filter_rule_strs),
      hostport=hostport)

  task = cloud_tasks_helpers.generate_simple_task(
      urls.NOTIFY_RULES_DELETED_TASK + '.do', params)
  cloud_tasks_helpers.create_task(
      task, queue=features_constants.QUEUE_NOTIFICATIONS)
