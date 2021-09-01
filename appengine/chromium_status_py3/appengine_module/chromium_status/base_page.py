# Copyright (c) 2012 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Utility base class."""

import datetime
import hashlib
import logging
import os
import re

from flask import make_response, render_template, request
from google.cloud import ndb

from appengine_module.chromium_status import utils

ndb_client = ndb.Client()


class Passwords(ndb.Model):
  """Super users. Useful for automated scripts."""
  password_sha1 = ndb.StringProperty(required=True)


class GlobalConfig(ndb.Model):
  """Instance-specific config like application name."""
  app_name = ndb.StringProperty(required=True)
  # Flag indicating that anonymous viewing is possible.
  public_access = ndb.BooleanProperty()
  # Flag indicating that this is a ChromiumOS status page
  is_chromiumos = ndb.BooleanProperty(default=False)
  # Preamble text to appear directly above status-editing field
  preamble = ndb.StringProperty(required=False)
  # Postamble text to appear directly beneath status-editing field
  postamble = ndb.StringProperty(required=False)


class BasePage:
  """Utility functions needed to validate user and display a template."""
  # Initialized in bootstrap(), which is called serially at process startup.
  APP_NAME = ''
  _VALID_PUBLIC_EMAIL = re.compile(
      r"""
      ^(
        .*@chromium\.org|
        .*@google\.com|
        luci-notify@appspot.gserviceaccount.com
      )$""", re.VERBOSE)
  _VALID_PRIVATE_EMAIL = re.compile(
      r"""
      ^(
        .*@google\.com|
        luci-notify@appspot.gserviceaccount.com
      )$""", re.VERBOSE)
  PUBLIC_ACCESS = False
  IS_CHROMIUMOS = False
  PREAMBLE = ''
  POSTAMBLE = ''

  def __init__(self, *args, **kwargs):  # pragma: no cover
    super(BasePage, self).__init__(*args, **kwargs)
    self._initialized = False
    # Read and write access mean to the datastore.
    # Bot access is specifically required (in addition to write access) for
    # some queries that are allowed to specify a username synthetically.
    # TODO (crbug.com/1121016): Implement user authentication and check for ACL
    # For now, just set read_access = True for testing
    self._read_access = True
    self._write_access = False
    self._bot_login = False
    self._user = None

  def _late_init(self):  # pragma: no cover
    """Initializes access control fields once the object is setup."""
    # TODO (crbug.com/1121016): Implement user authentication and check for ACL
    pass
    # def look_for_password():
    #   """Looks for password parameter. Not awesome."""
    #   password = self.request.get('password')
    #   if password:
    #     sha1_pass = hashlib.sha1(password).hexdigest()
    #     if Passwords.gql('WHERE password_sha1 = :1', sha1_pass).get():
    #       # The password is valid, this is a super admin.
    #       self._write_access = True
    #       self._read_access = True
    #       self._bot_login = True
    #     else:
    #       if utils.is_dev_env() and password == 'foobar':
    #         # Dev server is unsecure.
    #         self._read_access = True
    #         self._write_access = True
    #         self._bot_login = True
    #       else:
    #         logging.error('Password is invalid')

    # self._user = users.get_current_user()
    # if utils.is_dev_env():
    #   look_for_password()
    #   # Maybe the tests reloaded our public settings ...
    #   self.PUBLIC_ACCESS = GlobalConfig.query().get().public_access
    # elif not self._user:
    #   try:
    #     self._user = oauth.get_current_user()
    #   except oauth.OAuthRequestError:
    #     if self.request.scheme == 'https':
    #       look_for_password()

    # if not self._write_access and self._user:
    #   if self.PUBLIC_ACCESS:
    #     valid_email = self._VALID_PUBLIC_EMAIL
    #   else:
    #     valid_email = self._VALID_PRIVATE_EMAIL
    #   self._write_access = bool(
    #       users.is_current_user_admin() or
    #       valid_email.match(self._user.email()))
    # if self.PUBLIC_ACCESS:
    #   self._read_access = True
    # else:
    #   self._read_access = self._write_access

    # self._initialized = True
    # logging.info('ReadAccess: %r, WriteAccess: %r, BotLogin: %r, User: %s' % (
    #     self._read_access, self._write_access, self._bot_login, self._user))

  @property
  def write_access(self):  # pragma: no cover
    if not self._initialized:
      self._late_init()
    return self._write_access

  @property
  def read_access(self):  # pragma: no cover
    if not self._initialized:
      self._late_init()
    return self._read_access

  @property
  def user(self):  # pragma: no cover
    if not self._initialized:
      self._late_init()
    return self._user

  @property
  def bot_login(self):  # pragma: no cover
    if not self._initialized:
      self._late_init()
    return self._bot_login

  def InitializeTemplate(self, title):  # pragma: no cover
    """Initializes the template values with information needed by all pages."""
    # TODO (crbug.com/1121016): Implement user authentication and get user email
    user_email = ''
    template_values = {
        'app_name': self.APP_NAME,
        'username': user_email,
        'title': title,
        'current_UTC_time': datetime.datetime.now(),
        'write_access': self.write_access,
        'user': None,
    }
    return template_values

  # pylint: disable=unused-argument
  def DisplayTemplate(self, name, template_values,
                      use_cache=False):  # pragma: no cover
    """Replies to a http request with a template.

    Optionally cache it for 1 second. Only to be used for user-invariant
    pages!
    """
    r = make_response(render_template(name, **template_values))
    r.headers.set('Cache-Control', 'no-cache, private, max-age=0')
    return r


def bootstrap():  # pragma: no cover
  app_name = "Chromium"
  with ndb_client.context():
    config = GlobalConfig.query().get()
  if config is None:
    # Insert a dummy GlobalConfig so it can be edited through the admin
    # console
    config = GlobalConfig(app_name=app_name)
    config.public_access = False
    config.put()
  else:
    needs_update = False
    if not config.app_name:
      config.app_name = app_name
      needs_update = True
    if config.public_access is None:
      # Upgrade to public_access for existing waterfalls.
      config.public_access = True
      needs_update = True
    if needs_update:
      config.put()
  BasePage.APP_NAME = config.app_name
  BasePage.PUBLIC_ACCESS = config.public_access
  BasePage.IS_CHROMIUMOS = config.is_chromiumos
  BasePage.PREAMBLE = config.preamble
  BasePage.POSTAMBLE = config.postamble

  with ndb_client.context():
    if Passwords.query().get() is None:
      # Insert a dummy Passwords so it can be edited through the admin console
      Passwords(password_sha1='invalidhash').put()
