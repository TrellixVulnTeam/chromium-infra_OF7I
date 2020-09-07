# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest

from chromeperf.services import gerrit_service
from chromeperf.services import gitiles_service


def test_Gitiles_CommitInfo(request_json, service_request):
  return_value = {
      'commit':
          'commit_hash',
      'tree':
          'tree_hash',
      'parents': ['parent_hash'],
      'author': {
          'name': 'username',
          'email': 'email@chromium.org',
          'time': 'Fri Jan 01 00:00:00 2016',
      },
      'committer': {
          'name': 'Commit bot',
          'email': 'commit-bot@chromium.org',
          'time': 'Fri Jan 01 00:01:00 2016',
      },
      'message':
          'Subject.\n\nCommit message.',
      'tree_diff': [{
          'type': 'modify',
          'old_id': 'old_hash',
          'old_mode': 33188,
          'old_path': 'a/b/c.py',
          'new_id': 'new_hash',
          'new_mode': 33188,
          'new_path': 'a/b/c.py',
      },],
  }
  request_json.return_value = return_value
  assert gitiles_service.commit_info('https://chromium.googlesource.com/repo',
                                     'commit_hash') == return_value
  request_json.assert_called_once_with(
      'https://chromium.googlesource.com/repo/+/commit_hash?format=JSON',
      use_cache=False,
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE)


def test_Gitiles_CommitRange(request_json, service_request):
  return_value = {
      'log': [
          {
              'commit': 'commit_2_hash',
              'tree': 'tree_2_hash',
              'parents': ['parent_2_hash'],
              'author': {
                  'name': 'username',
                  'email': 'email@chromium.org',
                  'time': 'Sat Jan 02 00:00:00 2016',
              },
              'committer': {
                  'name': 'Commit bot',
                  'email': 'commit-bot@chromium.org',
                  'time': 'Sat Jan 02 00:01:00 2016',
              },
              'message': 'Subject.\n\nCommit message.',
          },
          {
              'commit': 'commit_1_hash',
              'tree': 'tree_1_hash',
              'parents': ['parent_1_hash'],
              'author': {
                  'name': 'username',
                  'email': 'email@chromium.org',
                  'time': 'Fri Jan 01 00:00:00 2016',
              },
              'committer': {
                  'name': 'Commit bot',
                  'email': 'commit-bot@chromium.org',
                  'time': 'Fri Jan 01 00:01:00 2016',
              },
              'message': 'Subject.\n\nCommit message.',
          },
      ],
  }
  request_json.return_value = return_value
  assert gitiles_service.commit_range('https://chromium.googlesource.com/repo',
                                      'commit_0_hash',
                                      'commit_2_hash') == return_value['log']
  request_json.assert_called_once_with(
      'https://chromium.googlesource.com/repo/+log/'
      'commit_0_hash..commit_2_hash?format=JSON',
      use_cache=False,
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE)


def test_Gitiles_CommitRangePaginated(request_json, service_request):
  return_value_1 = {
      'log': [
          {
              'commit': 'commit_4_hash'
          },
          {
              'commit': 'commit_3_hash'
          },
      ],
      'next': 'commit_2_hash',
  }
  return_value_2 = {
      'log': [
          {
              'commit': 'commit_2_hash'
          },
          {
              'commit': 'commit_1_hash'
          },
      ],
  }

  request_json.side_effect = return_value_1, return_value_2
  assert gitiles_service.commit_range(
      'https://chromium.googlesource.com/repo', 'commit_0_hash',
      'commit_4_hash') == return_value_1['log'] + return_value_2['log']


def test_Gitiles_FileContents(service_request):
  service_request.return_value = 'aGVsbG8='
  assert gitiles_service.file_contents('https://chromium.googlesource.com/repo',
                                       'commit_hash', 'path') == b'hello'
  service_request.assert_called_once_with(
      'https://chromium.googlesource.com/repo/+/commit_hash/path?format=TEXT',
      use_cache=False,
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE)


def test_Gitiles_Cache(request_json, service_request):
  request_json.return_value = {'log': []}
  service_request.return_value = 'aGVsbG8='

  repository = 'https://chromium.googlesource.com/repo'
  git_hash = '3a44bc56c4efa42a900a1c22b001559b81e457e9'

  gitiles_service.commit_info(repository, git_hash)
  request_json.assert_called_with(
      '%s/+/%s?format=JSON' % (repository, git_hash),
      use_cache=True,
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE)

  gitiles_service.commit_range(repository, git_hash, git_hash)
  request_json.assert_called_with(
      '%s/+log/%s..%s?format=JSON' % (repository, git_hash, git_hash),
      use_cache=True,
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE)

  gitiles_service.file_contents(repository, git_hash, 'path')
  service_request.assert_called_with(
      '%s/+/%s/path?format=TEXT' % (repository, git_hash),
      use_cache=True,
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE)
