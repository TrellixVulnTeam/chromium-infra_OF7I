# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Model for storing information to look up isolates.

An isolate is a way to describe the dependencies of a specific build.

More about isolates:
https://github.com/luci/luci-py/blob/master/appengine/isolate/doc/client/Design.md
"""
import dataclasses
import datetime

from google.cloud import datastore

# Isolates expire in isolate server after 60 days. We expire
# our isolate lookups a little bit sooner, just to be safe.
ISOLATE_EXPIRY_DURATION = datetime.timedelta(days=58)


@dataclasses.dataclass
class Isolate:
    isolate_server: str
    isolate_hash: str
    key: datastore.Key
    created: datetime.datetime = dataclasses.field(
        default_factory=datetime.datetime.utcnow)


def get(builder_name, change, target, datastore_client):
    """Retrieve an isolate hash from the Datastore.

  Args:
    builder_name: The name of the builder that produced the isolate.
    change: The Change the isolate was built at.
    target: The compile target the isolate is for.

  Returns:
    A tuple containing the isolate server and isolate hash as strings.
  """
    entity = datastore_client.get(
        datastore_client.key('Isolate',
                             _isolate_key(builder_name, change, target)))
    if not entity:
        raise KeyError(
            'No isolate with builder %s, change %s, and target %s.' %
            (builder_name, change, target))

    if entity['created'] + ISOLATE_EXPIRY_DURATION < datetime.datetime.now(
            datetime.timezone.utc):
        raise KeyError('Isolate with builder %s, change %s, and target %s was '
                       'found, but is expired.' %
                       (builder_name, change, target))

    return entity['isolate_server'], entity['isolate_hash']


def put(isolate_infos, datastore_client):
    """Add isolate hashes to the Datastore.

  This function takes multiple entries to do a batched Datstore put.

  Args:
    isolate_infos: An iterable of tuples. Each tuple is of the form
        (builder_name, change, target, isolate_server, isolate_hash).
  """
    def _encode_isolate_entity(isolate):
        e = datastore.Entity(isolate.key)
        e.update(dataclasses.asdict(isolate))
        del e['key']
        return e

    datastore_client.put_multi([
        _encode_isolate_entity(
            Isolate(isolate_server=server,
                    isolate_hash=hash_,
                    key=datastore_client.key(
                        'Isolate', _isolate_key(builder_name, change,
                                                target))))
        for builder_name, change, target, server, hash_ in isolate_infos
    ])


def _isolate_key(builder_name, change, target):
    # The key must be stable across machines, platforms,
    # Python versions, and Python invocations.
    return '\n'.join((builder_name, change.id_string, target))
