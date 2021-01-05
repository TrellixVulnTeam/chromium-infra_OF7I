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
from proto import tracker_pb2

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
        1, self.project_1.project_id, 'template_1_name')
    self.template_2 = self.services.template.TestAddIssueTemplateDef(
        2, self.project_2.project_id, 'template_2_name')
    self.dne_template_id = 3

    self.field_def_1_name = 'test_field'
    self.field_def_1 = self.services.config.CreateFieldDef(
        self.cnxn, self.project_1.project_id, self.field_def_1_name, 'STR_TYPE',
        None, None, None, None, None, None, None, None, None, None, None, None,
        None, None, [], [])
    self.approval_def_1_name = 'approval_field_1'
    self.approval_def_1_id = self.services.config.CreateFieldDef(
        self.cnxn, self.project_1.project_id, self.approval_def_1_name,
        'APPROVAL_TYPE', None, None, None, None, None, None, None, None, None,
        None, None, None, None, None, [], [])
    self.component_def_1_path = 'Foo'
    self.component_def_1_id = self.services.config.CreateComponentDef(
        self.cnxn, self.project_1.project_id, self.component_def_1_path, '',
        False, [], [], None, 111, [])
    self.component_def_2_path = 'Foo>Bar'
    self.component_def_2_id = self.services.config.CreateComponentDef(
        self.cnxn, self.project_1.project_id, self.component_def_2_path, '',
        False, [], [], None, 111, [])
    self.component_def_3_path = 'Fizz'
    self.component_def_3_id = self.services.config.CreateComponentDef(
        self.cnxn, self.project_2.project_id, self.component_def_3_path, '',
        False, [], [], None, 111, [])
    self.dne_component_def_id = 999
    self.dne_field_def_id = 999999
    self.psq_1 = tracker_pb2.SavedQuery(
        query_id=2, name='psq1 name', base_query_id=1, query='foo=bar')
    self.psq_2 = tracker_pb2.SavedQuery(
        query_id=3, name='psq2 name', base_query_id=1, query='fizz=buzz')
    self.dne_psq_id = 987
    self.services.features.UpdateCannedQueries(
        self.cnxn, self.project_1.project_id, [self.psq_1, self.psq_2])

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

  def testIngestApprovalDefName(self):
    approval_id = rnc.IngestApprovalDefName(
        self.cnxn, 'projects/proj/approvalDefs/123', self.services)
    self.assertEqual(approval_id, 123)

  def testIngestApprovalDefName_InvalidName(self):
    with self.assertRaises(exceptions.InputException):
      rnc.IngestApprovalDefName(
          self.cnxn, 'projects/proj/approvalDefs/123d', self.services)

    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestApprovalDefName(
          self.cnxn, 'projects/garbage/approvalDefs/123', self.services)

  def testIngestFieldDefName(self):
    """We can get a FieldDef's resource name match."""
    self.assertEqual(
        rnc.IngestFieldDefName(
            self.cnxn, 'projects/proj/fieldDefs/123', self.services),
        (789, 123))

  def testIngestFieldDefName_InvalidName(self):
    """An exception is raised if the FieldDef's resource name is invalid"""
    with self.assertRaises(exceptions.InputException):
      rnc.IngestFieldDefName(
          self.cnxn, 'projects/proj/fieldDefs/7Dog', self.services)

    with self.assertRaises(exceptions.InputException):
      rnc.IngestFieldDefName(
          self.cnxn, 'garbage/proj/fieldDefs/123', self.services)

    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestFieldDefName(
          self.cnxn, 'projects/garbage/fieldDefs/123', self.services)

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

  def testIngestHotlistItemNames_MultipleProjectsNotFound(self):
    """Aggregated exceptions raised if projects are not found."""
    names = [
        'hotlists/78909/items/proj.1', 'hotlists/78909/items/chicken.2',
        'hotlists/78909/items/cow.3'
    ]
    with self.assertRaisesRegexp(exceptions.NoSuchProjectException,
                                 'Project chicken not found.\n' +
                                 'Project cow not found.'):
      rnc.IngestHotlistItemNames(self.cnxn, names, self.services)

  def testIngestHotlistItems_IssueNotFound(self):
    """Exception is raised if an Issue is not found."""
    names = [
        'hotlists/78909/items/proj.1',
        'hotlists/78909/items/goose.5']
    with self.assertRaisesRegexp(exceptions.NoSuchIssueException, '%r' % names):
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

  def testIngestApprovalValueName(self):
    project_id, issue_id, approval_def_id = rnc.IngestApprovalValueName(
        self.cnxn, 'projects/proj/issues/1/approvalValues/404', self.services)
    self.assertEqual(project_id, self.project_1.project_id)
    self.assertEqual(issue_id, self.issue_1.issue_id)
    self.assertEqual(404, approval_def_id)  # We don't verify it exists.

  def testIngestApprovalValueName_ProjectDoesNotExist(self):
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestApprovalValueName(
          self.cnxn, 'projects/noproj/issues/1/approvalValues/1', self.services)

  def testIngestApprovalValueName_IssueDoesNotExist(self):
    with self.assertRaisesRegexp(exceptions.NoSuchIssueException,
                                 'projects/proj/issues/404'):
      rnc.IngestApprovalValueName(
          self.cnxn, 'projects/proj/issues/404/approvalValues/1', self.services)

  def testIngestApprovalValueName_InvalidStart(self):
    with self.assertRaises(exceptions.InputException):
      rnc.IngestApprovalValueName(
          self.cnxn, 'zprojects/proj/issues/1/approvalValues/1', self.services)

  def testIngestApprovalValueName_InvalidEnd(self):
    with self.assertRaises(exceptions.InputException):
      rnc.IngestApprovalValueName(
          self.cnxn, 'projects/proj/issues/1/approvalValues/1z', self.services)

  def testIngestApprovalValueName_InvalidCollection(self):
    with self.assertRaises(exceptions.InputException):
      rnc.IngestApprovalValueName(
          self.cnxn, 'projects/proj/issues/1/approvalValue/1', self.services)

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

  def testIngestIssueName_Moved(self):
    """We can get a moved issue."""
    moved_to_project_id = 987
    self.services.project.TestAddProject(
        'other', project_id=moved_to_project_id)
    new_issue_id = 1010
    issue = fake.MakeTestIssue(
        moved_to_project_id, 200, 'sum', 'New', 111, issue_id=new_issue_id)
    self.services.issue.TestAddIssue(issue)
    self.services.issue.TestAddMovedIssueRef(
        self.project_1.project_id, 404, moved_to_project_id, 200)

    self.assertEqual(
        rnc.IngestIssueName(
            self.cnxn, 'projects/proj/issues/404', self.services), new_issue_id)

  def testIngestIssueNames(self):
    """We can get an Issue global ids from resource names."""
    self.assertEqual(
        rnc.IngestIssueNames(
            self.cnxn, ['projects/proj/issues/1', 'projects/goose/issues/2'],
            self.services), [self.issue_1.issue_id, self.issue_2.issue_id])

  def testIngestIssueNames_EmptyList(self):
    """We get an empty list when providing an empty list of issue names."""
    self.assertEqual(rnc.IngestIssueNames(self.cnxn, [], self.services), [])

  def testIngestIssueNames_WithBadInputs(self):
    """We aggregate input exceptions."""
    with self.assertRaisesRegexp(
        exceptions.InputException,
        'Invalid resource name: projects/proj/badformat/1.\n' +
        'Invalid resource name: badformat/proj/issues/1.'):
      rnc.IngestIssueNames(
          self.cnxn, [
              'projects/proj/badformat/1', 'badformat/proj/issues/1',
              'projects/proj/issues/1'
          ], self.services)

  def testIngestIssueNames_OneDoesNotExist(self):
    """We get an exception if one issue name provided does not exist."""
    with self.assertRaises(exceptions.NoSuchIssueException):
      rnc.IngestIssueNames(
          self.cnxn, ['projects/proj/issues/1', 'projects/proj/issues/2'],
          self.services)

  def testIngestIssueNames_ManyDoNotExist(self):
    """We get an exception if one issue name provided does not exist."""
    dne_issues = ['projects/proj/issues/2', 'projects/proj/issues/3']
    with self.assertRaisesRegexp(exceptions.NoSuchIssueException,
                                 '%r' % dne_issues):
      rnc.IngestIssueNames(self.cnxn, dne_issues, self.services)

  def testIngestIssueNames_ProjectsNotExist(self):
    """Aggregated exceptions raised if projects are not found."""
    with self.assertRaisesRegexp(exceptions.NoSuchProjectException,
                                 'Project chicken not found.\n' +
                                 'Project cow not found.'):
      rnc.IngestIssueNames(
          self.cnxn, [
              'projects/chicken/issues/2', 'projects/cow/issues/3',
              'projects/proj/issues/1'
          ], self.services)

  def testIngestProjectFromIssue(self):
    self.assertEqual(rnc.IngestProjectFromIssue('projects/xyz/issues/1'), 'xyz')

  def testIngestProjectFromIssue_InvalidName(self):
    with self.assertRaises(exceptions.InputException):
      rnc.IngestProjectFromIssue('projects/xyz')
    with self.assertRaises(exceptions.InputException):
      rnc.IngestProjectFromIssue('garbage')
    with self.assertRaises(exceptions.InputException):
      rnc.IngestProjectFromIssue('projects/xyz/issues/garbage')

  def testIngestCommentName(self):
    name = 'projects/proj/issues/1/comments/0'
    actual = rnc.IngestCommentName(self.cnxn, name, self.services)
    self.assertEqual(
        actual, (self.project_1.project_id, self.issue_1.issue_id, 0))

  def testIngestCommentName_InputException(self):
    with self.assertRaises(exceptions.InputException):
      rnc.IngestCommentName(self.cnxn, 'misspelled name', self.services)

  def testIngestCommentName_NoSuchProject(self):
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestCommentName(
          self.cnxn, 'projects/doesnotexist/issues/1/comments/0', self.services)

  def testIngestCommentName_NoSuchIssue(self):
    with self.assertRaisesRegexp(exceptions.NoSuchIssueException,
                                 "['projects/proj/issues/404']"):
      rnc.IngestCommentName(
          self.cnxn, 'projects/proj/issues/404/comments/0', self.services)

  def testConvertCommentNames(self):
    """We can create comment names."""
    expected = {
        0: 'projects/proj/issues/1/comments/0',
        1: 'projects/proj/issues/1/comments/1'
    }
    self.assertEqual(rnc.CreateCommentNames(1, 'proj', [0, 1]), expected)

  def testConvertCommentNames_Empty(self):
    """Converting an empty list of comments returns an empty dict."""
    self.assertEqual(rnc.CreateCommentNames(1, 'proj', []), {})

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

  def testConvertApprovalValueNames(self):
    """We can create ApprovalValue resource names."""
    self.issue_1.approval_values = [tracker_pb2.ApprovalValue(
        approval_id=self.approval_def_1_id)]

    expected_name = (
        'projects/proj/issues/1/approvalValues/%d' % self.approval_def_1_id)
    self.assertEqual(
        {self.approval_def_1_id: expected_name},
        rnc.ConvertApprovalValueNames(
            self.cnxn, self.issue_1.issue_id, self.services))

  def testIngestUserName(self):
    """We can get a User ID from User resource name."""
    name = 'users/111'
    self.assertEqual(rnc.IngestUserName(self.cnxn, name, self.services), 111)

  def testIngestUserName_DisplayName(self):
    """We can get a User ID from User resource name with a display name set."""
    name = 'users/%s' % self.user_3.email
    self.assertEqual(rnc.IngestUserName(self.cnxn, name, self.services), 333)

  def testIngestUserName_NoSuchUser(self):
    """When autocreate=False, we raise an exception if a user is not found."""
    name = 'users/chicken@test.com'
    with self.assertRaises(exceptions.NoSuchUserException):
      rnc.IngestUserName(self.cnxn, name, self.services)

  def testIngestUserName_InvalidEmail(self):
    """We raise an exception if a given resource name's email is invalid."""
    name = 'users/chickentest.com'
    with self.assertRaises(exceptions.InputException):
      rnc.IngestUserName(self.cnxn, name, self.services)

  def testIngestUserName_Autocreate(self):
    """When autocreate=True create a new User if they don't already exist."""
    new_email = 'chicken@test.com'
    name = 'users/%s' % new_email
    user_id = rnc.IngestUserName(
        self.cnxn, name, self.services, autocreate=True)

    new_id = self.services.user.LookupUserID(
        self.cnxn, new_email, autocreate=False)
    self.assertEqual(new_id, user_id)

  def testIngestUserNames(self):
    """We can get User IDs from User resource names."""
    names = ['users/111', 'users/222', 'users/%s' % self.user_3.email]
    expected_ids = [111, 222, 333]
    self.assertEqual(
        rnc.IngestUserNames(self.cnxn, names, self.services), expected_ids)

  def testIngestUserNames_NoSuchUser(self):
    """When autocreate=False, we raise an exception if a user is not found."""
    names = [
        'users/111', 'users/chicken@test.com',
        'users/%s' % self.user_3.email
    ]
    with self.assertRaises(exceptions.NoSuchUserException):
      rnc.IngestUserNames(self.cnxn, names, self.services)

  def testIngestUserNames_InvalidEmail(self):
    """We raise an exception if a given resource name's email is invalid."""
    names = [
        'users/111', 'users/chickentest.com',
        'users/%s' % self.user_3.email
    ]
    with self.assertRaises(exceptions.InputException):
      rnc.IngestUserNames(self.cnxn, names, self.services)

  def testIngestUserNames_Autocreate(self):
    """When autocreate=True we create new Users if they don't already exist."""
    new_email = 'user_444@example.com'
    names = [
        'users/111',
        'users/%s' % new_email,
        'users/%s' % self.user_3.email
    ]
    ids = rnc.IngestUserNames(self.cnxn, names, self.services, autocreate=True)

    new_id = self.services.user.LookupUserID(
        self.cnxn, new_email, autocreate=False)
    expected_ids = [111, new_id, 333]
    self.assertEqual(expected_ids, ids)

  def testConvertUserName(self):
    """We can convert a single User ID to resource name."""
    self.assertEqual(rnc.ConvertUserName(111), 'users/111')

  def testConvertUserNames(self):
    """We can get User resource names."""
    expected_dict = {111: 'users/111', 222: 'users/222', 333: 'users/333'}
    self.assertEqual(rnc.ConvertUserNames(expected_dict.keys()), expected_dict)

  def testConvertUserNames_Empty(self):
    """We can process an empty Users list."""
    self.assertEqual(rnc.ConvertUserNames([]), {})

  def testConvertProjectStarName(self):
    """We can convert a User ID and Project ID to resource name."""
    name = rnc.ConvertProjectStarName(
        self.cnxn, 111, self.project_1.project_id, self.services)
    expected = 'users/111/projectStars/{}'.format(self.project_1.project_name)
    self.assertEqual(name, expected)

  def testConvertProjectStarName_NoSuchProjectException(self):
    """Throws an exception when Project ID is invalid."""
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertProjectStarName(self.cnxn, 111, 123455, self.services)

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
        self.project_1.project_name, self.template_1.template_id)
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

  def testConvertTemplateNames_NonExistentTemplate(self):
    """We only return templates that exist."""
    self.assertEqual(
        rnc.ConvertTemplateNames(
            self.cnxn, self.project_1.project_id, [self.dne_template_id],
            self.services), {})

  def testConvertTemplateNames_TemplateInProject(self):
    """We only return templates in the project."""
    expected_resource_name = 'projects/{}/templates/{}'.format(
        self.project_2.project_name, self.template_2.template_id)
    expected = {self.template_2.template_id: expected_resource_name}

    self.assertEqual(
        rnc.ConvertTemplateNames(
            self.cnxn, self.project_2.project_id,
            [self.template_1.template_id, self.template_2.template_id],
            self.services), expected)

  def testIngestTemplateName(self):
    name = 'projects/{}/templates/{}'.format(
        self.project_1.project_name, self.template_1.template_id)
    actual = rnc.IngestTemplateName(self.cnxn, name, self.services)
    expected = (self.template_1.template_id, self.project_1.project_id)
    self.assertEqual(actual, expected)

  def testIngestTemplateName_DoesNotExist(self):
    """We will ingest templates that don't exist."""
    name = 'projects/{}/templates/{}'.format(
        self.project_1.project_name, self.dne_template_id)
    actual = rnc.IngestTemplateName(self.cnxn, name, self.services)
    expected = (self.dne_template_id, self.project_1.project_id)
    self.assertEqual(actual, expected)

  def testIngestTemplateName_InvalidName(self):
    with self.assertRaises(exceptions.InputException):
      rnc.IngestTemplateName(
          self.cnxn, 'projects/asdf/misspelled_template/123', self.services)

    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestTemplateName(
          self.cnxn, 'projects/asdf/templates/123', self.services)

  def testConvertStatusDefNames(self):
    """We can get Status resource name."""
    expected_resource_name = 'projects/{}/statusDefs/{}'.format(
        self.project_1.project_name, self.issue_1.status)

    actual = rnc.ConvertStatusDefNames(
        self.cnxn, [self.issue_1.status], self.project_1.project_id,
        self.services)
    self.assertEqual(actual[self.issue_1.status], expected_resource_name)

  def testConvertStatusDefNames_NoSuchProjectException(self):
    """We can get an exception if project with id does not exist."""
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertStatusDefNames(
          self.cnxn, [self.issue_1.status], self.dne_project_id, self.services)

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

  def testIngestComponentDefNames(self):
    names = [
        'projects/proj/componentDefs/%d' % self.component_def_1_id,
        'projects/proj/componentDefs/%s' % self.component_def_2_path.lower()
    ]
    actual = rnc.IngestComponentDefNames(self.cnxn, names, self.services)
    self.assertEqual(actual, [
        (self.project_1.project_id, self.component_def_1_id),
        (self.project_1.project_id, self.component_def_2_id)])

  def testIngestComponentDefNames_NoSuchProjectException(self):
    names = ['projects/xyz/componentDefs/%d' % self.component_def_1_id]
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestComponentDefNames(self.cnxn, names, self.services)

    names = ['projects/xyz/componentDefs/1', 'projects/zyz/componentDefs/1']
    expected_error = 'Project not found: xyz.\nProject not found: zyz.'
    with self.assertRaisesRegexp(exceptions.NoSuchProjectException,
                                 expected_error):
      rnc.IngestComponentDefNames(self.cnxn, names, self.services)

  def testIngestComponentDefNames_NoSuchComponentException(self):
    names = ['projects/proj/componentDefs/%d' % self.dne_component_def_id]
    with self.assertRaises(exceptions.NoSuchComponentException):
      rnc.IngestComponentDefNames(self.cnxn, names, self.services)

    names = [
        'projects/proj/componentDefs/999', 'projects/proj/componentDefs/cow'
    ]
    expected_error = 'Component not found: 999.\nComponent not found: \'cow\'.'
    with self.assertRaisesRegexp(exceptions.NoSuchComponentException,
                                 expected_error):
      rnc.IngestComponentDefNames(self.cnxn, names, self.services)

  def testIngestComponentDefNames_InvalidNames(self):
    with self.assertRaises(exceptions.InputException):
      rnc.IngestHotlistName('projects/proj/componentDefs/not.path.or.id')

  def testIngestComponentDefNames_CrossProject(self):
    names = [
        'projects/proj/componentDefs/%d' % self.component_def_1_id,
        'projects/goose/componentDefs/%s' % self.component_def_3_path
    ]
    actual = rnc.IngestComponentDefNames(self.cnxn, names, self.services)
    self.assertEqual(actual, [
        (self.project_1.project_id, self.component_def_1_id),
        (self.project_2.project_id, self.component_def_3_id)])

  def testConvertFieldDefNames(self):
    """Returns resource names for fields that exist and ignores others."""
    expected_key = self.field_def_1
    expected_value = 'projects/{}/fieldDefs/{}'.format(
        self.project_1.project_name, self.field_def_1)

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

  def testConvertApprovalDefNames(self):
    outcome = rnc.ConvertApprovalDefNames(
        self.cnxn, [self.approval_def_1_id], self.project_1.project_id,
        self.services)

    expected_key = self.approval_def_1_id
    expected_value = 'projects/{}/approvalDefs/{}'.format(
        self.project_1.project_name, self.approval_def_1_id)
    self.assertEqual(outcome, {expected_key: expected_value})

  def testConvertApprovalDefNames_NoSuchProjectException(self):
    approval_ids = [self.approval_def_1_id]
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertApprovalDefNames(
          self.cnxn, approval_ids, self.dne_project_id, self.services)

  def testConvertProjectName(self):
    self.assertEqual(
        rnc.ConvertProjectName(
            self.cnxn, self.project_1.project_id, self.services),
        'projects/{}'.format(self.project_1.project_name))

  def testConvertProjectName_NoSuchProjectException(self):
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertProjectName(self.cnxn, self.dne_project_id, self.services)

  def testConvertProjectConfigName(self):
    self.assertEqual(
        rnc.ConvertProjectConfigName(
            self.cnxn, self.project_1.project_id, self.services),
        'projects/{}/config'.format(self.project_1.project_name))

  def testConvertProjectConfigName_NoSuchProjectException(self):
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertProjectConfigName(
          self.cnxn, self.dne_project_id, self.services)

  def testConvertProjectMemberName(self):
    self.assertEqual(
        rnc.ConvertProjectMemberName(
            self.cnxn, self.project_1.project_id, 111, self.services),
        'projects/{}/members/{}'.format(self.project_1.project_name, 111))

  def testConvertProjectMemberName_NoSuchProjectException(self):
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertProjectMemberName(
          self.cnxn, self.dne_project_id, 111, self.services)

  def testConvertProjectSavedQueryNames(self):
    query_ids = [self.psq_1.query_id, self.psq_2.query_id, self.dne_psq_id]
    outcome = rnc.ConvertProjectSavedQueryNames(
        self.cnxn, query_ids, self.project_1.project_id, self.services)

    expected_value_1 = 'projects/{}/savedQueries/{}'.format(
        self.project_1.project_name, self.psq_1.name)
    expected_value_2 = 'projects/{}/savedQueries/{}'.format(
        self.project_1.project_name, self.psq_2.name)
    self.assertEqual(
        outcome, {
            self.psq_1.query_id: expected_value_1,
            self.psq_2.query_id: expected_value_2
        })

  def testConvertProjectSavedQueryNames_NoSuchProjectException(self):
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.ConvertProjectSavedQueryNames(
          self.cnxn, [self.psq_1.query_id], self.dne_project_id, self.services)
