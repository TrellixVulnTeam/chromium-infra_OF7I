# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This file was ported from the catapult project, but has been edited down to
# the minimum requirements for the Pinpoint execution engine.

import datetime
import dataclasses
import re
import urllib
import typing

from chromeperf.pinpoint import errors
from chromeperf.pinpoint.models import commit
from chromeperf.pinpoint.models import repository as repository_module
from chromeperf.pinpoint import change_pb2
from chromeperf.services import gerrit_service


@dataclasses.dataclass
class GerritPatch:
    """A patch in Gerrit.

    change is a change ID of the format '<project>~<branch>~<Change-Id>' and
    revision is a commit ID. Both are described in the Gerrit API
    documentation.

    https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#ids

    They must be in a canonical format so we can look up builds precisely.
    """
    server: str
    change: str
    revision: str

    @classmethod
    def FromProto(cls, proto: change_pb2.GerritPatch):
        if not proto.server and not proto.change and not proto.revision:
            return None
        return cls(proto.server, proto.change, proto.revision)

    def to_proto(self) -> typing.Union[change_pb2.GerritPatch, None]:
        if not self.server and not self.change and not self.revision:
            return None
        return change_pb2.GerritPatch(server=self.server,
                                      change=self.change,
                                      revision=self.revision)

    def __str__(self):
        return self.revision[:7]

    @property
    def id_string(self):
        return '%s/%s/%s' % (self.server, self.change, self.revision)

    def BuildParameters(self):
        patch_info = gerrit_service.get_change(self.server,
                                               self.change,
                                               fields=('ALL_REVISIONS', ))
        revision_info = patch_info['revisions'][self.revision]
        return {
            'patch_gerrit_url': self.server,
            'patch_issue': patch_info['_number'],
            'patch_project': patch_info['project'],
            'patch_ref': revision_info['fetch']['http']['ref'],
            'patch_repository_url': revision_info['fetch']['http']['url'],
            'patch_set': revision_info['_number'],
            'patch_storage': 'gerrit',
        }

    @property
    def hostname(self):
        h = self.server.split('://')[-1]
        return h.split('/')[0]

    def BuildsetTags(self):
        patch_info = gerrit_service.get_change(self.server,
                                               self.change,
                                               fields=('ALL_REVISIONS', ))
        revision_info = patch_info['revisions'][self.revision]
        return 'buildset:patch/gerrit/%s/%s/%s' % (
            self.hostname, patch_info['_number'], revision_info['_number'])

    def AsDict(self):
        d = {
            'server': self.server,
            'change': self.change,
            'revision': self.revision,
        }

        patch_info = gerrit_service.get_change(self.server,
                                               self.change,
                                               fields=('ALL_REVISIONS',
                                                       'DETAILED_ACCOUNTS',
                                                       'COMMIT_FOOTERS'))
        revision_info = patch_info['revisions'][self.revision]
        url = '%s/c/%s/+/%d/%d' % (self.server, patch_info['project'],
                                   patch_info['_number'],
                                   revision_info['_number'])
        author = revision_info['uploader']['email']
        created = datetime.datetime.strptime(revision_info['created'],
                                             '%Y-%m-%d %H:%M:%S.%f000')
        subject = patch_info['subject']
        current_revision = patch_info['current_revision']
        message = patch_info['revisions'][current_revision][
            'commit_with_footers']

        d.update({
            'url': url,
            'author': author,
            'created': created.isoformat(),
            'subject': subject,
            'message': message,
        })

        return d

    @classmethod
    def FromData(cls, data):
        """Creates a new GerritPatch from the given request data.

        Args:
        data: A patch URL string, for example:
            https://chromium-review.googlesource.com/c/chromium/tools/build/+/679595
            Or a dict containing {server, change, revision [optional]}.
            change is a {change-id} as described in the Gerrit API
            documentation. revision is a commit ID hash or numeric patch
            number. If revision is omitted, it is the change's current
            revision.

        Returns:
        A GerritPatch.

        Raises:
        KeyError: The patch doesn't exist or doesn't have the given revision.
        ValueError: The URL has an unrecognized format.
        """
        if isinstance(data, dict):
            return cls.FromDict(data)
        else:
            return cls.FromUrl(data)

    @classmethod
    def FromUrl(cls, url):
        """Creates a new GerritPatch from the given URL.

        Args:
        url: A patch URL string, for example:
            https://chromium-review.googlesource.com/c/chromium/tools/build/+/679595

        Returns:
        A GerritPatch.

        Raises:
        KeyError: The patch doesn't have the given revision.
        ValueError: The URL has an unrecognized format.
        """
        url_parts = urllib.parse.urlparse(url)
        server = urllib.parse.urlunsplit(
            (url_parts.scheme, url_parts.netloc, '', '', ''))
        change_rev_match = re.match(r'^.*\/\+\/(\d+)(?:\/(\d+))?\/?$', url)
        change_match = re.match(r'^\/(\d+)\/?$', url_parts.path)
        redirector_match = re.match(r'^/c/(\d+)\/?$', url_parts.path)
        if change_rev_match:
            change = change_rev_match.group(1)
            revision = change_rev_match.group(2)
        elif change_match:
            # Support URLs returned by the 'git cl issue' command
            change = change_match.group(1)
            revision = None
        elif redirector_match:
            # Support non-fully-resolved URLs
            change = redirector_match.group(1)
            revision = None
        else:
            raise errors.BuildGerritURLInvalid(url)

        return cls.FromDict({
            'server': server,
            'change': int(change),
            'revision': int(revision) if revision else None,
        })

    @classmethod
    def FromDict(cls, data):
        """Creates a new GerritPatch from the given dict.

        Args:
            data: A dict containing {server, change, revision [optional]}.
                change is a {change-id} as described in the Gerrit API
                documentation. revision is a commit ID hash or numeric patch
                number. If revision is omitted, it is the change's current
                revision.

        Returns:
        A GerritPatch.

        Raises:
        KeyError: The patch doesn't have the given revision.
        """
        server = data['server']
        change = data['change']
        revision = data.get('revision')

        # Look up the patch and convert everything to a canonical format.
        try:
            patch_info = gerrit_service.get_change(
                server,
                change,
                fields=('ALL_REVISIONS', ),
            )
        except gerrit_service.NotFoundError as e:
            raise KeyError(str(e))
        change = patch_info['id']

        # Revision can be a revision ID or numeric patch number.
        if not revision:
            revision = patch_info['current_revision']
        for revision_id, revision_info in patch_info['revisions'].items():
            if revision == revision_id or revision == revision_info['_number']:
                revision = revision_id
                break
        else:
            raise KeyError('Patch revision not found: %s/%s revision %s' %
                           (server, change, revision))

        return cls(server, change, revision)


