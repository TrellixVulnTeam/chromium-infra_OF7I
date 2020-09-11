# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import datetime

from google.cloud import datastore


def repository_url(datastore_client, name):
    """Returns the URL of a repository, given its short name.

    If a repository moved locations or has multiple locations, a repository can
    have multiple URLs. The returned URL should be the current canonical one.

    Args:
      datastore_client: The client to use for the datastore query.
      name: The short name of the repository.

    Returns:
      A URL string, not including '.git'.
    """
    repositories = datastore_client.query(kind='Repository',
                                          order=('-time_added', ),
                                          filters=(('name', '=',
                                                    name), )).fetch()
    for repo in repositories:
        return repo['url']
    raise KeyError('Unknown repository name: ' + name)


def repository_name(datastore_client, url, add_if_missing=False):
    """Returns the short repository name, given its URL.

    By default, the short repository name is the last part of the URL.
    E.g. "https://chromium.googlesource.com/v8/v8": "v8"
    In some cases this is ambiguous, so the names can be manually adjusted.
    E.g. "../chromium/src": "chromium" and "../breakpad/breakpad/src":
    "breakpad"

    If a repository moved locations or has multiple locations, multiple URLs
    can map to the same name. This should only be done if they are exact
    mirrors and have the same git hashes.

    "https://webrtc.googlesource.com/src": "webrtc"
    "https://webrtc.googlesource.com/src/webrtc": "old_webrtc"
    "https://chromium.googlesource.com/external/webrtc/trunk/webrtc":
        "old_webrtc"

    Internally, all repositories are stored by short name, which always maps
    to the current canonical URL, so old URLs are automatically
    "upconverted".

    Args:
      url: The repository URL.
      add_if_missing: If True, also attempts to add the URL to the database with
        the default name mapping. Throws an exception if there's a name collision.

    Returns:
      The short name as a string.

    Raises:
      AssertionError: add_if_missing is True and there's a name collision.
    """
    if url.endswith('.git'):
        url = url[:-4]
    repo = datastore_client.get(datastore_client.key('Repository', url))
    if repo:
        return repo['name']
    if add_if_missing:
        name = url.split('/')[-1]
        return add_repository(datastore_client, name, url)

    raise KeyError('Unknown repository URL: %s' % url)


def add_repository(datastore_client, name, url):
    """Add a repository URL to the database with the default name mapping.

  The default short repository name is the last part of the URL.

  Returns:
    The short repository name.

  Raises:
    AssertionError: The default name is already in the database.
  """

    key = datastore_client.key('Repository', url)
    if datastore_client.get(key):
        raise AssertionError(f'Attempted to add a repository that\'s'
                             f'already in the Datastore: {name}: {url}')

    repo = datastore.Entity(key=key)
    repo.update({
        'name': name,
        'time_added': datetime.datetime.now(datetime.timezone.utc),
        'url': url,
    })
    datastore_client.put(repo)
    return name
