# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for MonorailServicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import time
import unittest
import mock
import mox

from components.prpc import server
from components.prpc import codes
from components.prpc import context
from google.appengine.ext import testbed
from google.protobuf import json_format

import settings
from api import monorail_servicer
from framework import authdata
from framework import exceptions
from framework import monorailcontext
from framework import permissions
from framework import ratelimiter
from framework import xsrf
from services import cachemanager_svc
from services import config_svc
from services import service_manager
from services import features_svc
from testing import fake
from testing import testing_helpers


class MonorailServicerFunctionsTest(unittest.TestCase):

  def testConvertPRPCStatusToHTTPStatus(self):
    """We can convert pRPC status codes to http codes for monitoring."""
    prpc_context = context.ServicerContext()

    prpc_context.set_code(codes.StatusCode.OK)
    self.assertEqual(
        200, monorail_servicer.ConvertPRPCStatusToHTTPStatus(prpc_context))

    prpc_context.set_code(codes.StatusCode.INVALID_ARGUMENT)
    self.assertEqual(
        400, monorail_servicer.ConvertPRPCStatusToHTTPStatus(prpc_context))

    prpc_context.set_code(codes.StatusCode.PERMISSION_DENIED)
    self.assertEqual(
        403, monorail_servicer.ConvertPRPCStatusToHTTPStatus(prpc_context))

    prpc_context.set_code(codes.StatusCode.NOT_FOUND)
    self.assertEqual(
        404, monorail_servicer.ConvertPRPCStatusToHTTPStatus(prpc_context))

    prpc_context.set_code(codes.StatusCode.INTERNAL)
    self.assertEqual(
        500, monorail_servicer.ConvertPRPCStatusToHTTPStatus(prpc_context))


class UpdateSomethingRequest(testing_helpers.Blank):
  """A fake request that would do a write."""
  pass


class ListSomethingRequest(testing_helpers.Blank):
  """A fake request that would do a read."""
  pass


class TestableServicer(monorail_servicer.MonorailServicer):
  """Fake servicer class."""

  def __init__(self, services):
    super(TestableServicer, self).__init__(services)
    self.was_called = False
    self.seen_mc = None
    self.seen_request = None

  @monorail_servicer.PRPCMethod
  def CalcSomething(self, mc, request):
    """Raise the test exception, or return what we got for verification."""
    self.was_called = True
    self.seen_mc = mc
    self.seen_request = request
    assert mc
    assert request
    if request.exc_class:
      raise request.exc_class()
    else:
      return 'fake response proto'


