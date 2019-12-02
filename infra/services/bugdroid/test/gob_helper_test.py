# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

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
