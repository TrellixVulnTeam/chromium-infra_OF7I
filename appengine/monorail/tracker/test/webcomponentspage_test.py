# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd
"""Tests for the Monorail SPA pages, as served by EZT."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import mock
import unittest

from third_party import ezt

import settings
from framework import permissions
from proto import project_pb2
from proto import site_pb2
from services import service_manager
from tracker import webcomponentspage
from testing import fake
from testing import testing_helpers


class WebComponentsPageTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(
        user=fake.UserService(),
        project=fake.ProjectService(),
        features=fake.FeaturesService())

    self.user = self.services.user.TestAddUser('user@example.com', 111)
    self.project = self.services.project.TestAddProject('proj', project_id=789)
    self.hotlist = self.services.features.TestAddHotlist(
        'HotlistName', summary='summary', owner_ids=[111], hotlist_id=1236)

    self.servlet = webcomponentspage.WebComponentsPage(
        'req', 'res', services=self.services)

  def testHotlistPage_OldUiUrl(self):
    mr = testing_helpers.MakeMonorailRequest(
        user_info={'user_id': 111},
        path='/hotlists/1236',
        services=self.services)

    page_data = self.servlet.GatherPageData(mr)
    self.assertEqual('/u/111/hotlists/HotlistName', page_data['old_ui_url'])

  def testHotlistPage_OldUiUrl_People(self):
    mr = testing_helpers.MakeMonorailRequest(
        user_info={'user_id': 111},
        path='/hotlists/1236/people',
        services=self.services)

    page_data = self.servlet.GatherPageData(mr)
    self.assertEqual(
        '/u/111/hotlists/HotlistName/people', page_data['old_ui_url'])

  def testHotlistPage_OldUiUrl_Settings(self):
    mr = testing_helpers.MakeMonorailRequest(
        user_info={'user_id': 111},
        path='/hotlists/1236/settings',
        services=self.services)

    page_data = self.servlet.GatherPageData(mr)
    self.assertEqual(
        '/u/111/hotlists/HotlistName/details', page_data['old_ui_url'])


class ProjectListPageTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(project=fake.ProjectService())

    self.project_a = self.services.project.TestAddProject('a', project_id=1)
    self.project_b = self.services.project.TestAddProject('b', project_id=2)

    self.servlet = webcomponentspage.ProjectListPage(
        'req', 'res', services=self.services)

  @mock.patch('settings.domain_to_default_project', {})
  def testMaybeRedirectToDomainDefaultProject_NoMatch(self):
    """No redirect if the user is not accessing via a configured domain."""
    mr = testing_helpers.MakeMonorailRequest()
    mr.request.host = 'example.com'
    msg = self.servlet._MaybeRedirectToDomainDefaultProject(mr)
    print('msg: ' + msg)
    self.assertTrue(msg.startswith('No configured'))

  @mock.patch('settings.domain_to_default_project', {'example.com': 'huh'})
  def testMaybeRedirectToDomainDefaultProject_NoSuchProject(self):
    """No redirect if the configured project does not exist."""
    mr = testing_helpers.MakeMonorailRequest()
    mr.request.host = 'example.com'
    print('host is %r' % mr.request.host)
    msg = self.servlet._MaybeRedirectToDomainDefaultProject(mr)
    print('msg: ' + msg)
    self.assertTrue(msg.endswith('not found'))

  @mock.patch('settings.domain_to_default_project', {'example.com': 'a'})
  def testMaybeRedirectToDomainDefaultProject_CantView(self):
    """No redirect if the user can't view the configured project."""
    self.project_a.access = project_pb2.ProjectAccess.MEMBERS_ONLY
    mr = testing_helpers.MakeMonorailRequest()
    mr.request.host = 'example.com'
    msg = self.servlet._MaybeRedirectToDomainDefaultProject(mr)
    print('msg: ' + msg)
    self.assertTrue(msg.startswith('User cannot'))

  @mock.patch('settings.domain_to_default_project', {'example.com': 'a'})
  def testMaybeRedirectToDomainDefaultProject_Redirect(self):
    """We redirect if there's a configured project that the user can view."""
    mr = testing_helpers.MakeMonorailRequest()
    mr.request.host = 'example.com'
    self.servlet.redirect = mock.Mock()
    msg = self.servlet._MaybeRedirectToDomainDefaultProject(mr)
    print('msg: ' + msg)
    self.assertTrue(msg.startswith('Redirected'))
    self.servlet.redirect.assert_called_once()
