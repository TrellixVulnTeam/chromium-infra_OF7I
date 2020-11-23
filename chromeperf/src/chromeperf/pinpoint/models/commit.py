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


## @dataclasses.dataclass
## class Dep:
##     repository_url: str
##     git_hash: str


@dataclasses.dataclass(frozen=True)
class Commit:
    repository: repository_module.Repository
    git_hash: str

    @classmethod
    def FromProto(cls, datastore_client, proto: change_pb2.Commit):
        repo = repository_module.Repository.FromName(datastore_client,
                                                     proto.repository)
        return cls(repo, proto.git_hash)

    def to_proto(self) -> change_pb2.Commit:
        return change_pb2.Commit(repository=self.repository.name,
                                 git_hash=self.git_hash)

    def __str__(self):
        """Returns an informal short string representation of this Commit."""
        return self.repository.name + '@' + self.git_hash[:7]

    @property
    def id_string(self):
        """Returns a string that is unique to this repository and git hash."""
        return self.repository.name + '@' + self.git_hash

    def repository_url(self, datastore_client):
        """The HTTPS URL of the repository as passed to `git clone`."""
        del datastore_client
        return self.repository.url

    def AsDict(self, datastore_client):
        d = {
            'repository': self.repository.name,
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

    @classmethod
    def MakeValidated(cls, datastore_client,
                      repository: repository_module.Repository, git_hash: str
                      ) -> 'Commit':
        # TODO: use a commit cache?
        try:
            # Use gitiles to resolve aliases like HEAD to a real hash.
            result = gitiles_service.commit_info(repository.url, git_hash)
            return cls(repository=repository, git_hash=result['commit'])
        except gitiles_service.NotFoundError as e:
            raise KeyError(str(e))


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


def commit_range(commit_a: Commit, commit_b: Commit):
    """Get commit info dicts from gitiles for a commit range."""
    # We need to get the full list of commits in between two git hashes, and
    # only look into the chain formed by following the first parents of each
    # commit. This gives us a linear view of the log even in the presence of
    # merge commits.
    commits = []

    # The commit_range by default is in reverse-chronological (latest commit
    # first) order. This means we should keep following the first parent to get
    # the linear history for a branch that we're exploring.
    expected_parent = commit_b.git_hash
    commit_range = gitiles_service.commit_range(
            commit_a.repository_url, commit_a.git_hash, commit_b.git_hash)
    for commit in commit_range:
        # Skip commits until we find the parent we're looking for.
        if commit['commit'] == expected_parent:
            commits.append(commit)
            if 'parents' in commit and len(commit['parents']):
                expected_parent = commit['parents'][0]

    return commits
