# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import cgi
import functools
import logging
import sys
import time
from google.appengine.api import oauth

from google.appengine.api import users
from google.protobuf import json_format
from components.prpc import codes
from components.prpc import server

from infra_libs.ts_mon.common import http_metrics

import settings
from framework import authdata
from framework import exceptions
from framework import framework_bizobj
from framework import framework_constants
from framework import monorailcontext
from framework import ratelimiter
from framework import permissions
from framework import sql
from framework import xsrf
from services import client_config_svc
from services import features_svc


# Header for XSRF token to protect cookie-based auth users.
XSRF_TOKEN_HEADER = 'x-xsrf-token'
# Header for test account email.  Only accepted for local dev server.
TEST_ACCOUNT_HEADER = 'x-test-account'
# Optional header to help us understand why certain calls were made.
REASON_HEADER = 'x-reason'
# Optional header to help prevent double updates.
REQUEST_ID_HEADER = 'x-request-id'


def ConvertPRPCStatusToHTTPStatus(context):
  """pRPC uses internal codes 0..16, but we want to report HTTP codes."""
  return server._PRPC_TO_HTTP_STATUS.get(context._code, 500)


def PRPCMethod(func):
  @functools.wraps(func)
  def wrapper(self, request, prpc_context, cnxn=None):
    return self.Run(
        func, request, prpc_context, cnxn=cnxn)

  wrapper.wrapped = func
  return wrapper


