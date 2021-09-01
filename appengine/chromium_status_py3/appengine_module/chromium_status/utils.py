# Copyright (c) 2011 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Utils."""

import os

from google.cloud import ndb


def is_dev_env():  # pragma: no cover
  """Returns True if we're running in the development environment."""
  return 'Dev' in os.environ.get('SERVER_SOFTWARE', '')


def requires_work_queue_login(func):  # pragma: no cover
  """Decorator that only allows a request if from cron job, task, or an admin.

  Also allows access if running in development server environment.

  Args:
    func: A webapp.RequestHandler method.

  Returns:
    Function that will return a 401 error if not from an authorized source.
  """

  def decorated(self, *args, **kwargs):
    if ('X-AppEngine-Cron' in self.request.headers or
        'X-AppEngine-TaskName' in self.request.headers or self.write_access):
      return func(self, *args, **kwargs)
    elif self.user is None:
      # TODO (crbug.com/1121016): redirect to login url
      pass
      # self.redirect(users.create_login_url(self.request.url))
    else:
      self.response.set_status(401)
      self.response.out.write('Handler only accessible for work queues')

  return decorated


def requires_bot_login(func):  # pragma: no cover
  """Allowed only when logged in via bot password. BasePage objects only."""

  def decorated(self, *args, **kwargs):
    if self.bot_login:
      return func(self, *args, **kwargs)
    else:
      self.response.headers['Content-Type'] = 'text/plain'
      self.response.out.write('Forbidden')
      self.error(403)

  return decorated


def requires_write_access(func):  # pragma: no cover
  """Write access via login or bot password. BasePage objects only."""

  def decorated(self, *args, **kwargs):
    if self.write_access:
      return func(self, *args, **kwargs)
    else:
      self.response.headers['Content-Type'] = 'text/plain'
      self.response.out.write('Forbidden')
      self.error(403)

  return decorated


def requires_login(func):  # pragma: no cover
  """Must be logged in for access. BasePage objects only."""

  def decorated(self, *args, **kwargs):
    if self.user:
      return func(self, *args, **kwargs)
    elif not self.user:
      # TODO (crbug.com/1121016): redirect to login url
      return func(self, *args, **kwargs)
    else:
      self.response.headers['Content-Type'] = 'text/plain'
      self.response.out.write('Forbidden')
      self.error(403)

  return decorated


def requires_read_access(func):  # pragma: no cover
  """Read access via login or anonymous if public. BasePage objects only."""

  def decorated(self, *args, **kwargs):
    if self.read_access:
      return func(self, *args, **kwargs)
    elif not self.user:
      # TODO (crbug.com/1121016): redirect to login url
      pass
      # self.redirect(users.create_login_url(self.request.url))
    else:
      self.response.headers['Content-Type'] = 'text/plain'
      self.response.out.write('Forbidden')
      self.error(403)

  return decorated


def AsDict(self):  # pragma: no cover
  """Converts ndb Model to a dict."""
  ret = {}
  for key in self._properties:
    value = getattr(self, key)
    if isinstance(value, (int, None.__class__, float)):
      ret[key] = value
    else:
      ret[key] = str(value)
  key = self.key
  if key:
    ret['key'] = key.string_id() or key.integer_id()
    if key.parent():
      ret['parent_key'] = key.parent().string_id() or key.parent().integer_id()
  return ret


def bootstrap():  # pragma: no cover
  """Monkey patch db.Model.AsDict()"""
  ndb.Model.AsDict = AsDict
