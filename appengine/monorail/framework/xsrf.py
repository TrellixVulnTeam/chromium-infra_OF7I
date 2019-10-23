# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Utility routines for avoiding cross-site-request-forgery."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import base64
import hmac
import logging
import time

# This is a file in the top-level directory that you must edit before deploying
import settings
from framework import framework_constants
from services import secrets_svc

# This is how long tokens are valid.
TOKEN_TIMEOUT_SEC = 2 * framework_constants.SECS_PER_HOUR

# The token refresh servlet accepts old tokens to generate new ones, but
# we still impose a limit on how old they can be.
REFRESH_TOKEN_TIMEOUT_SEC = 10 * framework_constants.SECS_PER_DAY

# When the JS on a page decides whether or not it needs to refresh the
# XSRF token before submitting a form, there could be some clock skew,
# so we subtract a little time to avoid having the JS use an existing
# token that the server might consider expired already.
TOKEN_TIMEOUT_MARGIN_SEC = 5 * framework_constants.SECS_PER_MINUTE

# When checking that the token is not from the future, allow a little
# margin for the possibliity that the clock of the GAE instance that
# generated the token could be a little ahead of the one checking.
CLOCK_SKEW_SEC = 5

# Form tokens and issue stars are limited to only work with the specific
# servlet path for the servlet that processes them.  There are several
# XHR handlers that mainly read data without making changes, so we just
# use 'xhr' with all of them.
XHR_SERVLET_PATH = 'xhr'


DELIMITER = ':'


def GenerateToken(user_id, servlet_path, token_time=None):
  """Return a security token specifically for the given user.

  Args:
    user_id: int user ID of the user viewing an HTML form.
    servlet_path: string URI path to limit the use of the token.
    token_time: Time at which the token is generated in seconds since the epoch.

  Returns:
    A url-safe security token.  The token is a string with the digest
    the user_id and time, followed by plain-text copy of the time that is
    used in validation.

  Raises:
    ValueError: if the XSRF secret was not configured.
  """
  token_time = token_time or int(time.time())
  digester = hmac.new(secrets_svc.GetXSRFKey())
  digester.update(str(user_id))
  digester.update(DELIMITER)
  digester.update(servlet_path)
  digester.update(DELIMITER)
  digester.update(str(token_time))
  digest = digester.digest()

  token = base64.urlsafe_b64encode('%s%s%d' % (digest, DELIMITER, token_time))
  return token


def ValidateToken(
  token, user_id, servlet_path, timeout=TOKEN_TIMEOUT_SEC):
  """Return True if the given token is valid for the given scope.

  Args:
    token: String token that was presented by the user.
    user_id: int user ID.
    servlet_path: string URI path to limit the use of the token.

  Raises:
    TokenIncorrect: if the token is missing or invalid.
  """
  if not token:
    raise TokenIncorrect('missing token')

  try:
    decoded = base64.urlsafe_b64decode(str(token))
    token_time = int(decoded.split(DELIMITER)[-1])
  except (TypeError, ValueError):
    raise TokenIncorrect('could not decode token')
  now = int(time.time())

  # The given token should match the generated one with the same time.
  expected_token = GenerateToken(user_id, servlet_path, token_time=token_time)
  if len(token) != len(expected_token):
    raise TokenIncorrect('presented token is wrong size')

  # Perform constant time comparison to avoid timing attacks
  different = 0
  for x, y in zip(token, expected_token):
    different |= ord(x) ^ ord(y)
  if different:
    raise TokenIncorrect(
        'presented token does not match expected token: %r != %r' % (
            token, expected_token))

  # We reject tokens from the future.
  if token_time > now + CLOCK_SKEW_SEC:
    raise TokenIncorrect('token is from future')

  # We check expiration last so that we only raise the expriration error
  # if the token would have otherwise been valid.
  if now - token_time > timeout:
    raise TokenIncorrect('token has expired')


def TokenExpiresSec():
  """Return timestamp when current tokens will expire, minus a safety margin."""
  now = int(time.time())
  return now + TOKEN_TIMEOUT_SEC - TOKEN_TIMEOUT_MARGIN_SEC


class Error(Exception):
  """Base class for errors from this module."""
  pass


# Caught separately in servlet.py
class TokenIncorrect(Error):
  """The POST body has an incorrect URL Command Attack token."""
  pass