@dataclasses.dataclass
class Change:
    """A particular set of Commits with or without an additional patch applied.

    For example, a Change might sync to src@9064a40 and catapult@8f26966,
    then apply patch 2423293002.
    """
    commits: list = dataclasses.field(default_factory=list)
    patch: GerritPatch = None

    @classmethod
    def FromProto(cls, datastore_client, proto: change_pb2.Change):
        return cls(commits=[commit.Commit.FromProto(datastore_client, c)
                            for c in proto.commits],
                   patch=GerritPatch.FromProto(proto.patch))

    def __str__(self):
        """Returns an informal short string representation of this Change."""
        string = ' '.join(str(commit) for commit in self.commits)
        if self.patch:
            string += ' + ' + str(self.patch)
        return string

    @property
    def id_string(self):
        """Returns a string that is unique to this set of commits and patch.

    This method treats the commits as unordered. chromium@a v8@b is the same as
    v8@b chromium@a. This is useful for looking up a build with this Change.
    """
        string = ' '.join(commit.id_string for commit in sorted(self.commits))
        if self.patch:
            string += ' + ' + self.patch.id_string
        return string

    @property
    def base_commit(self):
        return self.commits[0]

    @property
    def last_commit(self):
        return self.commits[-1]

    @property
    def deps(self):
        return tuple(self.commits[1:])


def reconstitute_change(datastore_client, change):
    return Change(
        commits=[
            commit.Commit(
                repository=repository_module.Repository.FromName(
                    datastore_client,
                    c.get('repository'),
                ),
                git_hash=c.get('git_hash'),
            ) for c in change.get('commits')
        ],
        patch=change.get('patch'),
    )
