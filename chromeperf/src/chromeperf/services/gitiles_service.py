# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Functions for getting commit information from Gitiles."""

import base64

from chromeperf.services import gerrit_service
from chromeperf.services import request

NotFoundError = request.NotFoundError


def commit_info(repository_url, git_hash):
  """Fetches information about a commit.

  Args:
    repository_url: The url of the git repository.
    git_hash: The git hash of the commit.

  Returns:
    A dictionary containing the author, message, time, file changes, and other
    information. See gitiles_service_test.py for an example.

  Raises:
    NotFoundError: The repository or commit was not found in Gitiles.
    httplib.HTTPException: A network or HTTP error occurred.
  """
  # TODO: Update the docstrings in this file.
  url = '%s/+/%s?format=JSON' % (repository_url, git_hash)
  return request.request_json(
      url,
      use_cache=is_hash(git_hash),
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE)


def commit_range(repository_url, first_git_hash, last_git_hash):
  """Fetches the commits in between first and last, including the latter.

  Args:
    repository_url: The git url of the repository.
    first_git_hash: The git hash of the earliest commit in the range.
    last_git_hash: The git hash of the latest commit in the range.

  Returns:
    A list of dictionaries, one for each commit after the first commit up to
    and including the last commit. For each commit, its dictionary will
    contain information about the author and the committer and the commit itself.
    See gitiles_service_test.py for an example. The list is in order from newest
    to oldest.

  Raises:
    NotFoundError: The repository or a commit was not found in Gitiles.
    httplib.HTTPException: A network or HTTP error occurred.
  """
  commits = []
  while last_git_hash:
    url = '%s/+log/%s..%s?format=JSON' % (repository_url, first_git_hash,
                                          last_git_hash)
    use_cache = is_hash(first_git_hash) and is_hash(last_git_hash)
    response = request.request_json(
        url,
        use_cache=use_cache,
        use_auth=True,
        scope=gerrit_service.GERRIT_SCOPE)
    commits += response['log']
    last_git_hash = response.get('next')
  return commits


def file_contents(repository_url, git_hash, path):
  """Fetches the contents of a file at a particular commit.

  Args:
    repository_url: The git url of the repository.
    git_hash: The git hash of the commit, or "HEAD".
    path: The path in the repository to the file.

  Returns:
    A string containing the file contents.

  Raises:
    NotFoundError: The repository, commit, or file was not found in Gitiles.
    httplib.HTTPException: A network or HTTP error occurred.
  """
  url = '%s/+/%s/%s?format=TEXT' % (repository_url, git_hash, path)
  response = request.request(
      url,
      use_cache=is_hash(git_hash),
      use_auth=True,
      scope=gerrit_service.GERRIT_SCOPE)
  return base64.b64decode(response)


def is_hash(git_hash):
  """Returns True iff git_hash is a full SHA-1 hash.

  Commits keyed by full git hashes are guaranteed to not change. It's unsafe
  to cache things that can change (e.g. `HEAD`, `master`, tag names)
  """
  return git_hash.isalnum() and len(git_hash) == 40