class MonorailServicer(object):
  """Abstract base class for API servicers.
  """

  def __init__(self, services, make_rate_limiter=True, xsrf_timeout=None):
    self.services = services
    if make_rate_limiter:
      self.rate_limiter = ratelimiter.ApiRateLimiter()
    else:
      self.rate_limiter = None
    # We allow subclasses to specify a different timeout. This allows the
    # RefreshToken method to check the token with a longer expiration and
    # generate a new one.
    self.xsrf_timeout = xsrf_timeout or xsrf.TOKEN_TIMEOUT_SEC

  def Run(
      self, handler, request, prpc_context,
      cnxn=None, perms=None, start_time=None, end_time=None):
    """Run a Do* method in an API context.

    Args:
      handler: API handler method to call with MonorailContext and request.
      request: API Request proto object.
      prpc_context: pRPC context object with status code.
      cnxn: Optional connection to SQL database.
      perms: PermissionSet passed in during testing.
      start_time: Int timestamp passed in during testing.
      end_time: Int timestamp passed in during testing.

    Returns:
      The response proto returned from the handler or None if that
      method raised an exception that we handle.

    Raises:
      Only programming errors should be raised as exceptions.  All
      execptions for permission checks and input validation that are
      raised in the Do* method are converted into pRPC status codes.
    """
    start_time = start_time or time.time()
    cnxn = cnxn or sql.MonorailConnection()
    if self.services.cache_manager:
      self.services.cache_manager.DoDistributedInvalidation(cnxn)

    response = None
    client_id = None  # TODO(jrobbins): consider using client ID.
    requester_auth = None
    metadata = dict(prpc_context.invocation_metadata())
    mc = monorailcontext.MonorailContext(self.services, cnxn=cnxn, perms=perms)
    try:
      self.AssertBaseChecks(request, metadata)
      requester_auth = self.GetAndAssertRequesterAuth(
          cnxn, metadata, self.services)
      logging.info('request proto is:\n%r\n', requester_auth.email)
      logging.info('requester is %r', requester_auth.email)

      if self.rate_limiter:
        self.rate_limiter.CheckStart(
            client_id, requester_auth.email, start_time)
      mc.auth = requester_auth
      if not perms:
        mc.LookupLoggedInUserPerms(self.GetRequestProject(mc.cnxn, request))
      response = handler(self, mc, request)

    except Exception as e:
      if not self.ProcessException(e, prpc_context, mc):
        raise e.__class__, e, sys.exc_info()[2]
    finally:
      if mc:
        mc.CleanUp()
      if self.rate_limiter and requester_auth and requester_auth.email:
        end_time = end_time or time.time()
        self.rate_limiter.CheckEnd(
            client_id, requester_auth.email, end_time, start_time)
      self.RecordMonitoringStats(start_time, request, response, prpc_context)

    return response

  def GetAndAssertRequesterAuth(self, cnxn, metadata, services):
    """Gets the requester identity and checks if the user has permission
       to make the request.
       Any users successfully authenticated with oauth must be whitelisted.
       Users identified using cookie-based auth must have valid XSRF tokens.
       Test accounts ending with @example.com are only allowed in the
       local_mode.

    Args:
      cnxn: connection to the SQL database.
      metadata: metadata sent by the client.
      services: connections to backend services.

    Returns:
      A new AuthData object representing a signed in or anonymous user.

    Raises:
      exceptions.NoSuchUserException: If the requester does not exist
      permissions.BannedUserException: If the user has been banned from the site
      permissions.PermissionException: If the user is not authorized with the
        Monorail scope, is not whitelisted, and has an invalid token.
    """
    requester_auth = None
    # When running on localhost, allow request to specify test account.
    if TEST_ACCOUNT_HEADER in metadata:
      if not settings.local_mode:
        raise exceptions.InputException(
            'x-test-account only accepted in local_mode')
      # For local development, we accept any request.
      # TODO(jrobbins): make this more realistic by requiring a fake XSRF token.
      test_account = metadata[TEST_ACCOUNT_HEADER]
      if not test_account.endswith('@example.com'):
        raise exceptions.InputException(
            'test_account must end with @example.com')
      logging.info('Using test_account: %r' % test_account)
      requester_auth = authdata.AuthData.FromEmail(cnxn, test_account, services)

    # TODO(jojwang): Oauth using Monorail's scope
    # Oauth for whitelisted users
    if not requester_auth:
      try:
        client_id = oauth.get_client_id(framework_constants.OAUTH_SCOPE)
        user = oauth.get_current_user(framework_constants.OAUTH_SCOPE)
        if user:
          auth_client_ids, auth_emails = (
              client_config_svc.GetClientConfigSvc().GetClientIDEmails())
          logging.info('Oauth requester %s', user.email())
          # Check if email or client_id is whitelisted
          if (user.email() in auth_emails) or (client_id in auth_client_ids):
            logging.info('Client %r is whitelisted', user.email())
            requester_auth = authdata.AuthData.FromEmail(
                cnxn, user.email(), services)
      except oauth.Error as ex:
        logging.info('Got oauth error: %r', ex)

    # Cookie-based auth for signed in and anonymous users.
    if not requester_auth:
      # Check for signed in user
      user = users.get_current_user()
      if user:
        logging.info('Using cookie user: %r', user.email())
        requester_auth = authdata.AuthData.FromEmail(
            cnxn, user.email(), services)
      else:
        # Create AuthData for anonymous user.
        requester_auth = authdata.AuthData.FromEmail(cnxn, None, services)

      # Cookie-based auth signed-in and anon users need to have the XSRF
      # token validate.
      try:
        token = metadata.get(XSRF_TOKEN_HEADER)
        xsrf.ValidateToken(
            token, requester_auth.user_id, xsrf.XHR_SERVLET_PATH,
            timeout=self.xsrf_timeout)
      except xsrf.TokenIncorrect:
        raise permissions.PermissionException(
            'Requester %s does not have permission to make this request.'
            % requester_auth.email)

    if permissions.IsBanned(requester_auth.user_pb, requester_auth.user_view):
      raise permissions.BannedUserException(
          'The user %s has been banned from using this site' %
          requester_auth.email)

    return requester_auth

  def AssertBaseChecks(self, request, metadata):
    """Reject requests that we refuse to serve."""
    # TODO(jrobbins): Add read_only check as an exception raised in sql.py.
    if (settings.read_only and
        not request.__class__.__name__.startswith(('Get', 'List'))):
      raise permissions.PermissionException(
          'This request is not allowed in read-only mode')

    if REASON_HEADER in metadata:
      logging.info('Request reason: %r', metadata[REASON_HEADER])
    if REQUEST_ID_HEADER in metadata:
      # TODO(jrobbins): Ignore requests with duplicate request_ids.
      logging.info('request_id: %r', metadata[REQUEST_ID_HEADER])

  def GetRequestProject(self, cnxn, request):
    """Return the Project business object that the user is viewing or None."""
    if hasattr(request, 'project_name'):
        project = self.services.project.GetProjectByName(
            cnxn, request.project_name)
        if not project:
          logging.info('Request has project_name: %r but it does not exist.',
                       request.project_name)
          return None
        return project
    else:
      return None

  def ProcessException(self, e, prpc_context, mc):
    """Return True if we convert an exception to a pRPC status code."""
    logging.exception(e)
    logging.info(e.message)
    exc_type = type(e)
    if exc_type == exceptions.NoSuchUserException:
      prpc_context.set_code(codes.StatusCode.NOT_FOUND)
      prpc_context.set_details('The user does not exist.')
    elif exc_type == exceptions.NoSuchProjectException:
      prpc_context.set_code(codes.StatusCode.NOT_FOUND)
      prpc_context.set_details('The project does not exist.')
    elif exc_type == exceptions.NoSuchTemplateException:
      prpc_context.set_code(codes.StatusCode.NOT_FOUND)
      prpc_context.set_details('The template does not exist.')
    elif exc_type == exceptions.NoSuchIssueException:
      prpc_context.set_code(codes.StatusCode.NOT_FOUND)
      prpc_context.set_details('The issue does not exist.')
    elif exc_type == exceptions.NoSuchCommentException:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details('No such comment')
    elif exc_type == exceptions.NoSuchComponentException:
      prpc_context.set_code(codes.StatusCode.NOT_FOUND)
      prpc_context.set_details('The component does not exist.')
    elif exc_type == permissions.BannedUserException:
      prpc_context.set_code(codes.StatusCode.PERMISSION_DENIED)
      prpc_context.set_details('The requesting user has been banned.')
    elif exc_type == permissions.PermissionException:
      logging.info('perms is %r', mc.perms)
      prpc_context.set_code(codes.StatusCode.PERMISSION_DENIED)
      prpc_context.set_details('Permission denied.')
    elif exc_type == exceptions.GroupExistsException:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details('The user group already exists.')
    elif exc_type == features_svc.HotlistAlreadyExists:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details('A hotlist with that name already exists.')
    elif exc_type == exceptions.InvalidComponentNameException:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details('That component name is invalid.')
    elif exc_type == exceptions.InputException:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details(
         'Invalid arguments: %s' % cgi.escape(e.message, quote=True))
    elif exc_type == ratelimiter.ApiRateLimitExceeded:
      prpc_context.set_code(codes.StatusCode.PERMISSION_DENIED)
      prpc_context.set_details('The requester has exceeded API quotas limit.')
    elif exc_type == oauth.InvalidOAuthTokenError:
      prpc_context.set_code(codes.StatusCode.UNAUTHENTICATED)
      prpc_context.set_details(
          'The oauth token was not valid or must be refreshed.')
    elif exc_type == xsrf.TokenIncorrect:
      logging.info('Bad XSRF token: %r', e.message)
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details('Bad XSRF token.')
    else:
      return False  # Re-raise any exception from programming errors.
    return True  # It if was one of the cases above, don't reraise.

  def RecordMonitoringStats(
      self, start_time, request, response, prpc_context, now=None):
    """Record monitoring info about this request."""
    now = now or time.time()
    elapsed_ms = int((now - start_time) * 1000)
    method_name = request.__class__.__name__
    if method_name.endswith('Request'):
      method_name = method_name[:-len('Request')]
    method_identifier = 'monorail.' + method_name
    fields = {
        # pRPC uses its own statuses, but we report HTTP status codes.
        'status': ConvertPRPCStatusToHTTPStatus(prpc_context),
        # Use the api name, not the request path, to prevent an
        # explosion in possible field values.
        'name': method_identifier,
        'is_robot': False,
        }

    http_metrics.server_durations.add(elapsed_ms, fields=fields)
    http_metrics.server_response_status.increment(fields=fields)
    http_metrics.server_request_bytes.add(
        len(json_format.MessageToJson(request)), fields=fields)
    response_size = 0
    if response:
      response_size = len(json_format.MessageToJson(response))
      http_metrics.server_response_bytes.add(response_size, fields=fields)
