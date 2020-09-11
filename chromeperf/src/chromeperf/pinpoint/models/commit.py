# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This file was ported from the catapult project, but has been edited down to
# the minimum requirements for the Pinpoint execution engine.

import dataclasses
import datetime
import re

from google.cloud import datastore

from chromeperf.services import gitiles_service
from chromeperf.pinpoint.models import repository as repository_module
from chromeperf.pinpoint import change_pb2


@dataclasses.dataclass
class Dep:
    repository_url: str
    git_hash: str


@dataclasses.dataclass
class Commit:
    repository: str
    git_hash: str

    @classmethod
    def FromProto(cls, proto: change_pb2.Commit):
        return cls(proto.repository, proto.git_hash)

    def __str__(self):
        """Returns an informal short string representation of this Commit."""
        return self.repository + '@' + self.git_hash[:7]

    @property
    def id_string(self):
        """Returns a string that is unique to this repository and git hash."""
        return self.repository + '@' + self.git_hash

    def repository_url(self, datastore_client):
        """The HTTPS URL of the repository as passed to `git clone`."""
        cached_url = getattr(self, '_repository_url', None)
        if not cached_url:
            self._repository_url = repository_module.repository_url(
                datastore_client, self.repository)
        return self._repository_url

    def AsDict(self, datastore_client):
        d = {
            'repository': self.repository,
            'git_hash': self.git_hash,
        }

        repo_url = self.repository_url(datastore_client)
        commit_info = gitiles_service.commit_info(repo_url, self.git_hash)
        d.update(commit_info)
        d['created'] = datetime.datetime.strptime(
            commit_info['committer']['time'],
            '%a %b %d %H:%M:%S %Y %z').isoformat()

        commit_position = _parse_commit_position(d['message'])
        if commit_position:
            d['commit_position'] = commit_position

        review_url = _parse_commit_field('Reviewed-on: ', d['message'])
        if review_url:
            d['review_url'] = review_url

        change_id = _parse_commit_field('Change-Id: ', d['message'])
        if change_id:
            d['change_id'] = change_id

        url = repo_url + '/+/' + commit_info['commit']
        d['url'] = url
        return d


def _parse_commit_position(commit_message):
    """Parses a commit message for the commit position.

    Args:
        commit_message: The commit message as a string.

    Returns:
        An int if there is a commit position, or None otherwise."""
    match = re.search(
        '^Cr-Commit-Position: [a-z/]+@{#([0-9]+)}$',
        commit_message,
        re.MULTILINE,
    )
    if match:
        return int(match.group(1))
    return None


def _parse_commit_field(field, commit_message):
    for l in commit_message.splitlines():
        match = l.split(field)
        if len(match) == 2:
            return match[1]
    return None
