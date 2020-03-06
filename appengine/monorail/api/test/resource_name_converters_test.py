# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for converting between resource names and external ids."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

from mock import Mock, patch
import unittest
import re

from api import resource_name_converters as rnc
from framework import exceptions
from testing import fake
from services import service_manager

class ResourceNameConverterTest(unittest.TestCase):

  def setUp(self):
    self.services = service_manager.Services(
        issue=fake.IssueService(),
        project=fake.ProjectService(),
        user=fake.UserService(),
        features=fake.FeaturesService(),
        template=fake.TemplateService(),
        config=fake.ConfigService())
    self.cnxn = fake.MonorailConnection()
    self.PAST_TIME = 12345
    self.project_1 = self.services.project.TestAddProject(
        'proj', project_id=789)
    self.project_2 = self.services.project.TestAddProject(
        'goose', project_id=788)
    self.dne_project_id = 1999

    self.issue_1 = fake.MakeTestIssue(
        self.project_1.project_id, 1, 'sum', 'New', 111,
        project_name=self.project_1.project_name)
    self.issue_2 = fake.MakeTestIssue(
        self.project_2.project_id, 2, 'sum', 'New', 111,
        project_name=self.project_2.project_name)
    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)

    self.user_1 = self.services.user.TestAddUser('user_111@example.com', 111)
    self.user_2 = self.services.user.TestAddUser('user_222@example.com', 222)
    self.user_3 = self.services.user.TestAddUser('user_333@example.com', 333)

    hotlist_items = [
        (self.issue_1.issue_id, 9, self.user_2.user_id, self.PAST_TIME, 'note'),
        (self.issue_2.issue_id, 1, self.user_1.user_id, self.PAST_TIME, 'note')]
    self.hotlist_1 = self.services.features.TestAddHotlist(
        'HotlistName', owner_ids=[], editor_ids=[],
        hotlist_item_fields=hotlist_items)

    self.template_1 = self.services.template.TestAddIssueTemplateDef(
        1, 789, 'template_1_name')

    self.field_def_1_name = 'test_field'
    self.field_def_1 = self.services.config.CreateFieldDef(
        self.cnxn, self.project_1.project_id, self.field_def_1_name, 'STR_TYPE',
        None, None, None, None, None, None, None, None, None, None, None, None,
        None, None, [], [])
    self.dne_field_def_id = 999999

  def testGetResourceNameMatch(self):
    """We can get a resource name match."""
    regex = re.compile(r'name\/(?P<group_name>[a-z]+)$')
    match = rnc._GetResourceNameMatch('name/honque', regex)
    self.assertEqual(match.group('group_name'), 'honque')

  def testGetResouceNameMatch_InvalidName(self):
    """An exception is raised if there is not match."""
    regex = re.compile(r'name\/(?P<group_name>[a-z]+)$')
    with self.assertRaises(exceptions.InputException):
      rnc._GetResourceNameMatch('honque/honque', regex)

  def testIngestHotlistName(self):
    """We can get a Hotlist's resource name match."""
    self.assertEqual(rnc.IngestHotlistName('hotlists/78909'), 78909)

  def testIngestHotlistName_InvalidName(self):
    """An exception is raised if the Hotlist's resource name is invalid"""
    with self.assertRaises(exceptions.InputException):
      rnc.IngestHotlistName('hotlists/789honk789')

  def testIngestHotlistItemNames(self):
    """We can get Issue IDs from HotlistItems resource names."""
    names = [
        'hotlists/78909/items/proj.1',
        'hotlists/78909/items/goose.2']
    self.assertEqual(
        rnc.IngestHotlistItemNames(self.cnxn, names, self.services),
        [self.issue_1.issue_id, self.issue_2.issue_id])

  def testIngestHotlistItemNames_ProjectNotFound(self):
    """Exception is raised if a project is not found."""
    names = [
        'hotlists/78909/items/proj.1',
        'hotlists/78909/items/chicken.2']
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestHotlistItemNames(self.cnxn, names, self.services)

  def testIngestHotlistItems_IssueNotFound(self):
    """Exception is raised if an Issue is not found."""
    names = [
        'hotlists/78909/items/proj.1',
        'hotlists/78909/items/goose.5']
    with self.assertRaises(exceptions.NoSuchIssueException):
      rnc.IngestHotlistItemNames(self.cnxn, names, self.services)

  def testConvertHotlistName(self):
    """We can get a Hotlist's resource name."""
    self.assertEqual(rnc.ConvertHotlistName(10), 'hotlists/10')

  def testConvertHotlistItemNames(self):
    """We can get Hotlist items' resource names."""
    expected_dict = {
        self.hotlist_1.items[0].issue_id: 'hotlists/7739/items/proj.1',
        self.hotlist_1.items[1].issue_id: 'hotlists/7739/items/goose.2',
    }
    self.assertEqual(
        rnc.ConvertHotlistItemNames(
            self.cnxn, self.hotlist_1.hotlist_id, expected_dict.keys(),
            self.services), expected_dict)

  def testIngestIssueName(self):
    """We can get an Issue global id from its resource name."""
    self.assertEqual(
        rnc.IngestIssueName(self.cnxn, 'projects/proj/issues/1', self.services),
        self.issue_1.issue_id)

  def testIngestIssueName_ProjectDoesNotExist(self):
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestIssueName(self.cnxn, 'projects/noproj/issues/1', self.services)

  def testIngestIssueName_IssueDoesNotExist(self):
    with self.assertRaises(exceptions.NoSuchIssueException):
      rnc.IngestIssueName(self.cnxn, 'projects/proj/issues/2', self.services)

  def testIngestIssueName_InvalidLocalId(self):
    """Issue resource name Local IDs are digits."""
    with self.assertRaises(exceptions.InputException):
      rnc.IngestIssueName(self.cnxn, 'projects/proj/issues/x', self.services)

  def testIngestIssueName_InvalidProjectId(self):
    """Project names are more than 1 character."""
    with self.assertRaises(exceptions.InputException):
      rnc.IngestIssueName(self.cnxn, 'projects/p/issues/1', self.services)

  def testIngestIssueName_InvalidFormat(self):
    """Issue resource names must begin with the project resource name."""
    with self.assertRaises(exceptions.InputException):
      rnc.IngestIssueName(self.cnxn, 'issues/1', self.services)

  def testIngestIssueNames(self):
    """We can get an Issue global ids from resource names."""
    self.assertEqual(
        rnc.IngestIssueNames(
            self.cnxn, ['projects/proj/issues/1', 'projects/goose/issues/2'],
            self.services), [self.issue_1.issue_id, self.issue_2.issue_id])

  def testIngestIssueNames_EmptyList(self):
    """We get an empty list when providing an empty list of issue names."""
    self.assertEqual(rnc.IngestIssueNames(self.cnxn, [], self.services), [])

  def testIngestIssueNames_OneDoesNotExist(self):
    """We get an exception if one issue name provided does not exist."""
    with self.assertRaises(exceptions.NoSuchIssueException):
      rnc.IngestIssueNames(
          self.cnxn, ['projects/proj/issues/1', 'projects/proj/issues/2'],
          self.services)

  def testConvertIssueName(self):
    """We can create an Issue resource name from an issue_id."""
    self.assertEqual(
        rnc.ConvertIssueName(self.cnxn, self.issue_1.issue_id, self.services),
        'projects/proj/issues/1')

  def testConvertIssueName_NotFound(self):
    """Exception is raised if the issue is not found."""
    with self.assertRaises(exceptions.NoSuchIssueException):
      rnc.ConvertIssueName(self.cnxn, 3279, self.services)

  def testConvertIssueNames(self):
    """We can create Issue resource names from issue_ids."""
    self.assertEqual(
        rnc.ConvertIssueNames(
            self.cnxn, [self.issue_1.issue_id, 3279], self.services),
        {self.issue_1.issue_id: 'projects/proj/issues/1'})

  def testIngestUserNames(self):
    """We can get User IDs from User resource names."""
    names = ['users/111', 'users/222']
    expected_ids = [111, 222]
    self.assertEqual(rnc.IngestUserNames(names), expected_ids)

  def testConvertUserNames(self):
    """We can get User resource names."""
    expected_dict = {111: 'users/111', 222: 'users/222', 333: 'users/333'}
    self.assertEqual(rnc.ConvertUserNames(expected_dict.keys()), expected_dict)

  def testConvertUserNames_Empty(self):
    """We can process an empty Users list."""
    self.assertEqual(rnc.ConvertUserNames([]), {})

  def testIngestProjectName(self):
    """We can get project name from Project resource names."""
    name = 'projects/{}'.format(self.project_1.project_name)
    expected = self.project_1.project_id
    self.assertEqual(
        rnc.IngestProjectName(self.cnxn, name, self.services), expected)

  def testIngestProjectName_InvalidName(self):
    """An exception is raised if the Hotlist's resource name is invalid"""
    with self.assertRaises(exceptions.InputException):
      rnc.IngestProjectName(self.cnxn, 'projects/', self.services)

  def testConvertTemplateNames(self):
    """We can get IssueTemplate resource names."""
    expected_resource_name = 'projects/{}/templates/{}'.format(
        self.project_1.project_name, self.template_1.name)
    expected = {self.template_1.template_id: expected_resource_name}

    self.assertEqual(
        rnc.ConvertTemplateNames(
            self.cnxn, self.project_1.project_id, [self.template_1.template_id],
            self.services), expected)

  def testConvertTemplateNames_NoSuchProjectException(self):
    """We get an exception if project with id does not exist."""
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertTemplateNames(
          self.cnxn, self.dne_project_id, [self.template_1.template_id],
          self.services)

  def testConvertStatusDefName(self):
    """We can get Status resource name."""
    expected_resource_name = 'projects/{}/statusDefs/{}'.format(
        self.project_1.project_name, self.issue_1.status)

    self.assertEqual(
        rnc.ConvertStatusDefName(
            self.cnxn, self.issue_1.status, self.project_1.project_id,
            self.services), expected_resource_name)

  def testConvertStatusDefName_NoSuchProjectException(self):
    """We can get an exception if project with id does not exist."""
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertStatusDefName(
          self.cnxn, self.issue_1.status, self.dne_project_id, self.services)

  def testConvertLabelDefNames(self):
    """We can get Label resource names."""
    expected_label = 'some label'
    expected_resource_name = 'projects/{}/labelDefs/{}'.format(
        self.project_1.project_name, expected_label)

    self.assertEqual(
        rnc.ConvertLabelDefNames(
            self.cnxn, [expected_label], self.project_1.project_id,
            self.services), {expected_label: expected_resource_name})

  def testConvertLabelDefNames_NoSuchProjectException(self):
    """We can get an exception if project with id does not exist."""
    some_label = 'some label'
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertLabelDefNames(
          self.cnxn, [some_label], self.dne_project_id, self.services)

  def testConvertComponentDefNames(self):
    """We can get Component resource names."""
    expected_id = 123456
    expected_resource_name = 'projects/{}/componentDefs/{}'.format(
        self.project_1.project_name, expected_id)

    self.assertEqual(
        rnc.ConvertComponentDefNames(
            self.cnxn, [expected_id], self.project_1.project_id, self.services),
        {expected_id: expected_resource_name})

  def testConvertComponentDefNames_NoSuchProjectException(self):
    """We can get an exception if project with id does not exist."""
    component_id = 123456
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertComponentDefNames(
          self.cnxn, [component_id], self.dne_project_id, self.services)

  def testConvertFieldDefNames(self):
    expected_key = self.field_def_1
    expected_value = 'projects/{}/fieldDefs/{}'.format(
        self.project_1.project_name, self.field_def_1_name)

    field_ids = [self.field_def_1, self.dne_field_def_id]
    self.assertEqual(
        rnc.ConvertFieldDefNames(
            self.cnxn, field_ids, self.project_1.project_id, self.services),
        {expected_key: expected_value})

  def testConvertFieldDefNames_NoSuchProjectException(self):
    field_ids = [self.field_def_1, self.dne_field_def_id]
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertFieldDefNames(
          self.cnxn, field_ids, self.dne_project_id, self.services)
