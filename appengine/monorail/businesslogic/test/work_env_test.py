# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is govered by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the WorkEnv class."""

import unittest

from google.appengine.api import memcache
from google.appengine.ext import testbed

from businesslogic import work_env
from framework import exceptions
from services import issue_svc
from services import project_svc
from services import service_manager
from testing import fake
from testing import testing_helpers


class WorkEnvTest(unittest.TestCase):

  def setUp(self):
    self.cnxn = 'fake connection'
    self.mr = testing_helpers.MakeMonorailRequest()
    self.services = service_manager.Services(
        config=fake.ConfigService(),
        issue=fake.IssueService(),
        user=fake.UserService(),
        project=fake.ProjectService(),
        issue_star=fake.IssueStarService(),
        spam=fake.SpamService())
    self.work_env = work_env.WorkEnv(
      self.mr, self.services, 'Testing phase')

  # FUTURE: GetSiteReadOnlyState()
  # FUTURE: SetSiteReadOnlyState()
  # FUTURE: GetSiteBannerMessage()
  # FUTURE: SetSiteBannerMessage()

  # FUTURE: CreateProject()
  # FUTURE: ListProjects()

  def testGetProject_Normal(self):
    """We can get an existing project by project_id."""
    project = self.services.project.TestAddProject('proj', project_id=789)
    with self.work_env as we:
      actual = we.GetProject(789)

    self.assertEqual(project, actual)

  def testGetProject_NoSuchProject(self):
    """We reject attempts to get a non-existent project."""
    with self.assertRaises(project_svc.NoSuchProjectException):
      with self.work_env as we:
        _actual = we.GetProject(999)

  def testGetProjectByName_Normal(self):
    """We can get an existing project by project_name."""
    project = self.services.project.TestAddProject('proj', project_id=789)
    with self.work_env as we:
      actual = we.GetProjectByName('proj')

    self.assertEqual(project, actual)

  def testGetProjectByName_NoSuchProject(self):
    """We reject attempts to get a non-existent project."""
    with self.assertRaises(project_svc.NoSuchProjectException):
      with self.work_env as we:
        _actual = we.GetProjectByName('huh-what')

  # FUTURE: UpdateProject()
  # FUTURE: DeleteProject()

  # FUTURE: SetProjectStar()
  # FUTURE: GetProjectStarsByUser()

  def testGetProjectConfig_Normal(self):
    """We can get an existing config by project_id."""
    config = fake.MakeTestConfig(789, ['LabelOne'], ['New'])
    self.services.config.StoreConfig('cnxn', config)
    with self.work_env as we:
      actual = we.GetProjectConfig(789)

    self.assertEqual(config, actual)

  def testGetProjectConfig_NoSuchProject(self):
    """We reject attempts to get a non-existent config."""
    self.services.config.strict = True
    with self.assertRaises(project_svc.NoSuchProjectException):
      with self.work_env as we:
        _actual = we.GetProjectConfig(789)

  # FUTURE: labels, statuses, fields, components, rules, templates, and views.
  # FUTURE: project saved queries.
  # FUTURE: GetProjectPermissionsForUser()

  # FUTURE: CreateIssue()
  # FUTURE: ListIssues()

  def testGetIssue_Normal(self):
    """We can get an existing issue by issue_id."""
    issue = fake.MakeTestIssue(789, 1, 'sum', 'New', 111L, issue_id=78901)
    self.services.issue.TestAddIssue(issue)
    with self.work_env as we:
      actual = we.GetIssue(78901)

    self.assertEqual(issue, actual)

  def testGetIssue_NoSuchIssue(self):
    """We reject attempts to get a non-existent issue."""
    with self.assertRaises(issue_svc.NoSuchIssueException):
      with self.work_env as we:
        _actual = we.GetIssue(78901)

  def testGetIssueByLocalID_Normal(self):
    """We can get an existing issue by project_id and local_id."""
    issue = fake.MakeTestIssue(789, 1, 'sum', 'New', 111L, issue_id=78901)
    self.services.issue.TestAddIssue(issue)
    with self.work_env as we:
      actual = we.GetIssueByLocalID(789, 1)

    self.assertEqual(issue, actual)

  def testGetIssueByLocalID_ProjectNotSpecified(self):
    """We reject calls with missing information."""
    with self.assertRaises(exceptions.InputException):
      with self.work_env as we:
        _actual = we.GetIssueByLocalID(None, 1)

  def testGetIssueByLocalID_IssueNotSpecified(self):
    """We reject calls with missing information."""
    with self.assertRaises(exceptions.InputException):
      with self.work_env as we:
        _actual = we.GetIssueByLocalID(789, None)

  def testGetIssueByLocalID_NoSuchIssue(self):
    """We reject attempts to get a non-existent issue."""
    with self.assertRaises(issue_svc.NoSuchIssueException):
      with self.work_env as we:
        _actual = we.GetIssueByLocalID(789, 1)

  # FUTURE: UpdateIssue()
  # FUTURE: DeleteIssue()
  # FUTURE: GetIssuePermissionsForUser()

  # FUTURE: CreateComment()

  def testGetIssueComments_Normal(self):
    """We can get an existing issue by project_id and local_id."""
    issue = fake.MakeTestIssue(789, 1, 'sum', 'New', 111L, issue_id=78901)
    self.services.issue.TestAddIssue(issue)

    with self.work_env as we:
      actual = we.GetIssueByLocalID(789, 1)

    self.assertEqual(issue, actual)


  # FUTURE: UpdateComment()
  # FUTURE: DeleteComment()

  # FUTURE: SetIssueStar()
  # FUTURE: GetIssueStars()
  # FUTURE: GetIssueStarsByUser()

  # FUTURE: GetUser()
  # FUTURE: UpdateUser()
  # FUTURE: DeleteUser()

  # FUTURE: CreateGroup()
  # FUTURE: ListGroups()
  # FUTURE: UpdateGroup()
  # FUTURE: DeleteGroup()

  # FUTURE: CreateHotlist()
  # FUTURE: ListHotlistsByUser()
  # FUTURE: UpdateHotlist()
  # FUTURE: DeleteHotlist()
