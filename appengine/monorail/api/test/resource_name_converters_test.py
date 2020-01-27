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
        project=fake.ProjectService())
    self.cnxn = fake.MonorailConnection()
    self.project_1 = self.services.project.TestAddProject(
        'proj', project_id=789)
    self.project_2 = self.services.project.TestAddProject(
        'goose', project_id=788)

    self.issue_1 = fake.MakeTestIssue(
        self.project_1.project_id, 1, 'sum', 'New', 111,
        project_name=self.project_1.project_name)
    self.issue_2 = fake.MakeTestIssue(
        self.project_2.project_id, 2, 'sum', 'New', 111,
        project_name=self.project_2.project_name)
    self.services.issue.TestAddIssue(self.issue_1)
    self.services.issue.TestAddIssue(self.issue_2)

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
        'hotlists/78909/items/%s.%d' % (
            self.project_1.project_name, self.issue_1.local_id),
        'hotlists/78909/items/%s.%d' % (
            self.project_2.project_name, self.issue_2.local_id)]
    self.assertEqual(
        rnc.IngestHotlistItemNames(self.cnxn, names, self.services),
        [self.issue_1.issue_id, self.issue_2.issue_id])

  def testIngestHotlistItemNames_ProjectNotFound(self):
    """Exception is raised if a project is not found."""
    names = [
        'hotlists/78909/items/%s.%d' % (
            self.project_1.project_name, self.issue_1.local_id),
        'hotlists/78909/items/chicken.2']
    with self.assertRaises(exceptions.NoSuchProjectException):
      rnc.IngestHotlistItemNames(self.cnxn, names, self.services)

  def testIngestHotlistItems_IssueNotFound(self):
    """Exception is raised if an Issue is not found."""
    names = [
        'hotlists/78909/items/%s.%d' % (
            self.project_1.project_name, self.issue_1.local_id),
        'hotlists/78909/items/%s.5' % (self.project_2.project_name)]
    with self.assertRaises(exceptions.NoSuchIssueException):
      rnc.IngestHotlistItemNames(self.cnxn, names, self.services)
