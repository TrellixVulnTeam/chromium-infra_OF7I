# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import datetime
import mock
import os
import unittest

from google.protobuf import field_mask_pb2
from google.protobuf import struct_pb2
from google.protobuf import text_format
from parameterized import parameterized

from go.chromium.org.luci.buildbucket.proto import build_pb2
from go.chromium.org.luci.buildbucket.proto import builder_common_pb2
from go.chromium.org.luci.buildbucket.proto import builds_service_pb2 as rpc_pb2
from go.chromium.org.luci.buildbucket.proto import common_pb2
from go.chromium.org.luci.buildbucket.proto import notification_pb2
from go.chromium.org.luci.buildbucket.proto import step_pb2

import validation

status_name = common_pb2.Status.Name

THIS_DIR = os.path.dirname(os.path.abspath(__file__))


class BaseTestCase(unittest.TestCase):
  func_name = None
  kwargs = None

  def validate(self, data):
    if self.kwargs:
      getattr(validation, self.func_name)(data, **self.kwargs)
    else:
      getattr(validation, self.func_name)(data)

  def assert_valid(self, data):
    self.validate(data)

  def assert_invalid(self, data, error_pattern):
    with self.assertRaisesRegexp(validation.Error, error_pattern):
      self.validate(data)


# Order of test classes must match the order of functions in validation.py

################################################################################
# Validation of common.proto messages.


class GerritChangeTests(BaseTestCase):
  func_name = 'validate_gerrit_change'

  def test_valid(self):
    msg = common_pb2.GerritChange(
        host='gerrit.example.com', change=1, patchset=1
    )
    self.assert_valid(msg)

  def test_no_host(self):
    msg = common_pb2.GerritChange(host='', change=1, patchset=1)
    self.assert_invalid(msg, r'host: required')


class GitilesCommitTests(BaseTestCase):
  func_name = 'validate_gitiles_commit'

  def test_valid(self):
    msg = common_pb2.GitilesCommit(
        host='gerrit.example.com',
        project='project',
        id='a' * 40,
        ref='refs/heads/master',
        position=1,
    )
    self.assert_valid(msg)

  def test_empty(self):
    msg = common_pb2.GitilesCommit()
    self.assert_invalid(msg, 'host: required')

  def test_host_and_project(self):
    msg = common_pb2.GitilesCommit(host='gerrit.example.com', project='project')
    self.assert_invalid(msg, 'ref: required')

  def test_invalid_ref(self):
    msg = common_pb2.GitilesCommit(
        host='gerrit.example.com',
        project='project',
        ref='master',
    )
    self.assert_invalid(msg, r'ref: must start with "refs/"')

  def test_invalid_id(self):
    msg = common_pb2.GitilesCommit(
        host='gerrit.example.com',
        project='project',
        ref='refs/heads/master',
        id='deadbeef',
    )
    self.assert_invalid(msg, r'id: does not match r"')


class TagsTests(BaseTestCase):

  def validate(self, data):
    validation.validate_tags(data, 'search')

  def test_valid(self):
    pairs = [common_pb2.StringPair(key='a', value='b')]
    self.assert_valid(pairs)

  def test_empty(self):
    pairs = []
    self.assert_valid(pairs)

  def test_key_has_colon(self):
    pairs = [common_pb2.StringPair(key='a:b', value='c')]
    self.assert_invalid(pairs, r'tag key "a:b" cannot have a colon')

  def test_no_key(self):
    pairs = [common_pb2.StringPair(key='', value='a')]
    self.assert_invalid(pairs, r'Invalid tag ":a": starts with ":"')


################################################################################
# Validation of build.proto messages.


class BuilderIDTests(BaseTestCase):
  func_name = 'validate_builder_id'

  def test_valid(self):
    msg = builder_common_pb2.BuilderID(
        project='chromium', bucket='try', builder='linux-rel'
    )
    self.assert_valid(msg)

  def test_no_project(self):
    msg = builder_common_pb2.BuilderID(
        project='', bucket='try', builder='linux-rel'
    )
    self.assert_invalid(msg, r'project: required')

  def test_invalid_project(self):
    msg = builder_common_pb2.BuilderID(
        project='Chromium', bucket='try', builder='linux-rel'
    )
    self.assert_invalid(msg, r'project: invalid')

  def test_invalid_bucket(self):
    msg = builder_common_pb2.BuilderID(
        project='chromium', bucket='a b', builder='linux-rel'
    )
    self.assert_invalid(
        msg, r'bucket: Bucket name "a b" does not match regular'
    )

  def test_v1_bucket(self):
    msg = builder_common_pb2.BuilderID(
        project='chromium', bucket='luci.chromium.ci', builder='linux-rel'
    )
    self.assert_invalid(
        msg,
        (
            r'bucket: invalid usage of v1 bucket format in v2 API; '
            'use u\'ci\' instead'
        ),
    )

  def test_invalid_builder(self):
    msg = builder_common_pb2.BuilderID(
        project='chromium', bucket='try', builder='#'
    )
    self.assert_invalid(msg, r'builder: invalid char\(s\)')


################################################################################
# Validation of rpc.proto messages.