class MonorailServicerTest(unittest.TestCase):

  def setUp(self):
    self.mox = mox.Mox()
    self.testbed = testbed.Testbed()
    self.testbed.activate()
    self.testbed.init_memcache_stub()
    self.testbed.init_datastore_v3_stub()
    self.testbed.init_user_stub()

    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        user=fake.UserService(),
        usergroup=fake.UserGroupService(),
        project=fake.ProjectService(),
        cache_manager=fake.CacheManager())
    self.project = self.services.project.TestAddProject(
        'proj', project_id=789, owner_ids=[111])
    # whitelisted_bot's email is whitelisted in testing/api_clients.cfg.
    self.whitelisted_bot = self.services.user.TestAddUser(
        '123456789@developer.gserviceaccount.com', 999)
    # whitelisted_client_id_user is used to test accounts that are only
    # whitelisted with the client_id.
    self.whitelisted_client_id_user = self.services.user.TestAddUser(
        'whitelisted-with-client-id@developer.gserviceaccount.com', 888)
    self.non_member = self.services.user.TestAddUser(
        'nonmember@example.com', 222)
    self.test_user = self.services.user.TestAddUser('test@example.com', 420)
    self.svcr = TestableServicer(self.services)
    self.nonmember_token = xsrf.GenerateToken(222, xsrf.XHR_SERVLET_PATH)
    self.request = UpdateSomethingRequest(exc_class=None)
    self.prpc_context = context.ServicerContext()
    self.prpc_context.set_code(codes.StatusCode.OK)
    self.prpc_context._invocation_metadata = [
        (monorail_servicer.XSRF_TOKEN_HEADER, self.nonmember_token)]

    self.oauth_patcher = mock.patch(
        'google.appengine.api.oauth.get_current_user')
    self.mock_oauth_gcu = self.oauth_patcher.start()
    self.mock_oauth_gcu.return_value = None

    self.oauth_client_id_patcher = mock.patch(
        'google.appengine.api.oauth.get_client_id')
    self.mock_oauth_gcid = self.oauth_client_id_patcher.start()
    self.mock_oauth_gcid.return_value = "1234common.clientid"

  def tearDown(self):
    self.mox.UnsetStubs()
    self.mox.ResetAll()
    self.testbed.deactivate()

  def SetUpRecordMonitoringStats(self):
    self.mox.StubOutWithMock(json_format, 'MessageToJson')
    json_format.MessageToJson(self.request).AndReturn('json of request')
    json_format.MessageToJson('fake response proto').AndReturn(
        'json of response')
    self.mox.ReplayAll()

  def testRun_SiteWide_Normal(self):
    """Calling the handler through the decorator."""
    self.testbed.setup_env(user_email=self.non_member.email, overwrite=True)
    self.SetUpRecordMonitoringStats()
    # pylint: disable=unexpected-keyword-arg
    response = self.svcr.CalcSomething(
        self.request, self.prpc_context, cnxn=self.cnxn)
    self.assertIsNone(self.svcr.seen_mc.cnxn)  # Because of CleanUp().
    self.assertEqual(self.svcr.seen_mc.auth.email, self.non_member.email)
    self.assertIn(permissions.CREATE_HOTLIST.lower(),
                  self.svcr.seen_mc.perms.perm_names)
    self.assertNotIn(permissions.ADMINISTER_SITE.lower(),
                     self.svcr.seen_mc.perms.perm_names)
    self.assertEqual(self.request, self.svcr.seen_request)
    self.assertEqual('fake response proto', response)
    self.assertEqual(codes.StatusCode.OK, self.prpc_context._code)

  def testRun_RequesterBanned(self):
    """If we reject the request, give PERMISSION_DENIED."""
    self.non_member.banned = 'Spammer'
    self.testbed.setup_env(user_email=self.non_member.email, overwrite=True)
    self.SetUpRecordMonitoringStats()
    # pylint: disable=unexpected-keyword-arg
    self.svcr.CalcSomething(
        self.request, self.prpc_context, cnxn=self.cnxn)
    self.assertFalse(self.svcr.was_called)
    self.assertEqual(
        codes.StatusCode.PERMISSION_DENIED, self.prpc_context._code)

  def testRun_AnonymousRequester(self):
    """Test we properly process anonymous users with valid tokens."""
    self.prpc_context._invocation_metadata = [
        (monorail_servicer.XSRF_TOKEN_HEADER,
         xsrf.GenerateToken(0, xsrf.XHR_SERVLET_PATH))]
    self.SetUpRecordMonitoringStats()
    # pylint: disable=unexpected-keyword-arg
    response = self.svcr.CalcSomething(
        self.request, self.prpc_context, cnxn=self.cnxn)
    self.assertIsNone(self.svcr.seen_mc.cnxn)  # Because of CleanUp().
    self.assertIsNone(self.svcr.seen_mc.auth.email)
    self.assertNotIn(permissions.CREATE_HOTLIST.lower(),
                  self.svcr.seen_mc.perms.perm_names)
    self.assertNotIn(permissions.ADMINISTER_SITE.lower(),
                     self.svcr.seen_mc.perms.perm_names)
    self.assertEqual(self.request, self.svcr.seen_request)
    self.assertEqual('fake response proto', response)
    self.assertEqual(codes.StatusCode.OK, self.prpc_context._code)

  def testRun_DistributedInvalidation(self):
    """The Run method must call DoDistributedInvalidation()."""
    self.testbed.setup_env(user_email=self.non_member.email, overwrite=True)
    self.SetUpRecordMonitoringStats()
    # pylint: disable=unexpected-keyword-arg
    self.svcr.CalcSomething(
        self.request, self.prpc_context, cnxn=self.cnxn)
    self.assertIsNotNone(self.services.cache_manager.last_call)

  def testRun_HandlerErrorResponse(self):
    """An expected exception in the method causes an error status."""
    self.testbed.setup_env(user_email=self.non_member.email, overwrite=True)
    self.SetUpRecordMonitoringStats()
    # pylint: disable=attribute-defined-outside-init
    self.request.exc_class = exceptions.NoSuchUserException
    # pylint: disable=unexpected-keyword-arg
    response = self.svcr.CalcSomething(
        self.request, self.prpc_context, cnxn=self.cnxn)
    self.assertTrue(self.svcr.was_called)
    self.assertIsNone(self.svcr.seen_mc.cnxn)  # Because of CleanUp().
    self.assertEqual(self.svcr.seen_mc.auth.email, self.non_member.email)
    self.assertEqual(self.request, self.svcr.seen_request)
    self.assertIsNone(response)
    self.assertEqual(codes.StatusCode.NOT_FOUND, self.prpc_context._code)

  def testRun_HandlerProgrammingError(self):
    """An unexception in the handler method is re-raised."""
    self.testbed.setup_env(user_email=self.non_member.email, overwrite=True)
    self.SetUpRecordMonitoringStats()
    # pylint: disable=attribute-defined-outside-init
    self.request.exc_class = NotImplementedError
    self.assertRaises(
        NotImplementedError,
        self.svcr.CalcSomething,
        self.request, self.prpc_context, cnxn=self.cnxn)
    self.assertTrue(self.svcr.was_called)
    self.assertIsNone(self.svcr.seen_mc.cnxn)  # Because of CleanUp().

  def testGetAndAssertRequesterAuth_Cookie_Anon(self):
    """We get and allow requests from anon user using cookie auth."""
    metadata = {
        monorail_servicer.XSRF_TOKEN_HEADER: xsrf.GenerateToken(
            0, xsrf.XHR_SERVLET_PATH)}
    # Signed out.
    self.assertIsNone(self.svcr.GetAndAssertRequesterAuth(
        self.cnxn, metadata, self.services).email)

  def testGetAndAssertRequesterAuth_Cookie_SignedIn(self):
    """We get and allow requests from signed in users using cookie auth."""
    metadata = dict(self.prpc_context.invocation_metadata())
    # Signed in with cookie auth.
    self.testbed.setup_env(user_email=self.non_member.email, overwrite=True)
    user_auth = self.svcr.GetAndAssertRequesterAuth(
        self.cnxn, metadata, self.services)
    self.assertEqual(self.non_member.email, user_auth.email)

  def testGetAndAssertRequester_Anon_BadToken(self):
    """We get the email address of the signed in user using oauth."""
    metadata = {}
    # Anonymous user has invalid token.
    with self.assertRaises(permissions.PermissionException):
      self.svcr.GetAndAssertRequesterAuth(self.cnxn, metadata, self.services)

  def testGetAndAssertRequesterAuth_Oauth_Whitelisted(self):
    metadata = {}
    # Signed in with oauth.
    self.mock_oauth_gcu.return_value = testing_helpers.Blank(
        email=lambda: self.whitelisted_bot.email)

    bot_auth = self.svcr.GetAndAssertRequesterAuth(
        self.cnxn, metadata, self.services)
    self.assertEqual(bot_auth.email, self.whitelisted_bot.email)

  def testGetAndAssertRequesterAuth_Oauth_NotWhitelisted(self):
    metadata = {}
    # Signed in with oauth.
    self.mock_oauth_gcu.return_value = testing_helpers.Blank(
        email=lambda: 'who-is-this@test.com')

    with self.assertRaises(permissions.PermissionException):
      self.svcr.GetAndAssertRequesterAuth(self.cnxn, metadata, self.services)

  def testGetAndAssertRequesterAuth_Oauth_ClientIDOnly(self):
    """We get and allow accounts that only have their client_id whitelisted."""
    metadata = {}
    self.mock_oauth_gcu.return_value = testing_helpers.Blank(
        email=lambda: self.whitelisted_client_id_user.email)
    self.mock_oauth_gcid.return_value = "98723764876"
    both_auth = self.svcr.GetAndAssertRequesterAuth(
        self.cnxn, metadata, self.services)
    self.assertEqual(both_auth.email, self.whitelisted_client_id_user.email)

  def testGetAndAssertRequesterAuth_Banned(self):
    self.non_member.banned = 'Spammer'
    metadata = dict(self.prpc_context.invocation_metadata())
    # Signed in with cookie auth.
    self.testbed.setup_env(user_email=self.non_member.email, overwrite=True)
    with self.assertRaises(permissions.BannedUserException):
      self.svcr.GetAndAssertRequesterAuth(self.cnxn, metadata, self.services)

  def testGetRequester_TestAccountOnAppspot(self):
    """Specifying test_account is ignored on deployed server."""
    # pylint: disable=attribute-defined-outside-init
    metadata = {'x-test-account': 'test@example.com'}
    with self.assertRaises(exceptions.InputException):
      self.svcr.GetAndAssertRequesterAuth(self.cnxn, metadata, self.services)

  def testGetRequester_TestAccountOnDev(self):
    """For integration testing, we can set test_account on dev_server."""
    try:
      orig_local_mode = settings.local_mode
      settings.local_mode = True

      # pylint: disable=attribute-defined-outside-init
      metadata = {'x-test-account': 'test@example.com'}
      test_auth = self.svcr.GetAndAssertRequesterAuth(
          self.cnxn, metadata, self.services)
      self.assertEqual('test@example.com', test_auth.email)

      # pylint: disable=attribute-defined-outside-init
      metadata = {'x-test-account': 'test@anythingelse.com'}
      with self.assertRaises(exceptions.InputException):
        self.svcr.GetAndAssertRequesterAuth(self.cnxn, metadata, self.services)
    finally:
      settings.local_mode = orig_local_mode

  def testAssertBaseChecks_SiteIsReadOnly_Write(self):
    """We reject writes and allow reads when site is read-only."""
    orig_read_only = settings.read_only
    try:
      settings.read_only = True
      metadata = {}
      self.assertRaises(
        permissions.PermissionException,
        self.svcr.AssertBaseChecks, self.request, metadata)
    finally:
      settings.read_only = orig_read_only

  def testAssertBaseChecks_SiteIsReadOnly_Read(self):
    """We reject writes and allow reads when site is read-only."""
    orig_read_only = settings.read_only
    try:
      settings.read_only = True
      metadata = {monorail_servicer.XSRF_TOKEN_HEADER: self.nonmember_token}

      # Our default request is an update.
      with self.assertRaises(permissions.PermissionException):
        self.svcr.AssertBaseChecks(self.request, metadata)

      # A method name starting with "List" or "Get" will run OK.
      self.request = ListSomethingRequest(exc_class=None)
      self.svcr.AssertBaseChecks(self.request, metadata)
    finally:
      settings.read_only = orig_read_only

  def testGetRequestProject(self):
    """We get a project specified by request field project_name."""
    # No project specified.
    self.assertIsNone(self.svcr.GetRequestProject(self.cnxn, self.request))

    # Existing project specified.
    # pylint: disable=attribute-defined-outside-init
    self.request.project_name = 'proj'
    self.assertEqual(
        self.project, self.svcr.GetRequestProject(self.cnxn, self.request))

    # Bad project specified.
    # pylint: disable=attribute-defined-outside-init
    self.request.project_name = 'not-a-proj'
    self.assertIsNone(self.svcr.GetRequestProject(self.cnxn, self.request))

  def CheckExceptionStatus(self, e, expected_code):
    mc = monorailcontext.MonorailContext(self.services)
    self.prpc_context.set_code(codes.StatusCode.OK)
    processed = self.svcr.ProcessException(e, self.prpc_context, mc)
    if expected_code:
      self.assertTrue(processed)
      self.assertEqual(expected_code, self.prpc_context._code)
    else:
      self.assertFalse(processed)
      self.assertEqual(codes.StatusCode.OK, self.prpc_context._code)

  def testProcessException(self):
    """Expected exceptions are converted to pRPC codes, expected not."""
    self.CheckExceptionStatus(
        exceptions.NoSuchUserException(), codes.StatusCode.NOT_FOUND)
    self.CheckExceptionStatus(
        exceptions.NoSuchProjectException(), codes.StatusCode.NOT_FOUND)
    self.CheckExceptionStatus(
        exceptions.NoSuchIssueException(), codes.StatusCode.NOT_FOUND)
    self.CheckExceptionStatus(
        exceptions.NoSuchComponentException(), codes.StatusCode.NOT_FOUND)
    self.CheckExceptionStatus(
        permissions.BannedUserException(), codes.StatusCode.PERMISSION_DENIED)
    self.CheckExceptionStatus(
        permissions.PermissionException(), codes.StatusCode.PERMISSION_DENIED)
    self.CheckExceptionStatus(
        exceptions.GroupExistsException(), codes.StatusCode.INVALID_ARGUMENT)
    self.CheckExceptionStatus(
        exceptions.InvalidComponentNameException(),
        codes.StatusCode.INVALID_ARGUMENT)
    self.CheckExceptionStatus(
        ratelimiter.ApiRateLimitExceeded('client_id', 'email'),
        codes.StatusCode.PERMISSION_DENIED)
    self.CheckExceptionStatus(
        features_svc.HotlistAlreadyExists(), codes.StatusCode.INVALID_ARGUMENT)
    self.CheckExceptionStatus(NotImplementedError(), None)

  def testRecordMonitoringStats_RequestClassDoesNotEndInRequest(self):
    """We cope with request proto class names that do not end in 'Request'."""
    self.request = 'this is a string'
    self.SetUpRecordMonitoringStats()
    start_time = 1522559788.939511
    now = 1522569311.892738
    self.svcr.RecordMonitoringStats(
        start_time, self.request, 'fake response proto', self.prpc_context,
        now=now)
