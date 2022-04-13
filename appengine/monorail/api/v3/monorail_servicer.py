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
import time
import sys

from google.oauth2 import id_token
from google.auth.transport import requests as google_requests

from google.appengine.api import oauth
from google.appengine.api import users
from google.appengine.api import app_identity
from google.protobuf import json_format
from components.prpc import codes
from components.prpc import server

from framework import monitoring

import settings
from api.v3 import converters
from framework import authdata
from framework import exceptions
from framework import framework_constants
from framework import monitoring
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
# Domain for service account emails.
SERVICE_ACCOUNT_DOMAIN = 'gserviceaccount.com'


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
    self.converter = None

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
      exceptions for permission checks and input validation that are
      raised in the Do* method are converted into pRPC status codes.
    """
    start_time = start_time or time.time()
    cnxn = cnxn or sql.MonorailConnection()
    if self.services.cache_manager:
      self.services.cache_manager.DoDistributedInvalidation(cnxn)

    response = None
    requester_auth = None
    metadata = dict(prpc_context.invocation_metadata())
    mc = monorailcontext.MonorailContext(self.services, cnxn=cnxn, perms=perms)
    try:
      self.AssertBaseChecks(request, metadata)
      client_id, requester_auth = self.GetAndAssertRequesterAuth(
          cnxn, metadata, self.services)
      logging.info('request proto is:\n%r\n', request)
      logging.info('requester is %r', requester_auth.email)
      monitoring.IncrementAPIRequestsCount(
          'v3',
          client_id,
          client_email=requester_auth.email,
          handler=handler.func_name)

      # TODO(crbug.com/monorail/8161)We pass in a None client_id for rate
      # limiting because CheckStart and CheckEnd will track and limit requests
      # per email and client_id separately.
      # So if there are many site users one day, we may end up rate limiting our
      # own site. With a None client_id we are only rate limiting by emails.
      if self.rate_limiter:
        self.rate_limiter.CheckStart(None, requester_auth.email, start_time)
      mc.auth = requester_auth
      if not perms:
        # NOTE(crbug/monorail/7614): We rely on servicer methods to call
        # to call LookupLoggedInUserPerms() with a project when they need to.
        mc.LookupLoggedInUserPerms(None)

      self.converter = converters.Converter(mc, self.services)
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
            None, requester_auth.email, end_time, start_time)
      self.RecordMonitoringStats(start_time, request, response, prpc_context)

    return response

  def CheckIDToken(self, cnxn, metadata):
    # type: (MonorailConnection, Mapping[str, str])
    #     -> Tuple[Optional[str], Optional[authdata.AuthData]]
    """Authenticate user from an ID token.

    Args:
      cnxn: connection to the SQL database.
      metadata: metadata sent by the client.

    Returns:
      The audience (AKA client_id) and a new AuthData object representing
      the user making the request or (None, None) if no ID token was found.

    Raises:
      permissions.PermissionException: If the token is invalid, the client ID
        is not allowlisted, or no user email was found in the ID token.
    """
    bearer = metadata.get('authorization')
    if not bearer:
      return None, None
    if bearer.lower().startswith('bearer '):
      token = bearer[7:]
    else:
      raise permissions.PermissionException('Invalid authorization token.')
    # TODO(crbug.com/monorail/7724): Use cachecontrol module to cache
    # certification used for verification.
    request = google_requests.Request()

    try:
      id_info = id_token.verify_oauth2_token(token, request)
      logging.info('ID token info: %r' % id_info)
    except ValueError:
      raise permissions.PermissionException(
          'Invalid bearer token.')

    audience = id_info['aud']
    email = id_info.get('email')
    if not email:
      raise permissions.PermissionException(
          'No email found in token info. '
          'Make sure requests are made with scopes `openid` and `email`')

    auth_client_ids, service_account_emails = (
        client_config_svc.GetClientConfigSvc().GetClientIDEmails())

    if email.endswith(SERVICE_ACCOUNT_DOMAIN):
      # For service accounts, the email must be allowlisted to call the
      # API and we must confirm that the ID token was meant for
      # Monorail by checking the audience.

      # An API call to any <version>-dot-<service>-dot-<app_id>.appspot.com
      # must have token audience of `https://<app_id>.appspot.com`
      app_id = app_identity.get_application_id()  # e.g. 'monorail-prod'
      host = 'https://%s.appspot.com' % app_id
      if audience != host:
        raise permissions.PermissionException(
            'Invalid token audience: %s.' % audience)
      if email not in service_account_emails:
        raise permissions.PermissionException(
            'Account %s is not allowlisted' % email)
    else:
      # For users, the audience is the client_id of the site used to make
      # the call to Monorail's API. The client_id must be allow-listed.
      if audience not in auth_client_ids:
        raise permissions.PermissionException(
            'Client %s is not allowlisted' % audience)

    # We must confirm the client/email is allowlisted before we
    # potentially auto-create the user account in Monorail.
    return audience, authdata.AuthData.FromEmail(
        cnxn, email, self.services, autocreate=True)

  def GetAndAssertRequesterAuth(self, cnxn, metadata, services):
    # type: (MonorailConnection, Mapping[str, str], Services ->
    #    Tuple[str, authdata.AuthData]
    """Gets the requester identity and checks if the user has permission
       to make the request.
       Any users successfully authenticated with oauth must be allowlisted or
       have accounts with the domains in api_allowed_email_domains.
       Users identified using cookie-based auth must have valid XSRF tokens.
       Test accounts ending with @example.com are only allowed in the
       local_mode.

    Args:
      cnxn: connection to the SQL database.
      metadata: metadata sent by the client.
      services: connections to backend services.

    Returns:
      The client ID and a new AuthData object representing a signed in or
      anonymous user.

    Raises:
      exceptions.NoSuchUserException: If the requester does not exist
      permissions.BannedUserException: If the user has been banned from the site
      permissions.PermissionException: If the user is not authorized with the
        Monorail scope, is not allowlisted, and has an invalid token.
    """
    # TODO(monorail:6538): Move different authentication methods into separate
    # functions.
    requester_auth = None
    client_id = None
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

    # Oauth2 ID token auth.
    if not requester_auth:
      client_id, requester_auth = self.CheckIDToken(cnxn, metadata)

    if client_id is None:
      # TODO(crbug.com/monorail/8160): For site users, we temporarily use
      # the host as the client_id, until we implement auth in the frontend
      # to make API requests with ID tokens that include client_ids.
      client_id = 'https://%s.appspot.com' % app_identity.get_application_id()


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

    return (client_id, requester_auth)

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
      details = 'The issue does not exist.'
      if e.message:
        details = cgi.escape(e.message, quote=True)
      prpc_context.set_details(details)
    elif exc_type == exceptions.NoSuchIssueApprovalException:
      prpc_context.set_code(codes.StatusCode.NOT_FOUND)
      prpc_context.set_details('The issue approval does not exist.')
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
      prpc_context.set_code(codes.StatusCode.ALREADY_EXISTS)
      prpc_context.set_details('The user group already exists.')
    elif exc_type == features_svc.HotlistAlreadyExists:
      prpc_context.set_code(codes.StatusCode.ALREADY_EXISTS)
      prpc_context.set_details('A hotlist with that name already exists.')
    elif exc_type == exceptions.ComponentDefAlreadyExists:
      prpc_context.set_code(codes.StatusCode.ALREADY_EXISTS)
      prpc_context.set_details('A component with that path already exists.')
    elif exc_type == exceptions.ActionNotSupported:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details('Requested action not supported.')
    elif exc_type == exceptions.InvalidComponentNameException:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details('That component name is invalid.')
    elif exc_type == exceptions.FilterRuleException:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details('Violates filter rule that should error.')
    elif exc_type == exceptions.InputException:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details(
         'Invalid arguments: %s' % cgi.escape(e.message, quote=True))
    elif exc_type == exceptions.OverAttachmentQuota:
      prpc_context.set_code(codes.StatusCode.RESOURCE_EXHAUSTED)
      prpc_context.set_details(
          'The request would exceed the attachment quota limit.')
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
    elif exc_type == exceptions.PageTokenException:
      prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
      prpc_context.set_details(
          'Page token invalid or incorrect for the accompanying request')
    else:
      prpc_context.set_code(codes.StatusCode.INTERNAL)
      prpc_context.set_details('Potential programming error.')
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

    fields = monitoring.GetCommonFields(
        # pRPC uses its own statuses, but we report HTTP status codes.
        ConvertPRPCStatusToHTTPStatus(prpc_context),
        # Use the API name, not the request path, to prevent an explosion in
        # possible field values.
        'monorail.v3.' + method_name)
    monitoring.AddServerDurations(elapsed_ms, fields)
    monitoring.IncrementServerResponseStatusCount(fields)
    monitoring.AddServerRequesteBytes(
        len(json_format.MessageToJson(request)), fields)
    response_length = 0
    if response:
      response_length = len(json_format.MessageToJson(response))
      monitoring.AddServerResponseBytes(response_length, fields)
