# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import logging

from gae_libs.handlers.base_handler import BaseHandler, Permission
from google.appengine.ext import ndb

from findit_v2.model.culprit_action import CulpritAction
from findit_v2.services.projects import GetProjectAPI


class AsyncAction(BaseHandler):
  """Performs a culprit action asynchronously."""
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandlePost(self):
    # Args are a json-encoded dict sent as the push task's payload.
    try:
      message = self._RunAction(json.loads(self.request.body))
    except RuntimeError as rte:
      message = rte.message
      raise
    finally:
      if message:
        logging.error(message)

  def _RunAction(self, task_args):
    try:
      project_api = GetProjectAPI(task_args['project'])
      culprit = ndb.Key(urlsafe=task_args['culprit_key']).get()
      if task_args['action'] == 'NotifyCulprit':
        return self._Notify(project_api, task_args, culprit)
      elif task_args['action'] == 'RequestReview':
        return self._RequestReview(project_api, task_args, culprit)
      elif task_args['action'] == 'CommitRevert':
        return self._CommitRevert(project_api, task_args, culprit)
      return 'Unknown action %s' % task_args['action']
    except KeyError as ke:
      return 'Push task is missing required argument: %s' % ke.message

  def _Notify(self, project_api, task_args, culprit):
    success = project_api.gerrit_actions.NotifyCulprit(
        culprit,
        task_args['message'],
        silent_notification=task_args['silent_notification'])
    if not success:
      raise RuntimeError('Notification failed')

  def _RequestReview(self, project_api, task_args, culprit):
    revert = project_api.gerrit_actions.CreateRevert(
        culprit, task_args['revert_description'])
    if not revert:
      raise RuntimeError('Revert creation failed')
    logging.info('Requesting revert %s to be reviewed', revert['id'])
    success = project_api.gerrit_actions.RequestReview(
        revert, task_args['request_review_message'])
    if not success:
      raise RuntimeError('Requesting revert review failed')
    self._Save(culprit, revert, False)

  def _CommitRevert(self, project_api, task_args, culprit):
    revert = project_api.gerrit_actions.CreateRevert(
        culprit, task_args['revert_description'])
    if not revert:
      raise RuntimeError('Revert creation failed')
    logging.info('Submitting revert %s', revert['id'])
    success = project_api.gerrit_actions.CommitRevert(
        revert, task_args['request_confirmation_message'])
    if not success:
      raise RuntimeError('Submitting revert failed')
    self._Save(culprit, revert, True)
    return None

  @ndb.transactional
  def _Save(self, culprit, revert, committed):
    action = CulpritAction.CreateKey(culprit).get()
    if not action.revert_change:
      action.revert_change = revert['id']
      action.revert_committed = committed
      action.put()
    else:
      logging.warning(
          'Possible duplicate revert for culrpit %s.'
          'We created %s, but datastore says %s already exists.', culprit,
          revert['id'], action.revert_change)
