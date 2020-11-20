# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import json
import mock
import unittest

import infra.services.bugdroid.gob_helper as gob_helper


class GobHelperTest(unittest.TestCase):

  def test_ParseAuthenticatedRepo(self):
    auth_res, unauth_res = gob_helper.ParseAuthenticatedRepo(
        'https://chromium.googlesource.com/a/chromium/src.git')
    self.assertEqual('/a/chromium/src.git', auth_res.path)
    self.assertEqual('/chromium/src.git', unauth_res.path)

  def test_GetRepoUrlFromFetchInfo(self):
    self.assertEqual(
        'url',
        gob_helper.GetRepoUrlFromFetchInfo({'http': {'url': 'url'}}))
    self.assertEqual(
        'https://foo.googlesource.com/bar',
        gob_helper.GetRepoUrlFromFetchInfo({'sso': {'url': 'sso://foo/bar'}}))
    self.assertIsNone(
        gob_helper.GetRepoUrlFromFetchInfo({'foo': {'url': 'url'}}))



class GitLogEntryTest(unittest.TestCase):
  def _make_entry(self, message):
    entry = gob_helper.GitLogEntry(
        'abcdef', ['123456'], 'Author', 'author@example.com', 'Committer',
        'committer@example.com', '2005-05-05 05:05:05.000000000',
        '2010-10-10 10:10:10.000000000', message,
        branch='refs/heads/branch',
        repo_url='https://example.googlesource.com/foo')
    entry.add_path('modify', 'modified/file', None)
    entry.add_path('add', 'added/file', None)
    entry.add_path('delete', 'gone', 'deleted/file')
    return entry

  def test_GetCommitUrl(self):
    entry = self._make_entry('Message')
    self.assertEqual(
        entry.GetCommitUrl(),
        'https://example.googlesource.com/foo/+/abcdef')

  def test_GetCommitUrl_no_repo_url_no_shorten(self):
    entry = self._make_entry('Message')
    entry.repo_url = None
    self.assertIsNone(entry.GetCommitUrl())

  def test_GetCommitUrl_no_repo_url_shorten(self):
    entry = self._make_entry('Message')
    entry.repo_url = None
    self.assertEqual(
        entry.GetCommitUrl(shorten=True), 'https://crrev.com/abcdef')

  def test_GetCommitUrl_parent(self):
    entry = self._make_entry('Message')
    self.assertEqual(
        entry.GetCommitUrl(parent=True),
        'https://example.googlesource.com/foo/+/123456')

  def test_GetCommitUrl_shorten(self):
    entry = self._make_entry('Message')
    self.assertEqual(
        entry.GetCommitUrl(shorten=True), 'https://crrev.com/abcdef')

  def test_GetPathUrl(self):
    entry = self._make_entry('Message')
    self.assertEqual(
        entry.GetPathUrl('path'),
        'https://example.googlesource.com/foo/+/abcdef/path')

  def test_GetPathUrl_no_repo_url(self):
    entry = self._make_entry('Message')
    entry.repo_url = None
    self.assertEqual(entry.GetPathUrl('path'), 'path')

  def test_GetPathUrl_no_repo_url_shorten(self):
    entry = self._make_entry('Message')
    entry.repo_url = None
    self.assertEqual(
        entry.GetPathUrl('path', shorten=True),
        'https://crrev.com/abcdef/path')

  def test_GetPathUrl_parent(self):
    entry = self._make_entry('Message')
    self.assertEqual(
        entry.GetPathUrl('path', parent=True),
        'https://example.googlesource.com/foo/+/123456/path')

  def test_GetPathUrl_shorten(self):
    entry = self._make_entry('Message')
    self.assertEqual(
        entry.GetPathUrl('path', shorten=True),
        'https://crrev.com/abcdef/path')


class GerritHelperTest(unittest.TestCase):

  def setUp(self):
    test_gerrit_response = r'''
  [
    {
      "id": "myProject~master~I3ea943139cb62e86071996f2480e58bf3eeb9dd2",
      "project": "myProject",
      "current_revision": "27cc4558b5a3d3387dd11ee2df7a117e7e581822",
      "revisions": {
        "27cc4558b5a3d3387dd11ee2df7a117e7e581822": {
          "kind": "REWORK",
          "_number": 2,
          "ref": "refs/changes/99/4799/2",
          "fetch": {
            "http": {
              "url": "http://gerrit:8080/myProject",
              "ref": "refs/changes/99/4799/2"
            }
          },
          "commit": {
            "parents": [
              {
                "commit": "b4003890dadd406d80222bf1ad8aca09a4876b70",
                "subject": "Implement Feature A"
              }
          ],
          "author": {
            "name": "John Doe",
            "email": "john.doe@example.com",
            "date": "2013-05-07 15:21:27.000000000",
            "tz": 120
          },
          "committer": {
            "name": "Gerrit Code Review",
            "email": "gerrit-server@example.com",
            "date": "2013-05-07 15:35:43.000000000",
            "tz": 120
          },
          "subject": "Implement Feature X",
          "message": "Implement Feature X\n\nAdded feature X."
          }
        }
      }
    }
  ]'''
    self.test_merged_change_return = json.loads(test_gerrit_response)
    self.test_api_url = "https://example-review.googlesource.com/a"

  # Autospec fails on @classmethod ParseChange for version 2.7.
  # https://github.com/testing-cabal/mock/issues/241
  @mock.patch.object(gob_helper.GerritHelper, 'ParseChange', return_value=None)
  @mock.patch.object(gob_helper.GerritHelper, 'GetMergedChanges', autospec=True)
  def test_ParseChange_success(self, mock_gmc, mock_pc):
    mock_gmc.return_value = self.test_merged_change_return
    test_helper = gob_helper.GerritHelper(self.test_api_url)
    test_helper.GetLogEntries()
    self.assertEqual(mock_pc.call_count, 1)

  @mock.patch.object(gob_helper.GerritHelper, 'ParseChange', return_value=None)
  @mock.patch.object(gob_helper.GerritHelper, 'GetMergedChanges', autospec=True)
  def test_ParseChange_skipped_with_ignored_projects(self, mock_gmc, mock_pc):
    mock_gmc.return_value = self.test_merged_change_return
    test_helper = gob_helper.GerritHelper(
        self.test_api_url, ignore_projects=['myProject'])
    test_helper.GetLogEntries()
    self.assertEqual(mock_pc.call_count, 0)

  @mock.patch.object(gob_helper.GerritHelper, 'ParseChange', return_value=None)
  @mock.patch.object(gob_helper.GerritHelper, 'GetMergedChanges', autospec=True)
  def test_ParseChange_called_in_test_mode(self, mock_gmc, mock_pc):
    mock_gmc.return_value = self.test_merged_change_return
    test_helper = gob_helper.GerritHelper(
        self.test_api_url, ignore_projects=['myProject'], test_mode=True)
    test_helper.GetLogEntries()
    self.assertEqual(mock_pc.call_count, 1)
