# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import dataclasses
import pytest

from chromeperf.pinpoint.models import repository as repository_module


def test_raises_when_repo_unknown(datastore_client):
    with pytest.raises(KeyError):
        repository_module.repository_url(datastore_client, 'no-such-name')
    with pytest.raises(KeyError):
        repository_module.Repository.FromName(datastore_client, 'no-such-name')
    with pytest.raises(KeyError):
        repository_module.repository_name(datastore_client,
                                          'https://no-such-url/')
    with pytest.raises(KeyError):
        repository_module.Repository.FromUrl(datastore_client,
                                             'https://no-such-url/')

def test_resolves_known_names_and_url(datastore_client):
    NAME, URL = 'foo-name', 'https://host/foo'
    repository_module.add_repository(datastore_client, NAME, URL)

    assert repository_module.repository_url(datastore_client, NAME) == URL
    assert repository_module.repository_name(datastore_client, URL) == NAME
    repo_from_name = repository_module.Repository.FromName(datastore_client,
                                                           NAME)
    assert repo_from_name.name == NAME
    assert repo_from_name.url == URL
    repo_from_url = repository_module.Repository.FromUrl(datastore_client, URL)
    assert repo_from_url.name == NAME
    assert repo_from_url.url == URL

def test_resolves_names_to_canonical_url(datastore_client):
    NAME = 'foo-name'
    repository_module.add_repository(datastore_client, NAME, 'https://foo/1')
    repository_module.add_repository(datastore_client, NAME, 'https://foo/2')

    # The most recently added URL is the canonical URL, i.e. https://foo/2.
    assert (repository_module.repository_url(datastore_client, NAME)
            == 'https://foo/2')
    repo_from_name = repository_module.Repository.FromName(datastore_client,
                                                           NAME)
    assert repo_from_name.name == NAME
    assert repo_from_name.url == 'https://foo/2'
    for url in ['https://foo/1', 'https://foo/2']:
        repo_from_url = repository_module.Repository.FromUrl(datastore_client,
                                                             url)
        assert repo_from_url.name == NAME
        # Regardless of which URL we looked this up with, we get the canonical
        # URL back.
        assert repo_from_url.url == 'https://foo/2'

def test_resolves_ignoring_dotgit_suffix(datastore_client):
    NAME, URL = 'foo-name', 'https://host/foo'
    repository_module.add_repository(datastore_client, NAME, URL)

    SUFFIXED_URL = 'https://host/foo.git'
    assert (repository_module.repository_name(datastore_client, SUFFIXED_URL)
            == NAME)
    repo_from_url = repository_module.Repository.FromUrl(datastore_client,
                                                         SUFFIXED_URL)
    assert repo_from_url.name == NAME
    assert repo_from_url.url == URL


def test_Repository_not_directly_constructible():
    with pytest.raises(AssertionError):
        repository_module.Repository('name', 'https://url/')