class ScheduleBuildRequestTests(BaseTestCase):
  func_name = 'validate_schedule_build_request'

  def setUp(self):
    self.wke = set()
    self.kwargs = {'well_known_experiments': self.wke}

  def test_valid(self):
    self.wke.add('luci.use_realms')
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=dict(project='chromium', bucket='try', builder='linux-rel'),
        gitiles_commit=dict(
            host='gerrit.example.com',
            project='project',
            id='a' * 40,
            ref='refs/heads/master',
            position=1,
        ),
        gerrit_changes=[
            dict(host='gerrit.example.com', change=1, patchset=1),
            dict(host='gerrit.example.com', change=2, patchset=2),
        ],
        tags=[
            dict(key='a', value='a1'),
            dict(key='a', value='a2'),
            dict(key='b', value='b1'),
        ],
        dimensions=[
            dict(key='d1', value='dv1'),
            dict(key='d2', value='dv2'),
            dict(key='d2', value='dv3'),
        ],
        priority=100,
        experiments={
            "some.experiment": True,
            "luci.use_realms": False,
        },
        notify=dict(
            pubsub_topic='projects/project_id/topics/topic_id',
            user_data='blob',
        ),
    )
    msg.properties.update({'a': 1, '$recipe_engine/runtime': {'b': 1}})
    self.assert_valid(msg)

  def test_empty_property_value(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=dict(project='chromium', bucket='try', builder='linux-rel'),
        properties=dict(fields=dict(a=struct_pb2.Value())),
    )
    self.assert_invalid(
        msg, r'properties\.a: value is not set; for null, initialize null_value'
    )

  def test_no_builder_and_template_build_id(self):
    msg = rpc_pb2.ScheduleBuildRequest()
    self.assert_invalid(msg, 'builder or template_build_id is required')

  def test_no_builder_but_template_build_id(self):
    msg = rpc_pb2.ScheduleBuildRequest(template_build_id=1)
    self.assert_valid(msg)

  def test_gitiles_commit_incomplete(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        gitiles_commit=common_pb2.GitilesCommit(
            host='gerrit.example.com', project='project'
        ),
    )
    self.assert_invalid(msg, r'gitiles_commit\.ref: required')

  def test_gerrit_change(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        gerrit_changes=[
            common_pb2.GerritChange(host='gerrit.example.com', change=2),
        ],
    )
    self.assert_invalid(msg, r'gerrit_changes\[0\]\.patchset: required')

  def test_tags(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        tags=[dict()]
    )
    self.assert_invalid(msg, r'tags: Invalid tag ":": starts with ":"')

  def test_dimensions(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        dimensions=[dict()]
    )
    self.assert_invalid(msg, r'dimensions\[0\]\.key: required')

  def test_priority(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        priority=256,
    )
    self.assert_invalid(msg, r'priority: must be in \[0, 255\]')

  def test_experiments(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        experiments={"bad!": True},
    )
    self.assert_invalid(msg, r'does not match')

    msg.experiments.clear()
    msg.experiments['luci.use_ralms'] = True
    self.assert_invalid(msg, r'unknown experiment has reserved prefix')

  def test_notify_pubsub_topic(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        notify=notification_pb2.NotificationConfig(),
    )
    self.assert_invalid(msg, r'notify.pubsub_topic: required')

  def test_notify_user_data(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        notify=notification_pb2.NotificationConfig(
            pubsub_topic='x',
            user_data='a' * 5000,
        ),
    )
    self.assert_invalid(msg, r'notify.user_data: must be <= 4096 bytes')

  def test_cipd_version_latest(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        exe=dict(cipd_version='latest'),
    )
    self.assert_valid(msg)

  def test_cipd_version_invalid(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        exe=dict(cipd_version=':'),
    )
    self.assert_invalid(msg, r'exe.cipd_version: invalid version ":"')

  def test_cipd_package(self):
    msg = rpc_pb2.ScheduleBuildRequest(
        builder=builder_common_pb2.BuilderID(
            project='chromium', bucket='try', builder='linux-rel'
        ),
        exe=dict(cipd_package='something'),
    )
    self.assert_invalid(msg, r'exe.cipd_package: disallowed')


class RequestedDimensionTests(BaseTestCase):
  func_name = 'validate_requested_dimension'

  def test_valid(self):
    msg = common_pb2.RequestedDimension(key='a', value='b')
    self.assert_valid(msg)

  def test_valid_with_expiration(self):
    msg = common_pb2.RequestedDimension(
        key='a', value='b', expiration=dict(seconds=60)
    )
    self.assert_valid(msg)

  def test_valid_with_no_key(self):
    msg = common_pb2.RequestedDimension(key='', value='b')
    self.assert_invalid(msg, r'key: required')

  def test_valid_with_no_value(self):
    msg = common_pb2.RequestedDimension(key='a', value='')
    self.assert_invalid(msg, r'value: required')

  def test_valid_with_caches(self):
    msg = common_pb2.RequestedDimension(key='caches', value='b')
    self.assert_invalid(msg, r'key: "caches" is invalid')

  def test_valid_with_pool(self):
    msg = common_pb2.RequestedDimension(key='pool', value='b')
    self.assert_invalid(msg, r'key: "pool" is not allowed')

  def test_valid_with_negative_seconds(self):
    msg = common_pb2.RequestedDimension(
        key='a', value='b', expiration=dict(seconds=-1)
    )
    self.assert_invalid(msg, r'seconds: must not be negative')

  def test_valid_with_42_seconds(self):
    msg = common_pb2.RequestedDimension(
        key='a', value='b', expiration=dict(seconds=42)
    )
    self.assert_invalid(msg, r'seconds: must be a multiple of 60')

  def test_valid_with_nanos(self):
    msg = common_pb2.RequestedDimension(
        key='a', value='b', expiration=dict(nanos=1)
    )
    self.assert_invalid(msg, r'nanos: must be 0')
