# Copyright (c) 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Git LKGR management webpages."""

import json
import logging

from flask import make_response, abort, request
from google.cloud import ndb

from appengine_module.chromium_status import utils
from appengine_module.chromium_status.base_page import BasePage

ndb_client = ndb.Client()


class Commit(ndb.Model):  # pylint: disable=W0232
  """Description of a commit, keyed by random integer IDs."""
  # Git hash of this commit. A property so it can be viewed in datastore.
  git_hash = ndb.StringProperty()
  # Git commit position for this commit (required for sorting).
  position_ref = ndb.StringProperty()
  position_num = ndb.IntegerProperty()
  # Time at which this commit was set as the LKGR.
  date = ndb.DateTimeProperty(auto_now_add=True)


class Commits(BasePage):
  """Displays the Git LKGR history page containing the last 100 LKGRs."""

  @utils.requires_read_access
  def get(self):
    """Returns information about the history of LKGR."""
    limit = min(int(request.args.get('limit', '100')), 1000)
    commits = Commit.query().order('-position_num').order('position_ref').fetch(
        limit)

    if request.args.get('format') == 'json':
      data = json.dumps([commit.AsDict() for commit in commits])
      r = make_response(data)
      r.headers['Content-Type'] = 'application/json'
      r.headers['Access-Control-Allow-Origin'] = '*'
      return r

    template_values = self.InitializeTemplate('Chromium Git LKGR History')
    page_value = {'commits': commits, 'limit': limit}
    template_values.update(page_value)
    return self.DisplayTemplate('commits.html', template_values)

  @utils.requires_write_access
  def post(self):
    """Adds a new revision status."""
    git_hash = request.args.get('hash')
    position_ref = request.args.get('position_ref')
    position_num = int(request.args.get('position_num'))
    if git_hash and position_ref and position_num:
      obj = Commit(
          git_hash=git_hash,
          position_ref=position_ref,
          position_num=position_num)
      obj.put()
    else:
      abort(400)


class LastKnownGoodRevisionGIT(BasePage):
  """Displays the /git-lkgr page."""

  @utils.requires_read_access
  def get(self):
    """Look for the latest successful revision and return it."""
    commit = Commit.query().order('-position_num').order('position_ref').get()
    if commit:
      r = make_response(commit.git_hash)
      r.headers['Cache-Control'] = 'no-cache, private, max-age=5'
      r.headers['Content-Type'] = 'text/plain'
      return r
    else:
      logging.error('OMG There\'s no git-lkgr!?')
      abort(404)


def bootstrap():
  with ndb_client.context():
    Commit.get_or_insert(
        'dummy-commit',
        git_hash='0' * 40,
        position_ref='refs/heads/main',
        position_num=0)
