# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the projects servicer."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import json
import mock
import unittest
import mox

from google.protobuf import wrappers_pb2

from components.prpc import codes
from components.prpc import context
from components.prpc import server

from api import converters
from api.api_proto import common_pb2
from api.api_proto import features_pb2
from api.api_proto import features_objects_pb2
from api.api_proto import issue_objects_pb2
from framework import authdata
from framework import exceptions
from framework import monorailcontext
from framework import permissions
from framework import sorting
from testing import fake
from testing import testing_helpers
from tracker import tracker_bizobj
from services import features_svc
from services import service_manager

# Import component_helpers_test to mock cloudstorage before it is imported by
# component_helpers via features servicer.
from features.test import component_helpers_test
from api import features_servicer  # pylint: disable=ungrouped-imports


class FeaturesServicerTest(unittest.TestCase):

  def setUp(self):
    self.mox = mox.Mox()
    self.cnxn = fake.MonorailConnection()
    self.services = service_manager.Services(
        cache_manager=fake.CacheManager(),
        config=fake.ConfigService(),
        issue=fake.IssueService(),
        user=fake.UserService(),
        usergroup=fake.UserGroupService(),
        project=fake.ProjectService(),
        features=fake.FeaturesService(),
        hotlist_star=fake.HotlistStarService())
    sorting.InitializeArtValues(self.services)

    self.project = self.services.project.TestAddProject(
        'proj', project_id=789, owner_ids=[111], contrib_ids=[222, 333])
    self.config = tracker_bizobj.MakeDefaultProjectIssueConfig(789)
    self.user1 = self.services.user.TestAddUser('owner@example.com', 111)
    self.user2 = self.services.user.TestAddUser('editor@example.com', 222)
    self.user3 = self.services.user.TestAddUser('foo@example.com', 333)
    self.user4 = self.services.user.TestAddUser('bar@example.com', 444)
    self.features_svcr = features_servicer.FeaturesServicer(
        self.services, make_rate_limiter=False)
    self.prpc_context = context.ServicerContext()
    self.prpc_context.set_code(codes.StatusCode.OK)
    self.issue_1 = fake.MakeTestIssue(
        789, 1, 'sum', 'New', 111, project_name='proj', issue_id=78901)
    self.issue_2 = fake.MakeTestIssue(
        789, 2, 'sum', 'Fixed', 111, project_name='proj', issue_id=78902,
        closed_timestamp=112223344)
    self.issue_3 = fake.MakeTestIssue(
        789, 3, 'sum', 'New', 111, project_name='proj', issue_id=78903)

    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)
    self.services.issue.TestAddIssue(self.issue_3)

    self.project_2 = self.services.project.TestAddProject(
        'proj2', project_id=788, owner_ids=[111], contrib_ids=[222, 333])
    self.config_2 = tracker_bizobj.MakeDefaultProjectIssueConfig(788)
    self.issue_21 = fake.MakeTestIssue(
        788, 1, 'sum', 'New', 111, project_name='proj2', issue_id=78801)
    self.issue_22 = fake.MakeTestIssue(
        788, 2, 'sum', 'New', 111, project_name='proj2', issue_id=78802)
    self.issue_23 = fake.MakeTestIssue(
        788, 3, 'sum', 'New', 111, project_name='proj2', issue_id=78803)
    self.services.issue.TestAddIssue(self.issue_21)
    self.services.issue.TestAddIssue(self.issue_22)
    self.services.issue.TestAddIssue(self.issue_23)

    self.PAST_TIME = 123456

    # For testing PredictComponent
    self._ml_engine = component_helpers_test.FakeMLEngine(self)
    self._top_words = None
    self._components_by_index = None

    mock.patch(
        'services.ml_helpers.setup_ml_engine', lambda: self._ml_engine).start()
    mock.patch(
        'features.component_helpers._GetTopWords',
        lambda _: self._top_words).start()
    mock.patch('cloudstorage.open', self.cloudstorageOpen).start()
    mock.patch('settings.component_features', 5).start()

    self.addCleanup(mock.patch.stopall)

  def cloudstorageOpen(self, name, mode):
    """Create a file mock that returns self._components_by_index when read."""
    open_fn = mock.mock_open(read_data=json.dumps(self._components_by_index))
    return open_fn(name, mode)

  def tearDown(self):
    self.mox.UnsetStubs()
    self.mox.ResetAll()

  def CallWrapped(self, wrapped_handler, *args, **kwargs):
    return wrapped_handler.wrapped(self.features_svcr, *args, **kwargs)

  def testListHotlistsByUser_SearchByEmail(self):
    """We can get a list of hotlists for a given email."""
    # Public hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])

    # Query for issues for 'owner@example.com'
    user_ref = common_pb2.UserRef(display_name='owner@example.com')
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)

    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)
    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual(111, hotlist.owner_ref.user_id)
    self.assertEqual('ow...@example.com', hotlist.owner_ref.display_name)
    self.assertEqual('Fake-Hotlist', hotlist.name)
    self.assertEqual('Summary', hotlist.summary)
    self.assertEqual('Description', hotlist.description)

  def testListHotlistsByUser_SearchByOwner(self):
    """We can get a list of hotlists for a given user."""
    # Public hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])

    # Query for issues for 'owner@example.com'
    user_ref = common_pb2.UserRef(user_id=111)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')

    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)
    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual(111, hotlist.owner_ref.user_id)
    self.assertEqual('ow...@example.com', hotlist.owner_ref.display_name)
    self.assertEqual('Fake-Hotlist', hotlist.name)
    self.assertEqual('Summary', hotlist.summary)
    self.assertEqual('Description', hotlist.description)

  def testListHotlistsByUser_SearchByEditor(self):
    """We can get a list of hotlists for a given user."""
    # Public hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])

    # Query for issues for 'editor@example.com'
    user_ref = common_pb2.UserRef(user_id=222)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')

    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)
    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual(111, hotlist.owner_ref.user_id)
    self.assertEqual('ow...@example.com', hotlist.owner_ref.display_name)
    self.assertEqual('Fake-Hotlist', hotlist.name)
    self.assertEqual('Summary', hotlist.summary)
    self.assertEqual('Description', hotlist.description)

  def testListHotlistsByUser_NotSignedIn(self):
    # Public hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])

    # Query for issues for 'owner@example.com'
    user_ref = common_pb2.UserRef(user_id=111)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're not authenticated
    mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)
    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)

    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual(111, hotlist.owner_ref.user_id)

  def testListHotlistsByUser_Empty(self):
    """There are no hotlists for the given user."""
    # Public hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])

    # Query for issues for 'bar@example.com'
    user_ref = common_pb2.UserRef(user_id=444)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)

    self.assertEqual(0, len(response.hotlists))

  def testListHotlistsByUser_NoHotlists(self):
    """There are no hotlists."""
    # No hotlists
    # Query for issues for 'owner@example.com'
    user_ref = common_pb2.UserRef(user_id=111)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)
    self.assertEqual(0, len(response.hotlists))

  def testListHotlistsByUser_PrivateIssueAsOwner(self):
    # Private hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222], is_private=True)

    # Query for issues for 'owner@example.com'
    user_ref = common_pb2.UserRef(user_id=111)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're authenticated as 'owner@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)

    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual(111, hotlist.owner_ref.user_id)

  def testListHotlistsByUser_PrivateIssueAsEditor(self):
    # Private hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222], is_private=True)

    # Query for issues for 'owner@example.com'
    user_ref = common_pb2.UserRef(user_id=111)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're authenticated as 'editor@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='editor@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)

    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual(111, hotlist.owner_ref.user_id)

  def testListHotlistsByUser_PrivateIssueNoAccess(self):
    # Private hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222], is_private=True)

    # Query for issues for 'owner@example.com'
    user_ref = common_pb2.UserRef(user_id=111)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)

    self.assertEqual(0, len(response.hotlists))

  def testListHotlistsByUser_PrivateIssueNotSignedIn(self):
    # Private hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222], is_private=True)

    # Query for issues for 'owner@example.com'
    user_ref = common_pb2.UserRef(user_id=111)
    request = features_pb2.ListHotlistsByUserRequest(user=user_ref)

    # We're not authenticated
    mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)
    response = self.CallWrapped(self.features_svcr.ListHotlistsByUser, mc,
                                request)

    self.assertEqual(0, len(response.hotlists))

  def AddIssueToHotlist(self, hotlist_id, issue_id=78901, adder_id=111):
    self.services.features.AddIssuesToHotlists(
        self.cnxn, [hotlist_id], [(issue_id, adder_id, 0, '')],
        None, None, None)

  def testListHotlistsByIssue_Normal(self):
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])
    self.AddIssueToHotlist(hotlist.hotlist_id)

    issue_ref = common_pb2.IssueRef(project_name='proj', local_id=1)
    request = features_pb2.ListHotlistsByIssueRequest(issue=issue_ref)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByIssue, mc,
                                request)

    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual('Fake-Hotlist', hotlist.name)

  def testListHotlistsByIssue_NotSignedIn(self):
    # Public hostlist owned by 'owner@example.com'
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])
    self.AddIssueToHotlist(hotlist.hotlist_id)

    issue_ref = common_pb2.IssueRef(project_name='proj', local_id=1)
    request = features_pb2.ListHotlistsByIssueRequest(issue=issue_ref)

    # We're not authenticated
    mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)
    response = self.CallWrapped(self.features_svcr.ListHotlistsByIssue, mc,
                                request)

    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual('Fake-Hotlist', hotlist.name)

  def testListHotlistsByIssue_Empty(self):
    """There are no hotlists with the given issue."""
    # Public hostlist owned by 'owner@example.com'
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])

    issue_ref = common_pb2.IssueRef(project_name='proj', local_id=1)
    request = features_pb2.ListHotlistsByIssueRequest(issue=issue_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByIssue, mc,
                                request)

    self.assertEqual(0, len(response.hotlists))

  def testListHotlistsByIssue_NoHotlists(self):
    issue_ref = common_pb2.IssueRef(project_name='proj', local_id=1)
    request = features_pb2.ListHotlistsByIssueRequest(issue=issue_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByIssue, mc,
                                request)
    self.assertEqual(0, len(response.hotlists))

  def testListHotlistsByIssue_PrivateHotlistAsOwner(self):
    """An owner can view their private issues."""
    # Private hostlist owned by 'owner@example.com'
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222], is_private=True)
    self.AddIssueToHotlist(hotlist.hotlist_id)

    issue_ref = common_pb2.IssueRef(project_name='proj', local_id=1)
    request = features_pb2.ListHotlistsByIssueRequest(issue=issue_ref)

    # We're authenticated as 'owner@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByIssue, mc,
                                request)

    self.assertEqual(1, len(response.hotlists))
    hotlist = response.hotlists[0]
    self.assertEqual('Fake-Hotlist', hotlist.name)

  def testListHotlistsByIssue_PrivateHotlistNoAccess(self):
    # Private hostlist owned by 'owner@example.com'
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222], is_private=True)
    self.AddIssueToHotlist(hotlist.hotlist_id)

    issue_ref = common_pb2.IssueRef(project_name='proj', local_id=1)
    request = features_pb2.ListHotlistsByIssueRequest(issue=issue_ref)

    # We're authenticated as 'foo@example.com'
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    response = self.CallWrapped(self.features_svcr.ListHotlistsByIssue, mc,
                                request)

    self.assertEqual(0, len(response.hotlists))

  def testListRecentlyVisitedHotlists(self):
    hotlists = [
        self.services.features.CreateHotlist(
            self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
            owner_ids=[self.user2.user_id], editor_ids=[self.user1.user_id],
            default_col_spec='chicken'),
        self.services.features.CreateHotlist(
            self.cnxn, 'Fake-Hotlist-2', 'Summary', 'Description',
            owner_ids=[self.user1.user_id], editor_ids=[self.user2.user_id],
            default_col_spec='honk'),
        self.services.features.CreateHotlist(
            self.cnxn, 'Private-Hotlist', 'Summary', 'Description',
            owner_ids=[self.user3.user_id], editor_ids=[self.user2.user_id],
            is_private=True)]

    for hotlist in hotlists:
      self.services.user.AddVisitedHotlist(
          self.cnxn, self.user1.user_id, hotlist.hotlist_id)

    request = features_pb2.ListRecentlyVisitedHotlistsRequest()
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user1.email)
    response = self.CallWrapped(
        self.features_svcr.ListRecentlyVisitedHotlists, mc, request)

    expected_hotlists = [
        features_objects_pb2.Hotlist(
            owner_ref=common_pb2.UserRef(
                user_id=self.user2.user_id,
                display_name=testing_helpers.ObscuredEmail(self.user2.email)),
            editor_refs=[common_pb2.UserRef(
                user_id=self.user1.user_id,
                display_name=self.user1.email)],
            name='Fake-Hotlist',
            summary='Summary',
            description='Description',
            is_private=False,
            default_col_spec='chicken'),
        features_objects_pb2.Hotlist(
            owner_ref=common_pb2.UserRef(
                user_id=self.user1.user_id,
                display_name=self.user1.email),
            editor_refs=[common_pb2.UserRef(
                user_id=self.user2.user_id,
                display_name=testing_helpers.ObscuredEmail(self.user2.email))],
            name='Fake-Hotlist-2',
            summary='Summary',
            description='Description',
            is_private=False,
            default_col_spec='honk')]

    # We don't have permission to see the last issue, because it is marked as
    # private and we're not owners or editors.
    self.assertEqual(expected_hotlists, list(response.hotlists))

  def testListRecentlyVisitedHotlists_Anon(self):
    request = features_pb2.ListRecentlyVisitedHotlistsRequest()
    mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)
    response = self.CallWrapped(
        self.features_svcr.ListRecentlyVisitedHotlists, mc, request)
    self.assertEqual(0, len(response.hotlists))

  def testListStarredHotlists(self):
    hotlists = [
        self.services.features.CreateHotlist(
            self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
            owner_ids=[self.user2.user_id], editor_ids=[self.user1.user_id],
            default_col_spec='cow chicken'),
        self.services.features.CreateHotlist(
            self.cnxn, 'Fake-Hotlist-2', 'Summary', 'Description',
            owner_ids=[self.user1.user_id],
            editor_ids=[self.user2.user_id, self.user3.user_id],
            default_col_spec=''),
        self.services.features.CreateHotlist(
            self.cnxn, 'Private-Hotlist', 'Summary', 'Description',
            owner_ids=[self.user3.user_id], editor_ids=[self.user2.user_id],
            is_private=True, default_col_spec='chicken')]

    for hotlist in hotlists:
      self.services.hotlist_star.SetStar(
          self.cnxn, hotlist.hotlist_id, self.user1.user_id, True)

    request = features_pb2.ListStarredHotlistsRequest()
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    response = self.CallWrapped(
        self.features_svcr.ListStarredHotlists, mc, request)

    expected_hotlists = [
        features_objects_pb2.Hotlist(
            owner_ref=common_pb2.UserRef(
                user_id=self.user2.user_id,
                display_name=testing_helpers.ObscuredEmail(self.user2.email)),
            editor_refs=[common_pb2.UserRef(
                user_id=self.user1.user_id,
                display_name=self.user1.email)],
            name='Fake-Hotlist',
            summary='Summary',
            description='Description',
            is_private=False,
            default_col_spec='cow chicken'),
        features_objects_pb2.Hotlist(
            owner_ref=common_pb2.UserRef(
                user_id=self.user1.user_id,
                display_name=self.user1.email),
            editor_refs=[
                common_pb2.UserRef(
                    user_id=self.user2.user_id,
                    display_name=testing_helpers.ObscuredEmail(
                        self.user2.email)),
                common_pb2.UserRef(
                    user_id=self.user3.user_id,
                    display_name=testing_helpers.ObscuredEmail(
                        self.user3.email))],
            name='Fake-Hotlist-2',
            summary='Summary',
            description='Description',
            is_private=False)]

    # We don't have permission to see the last issue, because it is marked as
    # private and we're not owners or editors.
    self.assertEqual(expected_hotlists, list(response.hotlists))

  def testListStarredHotlists_Anon(self):
    request = features_pb2.ListStarredHotlistsRequest()
    mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)
    response = self.CallWrapped(
        self.features_svcr.ListStarredHotlists, mc, request)
    self.assertEqual(0, len(response.hotlists))

  def CallGetStarCount(self):
    # Query for hotlists for 'owner@example.com'
    owner_ref = common_pb2.UserRef(user_id=111)
    hotlist_ref = common_pb2.HotlistRef(name='Fake-Hotlist', owner=owner_ref)
    request = features_pb2.GetHotlistStarCountRequest(hotlist_ref=hotlist_ref)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    response = self.CallWrapped(
        self.features_svcr.GetHotlistStarCount, mc, request)
    return response.star_count

  def CallStar(self, requester='owner@example.com', starred=True):
    # Query for hotlists for 'owner@example.com'
    owner_ref = common_pb2.UserRef(user_id=111)
    hotlist_ref = common_pb2.HotlistRef(name='Fake-Hotlist', owner=owner_ref)
    request = features_pb2.StarHotlistRequest(
        hotlist_ref=hotlist_ref, starred=starred)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=requester)
    response = self.CallWrapped(
        self.features_svcr.StarHotlist, mc, request)
    return response.star_count

  def testStarCount_Normal(self):
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])
    self.assertEqual(0, self.CallGetStarCount())
    self.assertEqual(1, self.CallStar())
    self.assertEqual(1, self.CallGetStarCount())

  def testStarCount_StarTwiceSameUser(self):
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])
    self.assertEqual(1, self.CallStar())
    self.assertEqual(1, self.CallStar())
    self.assertEqual(1, self.CallGetStarCount())

  def testStarCount_StarTwiceDifferentUser(self):
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])
    self.assertEqual(1, self.CallStar())
    self.assertEqual(2, self.CallStar(requester='user_222@example.com'))
    self.assertEqual(2, self.CallGetStarCount())

  def testStarCount_RemoveStarTwiceSameUser(self):
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])
    self.assertEqual(1, self.CallStar())
    self.assertEqual(1, self.CallGetStarCount())

    self.assertEqual(0, self.CallStar(starred=False))
    self.assertEqual(0, self.CallStar(starred=False))
    self.assertEqual(0, self.CallGetStarCount())

  def testStarCount_RemoveStarTwiceDifferentUser(self):
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[222])
    self.assertEqual(1, self.CallStar())
    self.assertEqual(2, self.CallStar(requester='user_222@example.com'))
    self.assertEqual(2, self.CallGetStarCount())

    self.assertEqual(1, self.CallStar(starred=False))
    self.assertEqual(
        0, self.CallStar(requester='user_222@example.com', starred=False))
    self.assertEqual(0, self.CallGetStarCount())

  def testGetHotlist(self):
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[self.user3.user_id], editor_ids=[self.user4.user_id],
        is_private=True, default_col_spec='corgi butts')

    owner_ref = common_pb2.UserRef(user_id=self.user3.user_id)
    hotlist_ref = common_pb2.HotlistRef(
        name=hotlist.name, owner=owner_ref)
    request = features_pb2.GetHotlistRequest(hotlist_ref=hotlist_ref)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user4.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(
        self.features_svcr.GetHotlist, mc, request)

    self.assertEqual(
        response.hotlist,
        features_objects_pb2.Hotlist(
          owner_ref=common_pb2.UserRef(
              user_id=self.user3.user_id,
              display_name=testing_helpers.ObscuredEmail(self.user3.email)),
          editor_refs=[common_pb2.UserRef(
              user_id=self.user4.user_id,
              display_name=self.user4.email)],
          name=hotlist.name,
          summary=hotlist.summary,
          description=hotlist.description,
          default_col_spec='corgi butts',
          is_private=True))

  def testGetHotlist_BadInput(self):
    hotlist_ref = common_pb2.HotlistRef()
    request = features_pb2.GetHotlistRequest(hotlist_ref=hotlist_ref)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    with self.assertRaises(features_svc.NoSuchHotlistException):
      self.CallWrapped(self.features_svcr.GetHotlist, mc, request)

  def testGetHotlistID(self):
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[self.user3.user_id], editor_ids=[self.user4.user_id])

    owner_ref = common_pb2.UserRef(user_id=self.user3.user_id)
    hotlist_ref = common_pb2.HotlistRef(name=hotlist.name, owner=owner_ref)
    request = features_pb2.GetHotlistIDRequest(hotlist_ref=hotlist_ref)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user4.email)
    mc.LookupLoggedInUserPerms(None)
    response = self.CallWrapped(self.features_svcr.GetHotlistID, mc, request)
    self.assertEqual(hotlist.hotlist_id, response.hotlist_id)

  def testGetHotlistID_NoPermission(self):
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[self.user3.user_id], editor_ids=[self.user4.user_id],
        is_private=True)

    owner_ref = common_pb2.UserRef(user_id=self.user3.user_id)
    hotlist_ref = common_pb2.HotlistRef(name=hotlist.name, owner=owner_ref)
    request = features_pb2.GetHotlistIDRequest(hotlist_ref=hotlist_ref)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user2.email)
    mc.LookupLoggedInUserPerms(None)
    with self.assertRaises(permissions.PermissionException):
      self.CallWrapped(self.features_svcr.GetHotlistID, mc, request)

  def testGetHotlistID_NoSuchHotlist(self):
    owner_ref = common_pb2.UserRef(user_id=self.user3.user_id)
    hotlist_ref = common_pb2.HotlistRef(name='Honklist', owner=owner_ref)
    request = features_pb2.GetHotlistIDRequest(hotlist_ref=hotlist_ref)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user2.email)
    mc.LookupLoggedInUserPerms(None)
    with self.assertRaises(features_svc.NoSuchHotlistException):
      self.CallWrapped(self.features_svcr.GetHotlistID, mc, request)

  def testListHotlistItems(self):
    hotlist_id = self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[]).hotlist_id
    self.services.features.UpdateHotlistItems(
        self.cnxn, hotlist_id, [],
        [(self.issue_3.issue_id, 222, 12234, 'Note'),  # cutoff by `start`
         (self.issue_1.issue_id, 222, 12345, 'Note'),
         (self.issue_2.issue_id, 222, 12349, 'Note'),  # closed issue
         (self.issue_21.issue_id, 111, 12346, 'Note'),  # Restrict-View issue
         (self.issue_22.issue_id, 111, 12347, 'Note'),
         (self.issue_23.issue_id, 222, 12344, 'Note')])  # cutoff by `max_items`
    self.issue_21.labels = ['Restrict-View-CoreTeam']

    owner_ref = common_pb2.UserRef(user_id=111)
    hotlist_ref = common_pb2.HotlistRef(name='Fake-Hotlist', owner=owner_ref)
    request = features_pb2.ListHotlistItemsRequest(
        hotlist_ref=hotlist_ref, can=2, pagination=common_pb2.Pagination(
            max_items=2, start=2))

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='foo@example.com')
    mc.LookupLoggedInUserPerms(self.project)
    response = self.CallWrapped(
        self.features_svcr.ListHotlistItems, mc, request)

    # TODO(jojwang): monorail:6754 testing.fake.py's UpdateHotlistItems
    # does not calculate ranks correctly. Requires small follow-up CL.
    expected = features_pb2.ListHotlistItemsResponse(
        items=[
            features_objects_pb2.HotlistItem(
                issue=issue_objects_pb2.Issue(
                    local_id=self.issue_1.local_id,
                    project_name=self.project.project_name,
                    summary=self.issue_1.summary,
                    status_ref=common_pb2.StatusRef(
                        status='New', means_open=True),
                    reporter_ref=common_pb2.UserRef(
                        user_id=self.user1.user_id,
                        display_name=testing_helpers.ObscuredEmail(
                            self.user1.email)),
                    owner_ref=common_pb2.UserRef(
                        user_id=self.user1.user_id,
                        display_name=testing_helpers.ObscuredEmail(
                            self.user1.email))),
                rank=1,
                adder_ref=common_pb2.UserRef(
                    user_id=self.user2.user_id,
                    display_name=testing_helpers.ObscuredEmail(
                        self.user2.email)),
                added_timestamp=12345, note='Note'),
            features_objects_pb2.HotlistItem(
                issue=issue_objects_pb2.Issue(
                    local_id=self.issue_22.local_id,
                    project_name=self.project_2.project_name,
                    summary=self.issue_22.summary,
                    status_ref=common_pb2.StatusRef(
                        status='New', means_open=True),
                    reporter_ref=common_pb2.UserRef(
                        user_id=self.user1.user_id,
                        display_name=testing_helpers.ObscuredEmail(
                            self.user1.email)),
                    owner_ref=common_pb2.UserRef(
                        user_id=self.user1.user_id,
                        display_name=testing_helpers.ObscuredEmail(
                            self.user1.email))),
                rank=2,
                adder_ref=common_pb2.UserRef(
                    user_id=self.user1.user_id,
                    display_name=testing_helpers.ObscuredEmail(
                        self.user1.email)),
                added_timestamp=12347, note='Note')])
    self.assertEqual(expected, response)

  def testCreateHotlist_Normal(self):
    request = features_pb2.CreateHotlistRequest(
        name='Fake-Hotlist',
        summary='Summary',
        description='Description',
        editor_refs=[
            common_pb2.UserRef(user_id=222),
            common_pb2.UserRef(display_name='foo@example.com')],
        issue_refs=[
            common_pb2.IssueRef(project_name='proj', local_id=1),
            common_pb2.IssueRef(project_name='proj', local_id=2)],
        is_private=True)
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    self.CallWrapped(self.features_svcr.CreateHotlist, mc, request)

    # Check that the hotlist was successfuly added.
    hotlist_id = self.services.features.LookupHotlistIDs(
        self.cnxn, ['Fake-Hotlist'], [111]).get(('fake-hotlist', 111))
    hotlist = self.services.features.GetHotlist(self.cnxn, hotlist_id)
    self.assertEqual('Summary', hotlist.summary)
    self.assertEqual('Description', hotlist.description)
    self.assertEqual([111], hotlist.owner_ids)
    self.assertEqual([222, 333], hotlist.editor_ids)
    self.assertEqual(
        [self.issue_1.issue_id, self.issue_2.issue_id],
        [item.issue_id for item in hotlist.items])
    self.assertTrue(hotlist.is_private)

  def testCreateHotlist_Simple(self):
    request = features_pb2.CreateHotlistRequest(
        name='Fake-Hotlist',
        summary='Summary',
        description='Description')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    self.CallWrapped(self.features_svcr.CreateHotlist, mc, request)

    # Check that the hotlist was successfuly added.
    hotlist_id = self.services.features.LookupHotlistIDs(
        self.cnxn, ['Fake-Hotlist'], [111]).get(('fake-hotlist', 111))
    hotlist = self.services.features.GetHotlist(self.cnxn, hotlist_id)
    self.assertEqual('Summary', hotlist.summary)
    self.assertEqual('Description', hotlist.description)
    self.assertEqual([111], hotlist.owner_ids)
    self.assertEqual([], hotlist.editor_ids)
    self.assertEqual(0, len(hotlist.items))
    self.assertFalse(hotlist.is_private)

  def testCheckHotlistName_OK(self):
    request = features_pb2.CheckHotlistNameRequest(name='Fake-Hotlist')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    result = self.CallWrapped(self.features_svcr.CheckHotlistName, mc, request)
    self.assertEqual('', result.error)

  def testCheckHotlistName_Anon(self):
    request = features_pb2.CheckHotlistNameRequest(name='Fake-Hotlist')
    mc = monorailcontext.MonorailContext(self.services, cnxn=self.cnxn)

    with self.assertRaises(exceptions.InputException):
      self.CallWrapped(self.features_svcr.CheckHotlistName, mc, request)

  def testCheckHotlistName_InvalidName(self):
    request = features_pb2.CheckHotlistNameRequest(name='**Invalid**')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')

    result = self.CallWrapped(self.features_svcr.CheckHotlistName, mc, request)
    self.assertNotEqual('', result.error)

  def testCheckHotlistName_AlreadyExists(self):
    self.services.features.CreateHotlist(
        self.cnxn, 'Fake-Hotlist', 'Summary', 'Description',
        owner_ids=[111], editor_ids=[])

    request = features_pb2.CheckHotlistNameRequest(name='Fake-Hotlist')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')

    result = self.CallWrapped(self.features_svcr.CheckHotlistName, mc, request)
    self.assertNotEqual('', result.error)

  def testRemoveIssuesFromHotlists(self):
    # Create two hotlists with issues 1 and 2.
    hotlist_1 = self.services.features.CreateHotlist(
        self.cnxn, 'Hotlist-1', 'Summary', 'Description', owner_ids=[111],
        editor_ids=[])
    hotlist_2 = self.services.features.CreateHotlist(
        self.cnxn, 'Hotlist-2', 'Summary', 'Description', owner_ids=[111],
        editor_ids=[])
    self.services.features.AddIssuesToHotlists(
        self.cnxn,
        [hotlist_1.hotlist_id, hotlist_2.hotlist_id],
        [(self.issue_1.issue_id, 111, 0, ''),
         (self.issue_2.issue_id, 111, 0, '')],
        None, None, None)

    # Remove Issue 1 from both hotlists.
    request = features_pb2.RemoveIssuesFromHotlistsRequest(
        hotlist_refs=[
            common_pb2.HotlistRef(
                name='Hotlist-1',
                owner=common_pb2.UserRef(user_id=111)),
            common_pb2.HotlistRef(
                name='Hotlist-2',
                owner=common_pb2.UserRef(user_id=111))],
        issue_refs=[
            common_pb2.IssueRef(project_name='proj', local_id=1)])

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    self.CallWrapped(self.features_svcr.RemoveIssuesFromHotlists, mc, request)

    # Only Issue 2 should remain in both lists.
    self.assertEqual(
        [self.issue_2.issue_id],
        [item.issue_id for item in hotlist_1.items])
    self.assertEqual(
        [self.issue_2.issue_id],
        [item.issue_id for item in hotlist_2.items])

  def testAddIssuesToHotlists(self):
    # Create two hotlists
    hotlist_1 = self.services.features.CreateHotlist(
        self.cnxn, 'Hotlist-1', 'Summary', 'Description', owner_ids=[111],
        editor_ids=[])
    hotlist_2 = self.services.features.CreateHotlist(
        self.cnxn, 'Hotlist-2', 'Summary', 'Description', owner_ids=[111],
        editor_ids=[])

    # Add Issue 1 to both hotlists
    request = features_pb2.AddIssuesToHotlistsRequest(
        note='Foo',
        hotlist_refs=[
            common_pb2.HotlistRef(
                name='Hotlist-1',
                owner=common_pb2.UserRef(user_id=111)),
            common_pb2.HotlistRef(
                name='Hotlist-2',
                owner=common_pb2.UserRef(user_id=111))],
        issue_refs=[
            common_pb2.IssueRef(project_name='proj', local_id=1)])

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    self.CallWrapped(self.features_svcr.AddIssuesToHotlists, mc, request)

    self.assertEqual(
        [self.issue_1.issue_id],
        [item.issue_id for item in hotlist_1.items])
    self.assertEqual(
        [self.issue_1.issue_id],
        [item.issue_id for item in hotlist_2.items])

    self.assertEqual('Foo', hotlist_1.items[0].note)
    self.assertEqual('Foo', hotlist_2.items[0].note)

  def testRerankHotlistIssues(self):
    """Rerank a hotlist."""
    issue_3 = fake.MakeTestIssue(
        789, 3, 'sum', 'New', 111, project_name='proj', issue_id=78903)
    issue_4 = fake.MakeTestIssue(
        789, 4, 'sum', 'New', 111, project_name='proj', issue_id=78904)
    self.services.issue.TestAddIssue(issue_3)
    self.services.issue.TestAddIssue(issue_4)

    owner_ids = [self.user1.user_id]
    follower_ids = [self.user2.user_id]
    editor_ids = [self.user3.user_id]
    hotlist_items = [
        (78904, 31, self.user2.user_id, self.PAST_TIME, 'note'),
        (78903, 21, self.user2.user_id, self.PAST_TIME, 'note'),
        (78902, 11, self.user2.user_id, self.PAST_TIME, 'note'),
        (78901, 1, self.user2.user_id, self.PAST_TIME, 'note')]
    hotlist = self.services.features.TestAddHotlist(
        'RerankHotlistName', summary='summary', owner_ids=owner_ids,
        editor_ids=editor_ids, follower_ids=follower_ids,
        hotlist_id=1236, hotlist_item_fields=hotlist_items)

    request = features_pb2.RerankHotlistIssuesRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name='RerankHotlistName',
            owner=common_pb2.UserRef(user_id=self.user1.user_id)),
        moved_refs=[common_pb2.IssueRef(
            project_name='proj', local_id=2)],
        target_ref=common_pb2.IssueRef(project_name='proj', local_id=4),
        split_above=True)

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user1.email)
    mc.LookupLoggedInUserPerms(self.project)
    self.CallWrapped(self.features_svcr.RerankHotlistIssues, mc, request)

    self.assertEqual(
        [item.issue_id for item in hotlist.items],
        [78901, 78903, 78902, 78904])

  def testUpdateHotlistIssueNote(self):
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Hotlist-1', 'Summary', 'Description', owner_ids=[111],
        editor_ids=[])
    self.services.features.AddIssuesToHotlists(
        self.cnxn,
        [hotlist.hotlist_id], [(self.issue_1.issue_id, 111, 0, '')],
        None, None, None)

    request = features_pb2.UpdateHotlistIssueNoteRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name='Hotlist-1',
            owner=common_pb2.UserRef(user_id=111)),
        issue_ref=common_pb2.IssueRef(
            project_name='proj',
            local_id=1),
        note='Note')

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    self.CallWrapped(self.features_svcr.UpdateHotlistIssueNote, mc, request)

    self.assertEqual('Note', hotlist.items[0].note)

  def testUpdateHotlistIssueNote_NotAllowed(self):
    hotlist = self.services.features.CreateHotlist(
        self.cnxn, 'Hotlist-1', 'Summary', 'Description', owner_ids=[222],
        editor_ids=[])
    self.services.features.AddIssuesToHotlists(
        self.cnxn,
        [hotlist.hotlist_id], [(self.issue_1.issue_id, 222, 0, '')],
        None, None, None)

    request = features_pb2.UpdateHotlistIssueNoteRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name='Hotlist-1',
            owner=common_pb2.UserRef(user_id=222)),
        issue_ref=common_pb2.IssueRef(
            project_name='proj',
            local_id=1),
        note='Note')

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    with self.assertRaises(permissions.PermissionException):
      self.CallWrapped(self.features_svcr.UpdateHotlistIssueNote, mc, request)

  def testUpdateHotlistRoles(self):
    """Test we can update hotlist people."""
    owner_ids = [self.user2.user_id]
    editor_ids = [self.user4.user_id]
    hotlist = self.services.features.TestAddHotlist(
        name='Hotlist-1', summary='summary', description='description',
        owner_ids=owner_ids, editor_ids=editor_ids, hotlist_id=1233)
    request = features_pb2.UpdateHotlistRolesRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name='Hotlist-1',
            owner=common_pb2.UserRef(user_id=self.user2.user_id)),
        people_delta=features_objects_pb2.HotlistPeopleDelta(
            new_owner_ref=common_pb2.UserRef(user_id=self.user3.user_id),
            add_follower_refs=[common_pb2.UserRef(user_id=self.user4.user_id)])
        )

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user2.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.features_svcr.UpdateHotlistRoles, mc, request)

    hotlist = self.services.features.GetHotlist(self.cnxn, hotlist.hotlist_id)
    self.assertEqual(hotlist.owner_ids, [self.user3.user_id])
    self.assertEqual(hotlist.editor_ids, [])
    self.assertEqual(hotlist.follower_ids, [self.user4.user_id])

  def testUpdateHotlistRoles_EmptyNewOwnerRef(self):
    """Test that an empty UserRef in owner_user_ref is processed correctly"""
    owner_ids = [self.user2.user_id]
    editor_ids = [self.user4.user_id]
    hotlist = self.services.features.TestAddHotlist(
        name='Hotlist-1', summary='summary', description='description',
        owner_ids=owner_ids, editor_ids=editor_ids, hotlist_id=1234)
    request = features_pb2.UpdateHotlistRolesRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name='Hotlist-1',
            owner=common_pb2.UserRef(user_id=self.user2.user_id)),
        people_delta=features_objects_pb2.HotlistPeopleDelta(
            add_follower_refs=[common_pb2.UserRef(user_id=self.user4.user_id)])
        )

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user2.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.features_svcr.UpdateHotlistRoles, mc, request)

    hotlist = self.services.features.GetHotlist(self.cnxn, hotlist.hotlist_id)
    self.assertEqual(hotlist.owner_ids, owner_ids)
    self.assertEqual(hotlist.editor_ids, [])
    self.assertEqual(hotlist.follower_ids, [self.user4.user_id])

  def testUpdateHotlistSettings(self):
    """Hotlist settings can be correctly updated with a FieldMask."""
    name = 'Hotlist-1'
    description = 'description'
    summary = 'summary'
    is_private = True
    hotlist = self.services.features.TestAddHotlist(
        name=name, summary=summary, description=description,
        owner_ids=[self.user2.user_id], editor_ids=[], hotlist_id=1235,
        is_private=is_private)

    new_name = 'newName'
    new_description = 'new description'
    request = features_pb2.UpdateHotlistSettingsRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name=hotlist.name,
            owner=common_pb2.UserRef(user_id=self.user2.user_id)),
        name=wrappers_pb2.StringValue(value=new_name),
        description=wrappers_pb2.StringValue(value=new_description))

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user2.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.features_svcr.UpdateHotlistSettings, mc, request)

    self.assertEqual(hotlist.description, new_description)
    self.assertEqual(hotlist.name, new_name)
    self.assertEqual(hotlist.summary, summary)

  @mock.patch('businesslogic.work_env.WorkEnv.UpdateHotlistSettings')
  def testUpdateHotlistSettings_NoChanges(self, mockUpdateHotlistSettings):
    """All hotlist settings will be updated when no FieldMask is specified."""
    hotlist = self.services.features.TestAddHotlist(
        name='name', summary='summary', description='description',
        owner_ids=[self.user2.user_id], editor_ids=[], hotlist_id=1235,
        is_private=True)

    request = features_pb2.UpdateHotlistSettingsRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name=hotlist.name,
            owner=common_pb2.UserRef(user_id=self.user2.user_id)))

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user2.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.features_svcr.UpdateHotlistSettings, mc, request)
    mockUpdateHotlistSettings.assert_not_called()

  def testDeleteHotlist(self):
    """Test we can delete a hotlist via the API."""
    owner_ids = [self.user2.user_id]
    editor_ids = []
    hotlist = self.services.features.TestAddHotlist(
        name='Hotlist-1', summary='summary', description='description',
        owner_ids=owner_ids, editor_ids=editor_ids, hotlist_id=1235)
    request = features_pb2.DeleteHotlistRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name=hotlist.name,
            owner=common_pb2.UserRef(user_id=self.user2.user_id)))

    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user2.email)
    mc.LookupLoggedInUserPerms(None)
    self.CallWrapped(self.features_svcr.DeleteHotlist, mc, request)

    self.assertTrue(
        hotlist.hotlist_id in self.services.features.expunged_hotlist_ids)

  def testListHotlistPermissions_InvalidHotlistRef(self):
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user2.email)
    request = features_pb2.ListHotlistPermissionsRequest()
    with self.assertRaises(features_svc.NoSuchHotlistException):
      self.CallWrapped(self.features_svcr.ListHotlistPermissions, mc, request)

  def testListHotlistPermissions(self):
    owner_ids = [self.user1.user_id]
    editor_ids = [self.user3.user_id]
    hotlist = self.services.features.TestAddHotlist(
        name='Hotlist-1', summary='summary', description='description',
        owner_ids=owner_ids, editor_ids=editor_ids, hotlist_id=1235)

    request = features_pb2.ListHotlistPermissionsRequest(
        hotlist_ref=common_pb2.HotlistRef(
            name=hotlist.name, owner=common_pb2.UserRef(
                user_id=self.user1.user_id)))
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester=self.user3.email)
    mc.LookupLoggedInUserPerms(None)

    user_perms = self.CallWrapped(
        self.features_svcr.ListHotlistPermissions, mc, request)
    self.assertEqual(
        user_perms,
        features_pb2.ListHotlistPermissionsResponse(
            permissions=permissions.HOTLIST_EDITOR_PERMISSIONS))

  def testPredictComponent_Normal(self):
    """Test normal case when predicted component exists."""
    component_id = self.services.config.CreateComponentDef(
        cnxn=None, project_id=self.project.project_id, path='Ruta>Baga',
        docstring='', deprecated=False, admin_ids=[], cc_ids=[], created=None,
        creator_id=None, label_ids=[])

    self._top_words = {
        'foo': 0,
        'bar': 1,
        'baz': 2}
    self._components_by_index = {
        '0': '123',
        '1': str(component_id),
        '2': '789'}
    self._ml_engine.expected_features = [3, 0, 1, 0, 0]
    self._ml_engine.scores = [5, 10, 3]

    request = features_pb2.PredictComponentRequest(
        project_name='proj',
        text='foo baz foo foo')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    result = self.CallWrapped(self.features_svcr.PredictComponent, mc, request)

    self.assertEqual(
        common_pb2.ComponentRef(
            path='Ruta>Baga'),
        result.component_ref)

  def testPredictComponent_NoPrediction(self):
    """Test case when no component id was predicted."""
    self._top_words = {
        'foo': 0,
        'bar': 1,
        'baz': 2}
    self._components_by_index = {
        '0': '123',
        '1': '456',
        '2': '789'}
    self._ml_engine.expected_features = [3, 0, 1, 0, 0]
    self._ml_engine.scores = [5, 10, 3]

    request = features_pb2.PredictComponentRequest(
        project_name='proj',
        text='foo baz foo foo')
    mc = monorailcontext.MonorailContext(
        self.services, cnxn=self.cnxn, requester='owner@example.com')
    result = self.CallWrapped(self.features_svcr.PredictComponent, mc, request)

    self.assertEqual(common_pb2.ComponentRef(), result.component_ref)
