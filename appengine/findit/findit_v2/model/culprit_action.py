# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import datetime

from google.appengine.ext import ndb


class CulpritAction(ndb.Model):

  # Each action is related to exactly one culprit. Its key is:
  # ['Culprit', <Culprit Key>, 'CulpritAction', DEFAULT_ID]
  DEFAULT_ID = 1

  # Values for action_type.
  REVERT = 1
  CULPRIT_NOTIFIED = 2

  VALID_ACTION_TYPES = [
      REVERT,
      CULPRIT_NOTIFIED,
  ]

  action_type = ndb.IntegerProperty(required=True, choices=VALID_ACTION_TYPES)
  revert_committed = ndb.BooleanProperty(required=True, default=False)
  # Change number. See GetRevertURL below for how it's used.
  revert_change = ndb.IntegerProperty(indexed=False, required=False)
  create_timestamp = ndb.DateTimeProperty(required=True, auto_now_add=True)

  @classmethod
  def Create(cls,
             culprit,
             action_type,
             revert_committed=False,
             revert_change=None):
    return cls(
        key=cls.CreateKey(culprit),
        action_type=action_type,
        revert_committed=revert_committed,
        revert_change=revert_change)

  @classmethod
  def CreateKey(cls, culprit):
    assert culprit
    return ndb.Key(cls, cls.DEFAULT_ID, parent=culprit.key)

  @classmethod
  def GetRecentActionsByType(cls,
                             action_type,
                             revert_committed=False,
                             window=datetime.timedelta(days=1)):
    """Gets the actions of the given type taken most recently (24 hr default).

    Args:
      action_type: One of {CulrpitAction.REVERT, CulpritAction.CULPRIT_NOTIFIED}
      revert_committed (bool): In the case of CulpritAction.REVERT whether the
          revert was automatically committed.
      window (datetime.timedelta): How far back to look for actions.
    """
    return cls.query(
        cls.action_type == action_type,
        cls.revert_committed == revert_committed,
        cls.create_timestamp > datetime.datetime.utcnow() - window).fetch()

  def GetRevertURL(self):
    if not self.revert_change:
      return None
    culprit = self.key.parent().get()
    host, domain = culprit.gitiles_host.split('.', 1)
    return 'https://{gerrit_host}/c/{project}/+/{change_number}'.format(
        gerrit_host='.'.join([host + '-review', domain]),
        project=culprit.gitiles_project,
        change_number=self.revert_change,
    )
