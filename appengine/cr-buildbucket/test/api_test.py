# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import os
import datetime
import logging

from google.appengine.ext import ndb
from google.protobuf import text_format, field_mask_pb2

from components import auth
from components import prpc
from components import protoutil
from components.prpc import context as prpc_context
from testing_utils import testing
import mock

from go.chromium.org.luci.buildbucket.proto import build_pb2
from go.chromium.org.luci.buildbucket.proto import builder_pb2
from go.chromium.org.luci.buildbucket.proto import builds_service_pb2 as rpc_pb2
from go.chromium.org.luci.buildbucket.proto import common_pb2
from go.chromium.org.luci.buildbucket.proto import notification_pb2
from test import test_util
import api
import bbutil
import creation
import errors
import experiments
import model
import search
import user
import validation

future = test_util.future

THIS_DIR = os.path.dirname(os.path.abspath(__file__))


class BaseTestCase(testing.AppengineTestCase):
  """Base class for api.py tests."""

  def setUp(self):
    super(BaseTestCase, self).setUp()

    self.perms = test_util.mock_permissions(self)
    self.perms['chromium/try'] = [
        user.PERM_BUILDS_GET,
        user.PERM_BUILDS_LIST,
        user.PERM_BUILDS_ADD,
        user.PERM_BUILDS_CANCEL,
        user.PERM_BUILDERS_LIST,
    ]

    self.now = datetime.datetime(2015, 1, 1)
    self.patch('components.utils.utcnow', side_effect=lambda: self.now)

    self.api = api.BuildsApi()

  def call(
      self,
      method,
      req,
      ctx=None,
      expected_code=prpc.StatusCode.OK,
      expected_details=None
  ):
    ctx = ctx or prpc_context.ServicerContext()
    res = method(req, ctx)
    self.assertEqual(ctx.code, expected_code)
    if expected_details is not None:
      self.assertEqual(ctx.details, expected_details)
    if expected_code != prpc.StatusCode.OK:
      self.assertIsNone(res)
    return res


class RpcImplTests(BaseTestCase):

  def error_handling_test(self, ex, expected_code, expected_details):

    @api.rpc_impl_async('GetBuild')
    @ndb.tasklet
    def get_build_async(_req, _res, _ctx, _mask):
      raise ex

    ctx = prpc_context.ServicerContext()
    req = rpc_pb2.GetBuildRequest(id=1)
    res = build_pb2.Build()
    # pylint: disable=no-value-for-parameter
    get_build_async(req, res, ctx).get_result()
    self.assertEqual(ctx.code, expected_code)
    self.assertEqual(ctx.details, expected_details)

  def test_authorization_error_handling(self):
    self.error_handling_test(
        auth.AuthorizationError(), prpc.StatusCode.NOT_FOUND, 'not found'
    )

  def test_status_code_error_handling(self):
    self.error_handling_test(
        api.invalid_argument('bad'), prpc.StatusCode.INVALID_ARGUMENT, 'bad'
    )

  def test_invalid_field_mask(self):
    req = rpc_pb2.GetBuildRequest(fields=dict(paths=['invalid']))
    self.call(
        self.api.GetBuild,
        req,
        expected_code=prpc.StatusCode.INVALID_ARGUMENT,
        expected_details=(
            'invalid fields: invalid path "invalid": '
            'field "invalid" does not exist in message '
            'buildbucket.v2.Build'
        )
    )

  @mock.patch('creation.add_async', autospec=True)
  def test_trimming_exclude(self, add_async):
    add_async.return_value = future(
        test_util.build(
            id=54,
            builder=dict(project='chromium', bucket='try', builder='linux'),
            input=dict(properties=bbutil.dict_to_struct({'a': 'b'}))
        ),
    )
    req = rpc_pb2.ScheduleBuildRequest(
        builder=dict(project='chromium', bucket='try', builder='linux'),
    )
    res = self.call(self.api.ScheduleBuild, req)
    self.assertFalse(res.input.HasField('properties'))

  @mock.patch('creation.add_async', autospec=True)
  def test_trimming_include(self, add_async):
    add_async.return_value = future(
        test_util.build(
            id=54,
            builder=dict(project='chromium', bucket='try', builder='linux'),
            input=dict(properties=bbutil.dict_to_struct({'a': 'b'}))
        ),
    )
    req = rpc_pb2.ScheduleBuildRequest(
        builder=dict(project='chromium', bucket='try', builder='linux'),
        fields=dict(paths=['id'])
    )
    res = self.call(self.api.ScheduleBuild, req)
    self.assertEqual(res.id, 54)


class ScheduleBuildTests(BaseTestCase):

  @mock.patch('creation.add_async', autospec=True)
  def test_schedule(self, add_async):
    add_async.return_value = future(
        test_util.build(
            id=54,
            builder=dict(project='chromium', bucket='try', builder='linux'),
        ),
    )
    req = rpc_pb2.ScheduleBuildRequest(
        builder=dict(project='chromium', bucket='try', builder='linux'),
    )
    res = self.call(self.api.ScheduleBuild, req)
    self.assertEqual(res.id, 54)
    add_async.assert_called_once_with(
        creation.BuildRequest(schedule_build_request=req)
    )

  @mock.patch('creation.add_async', autospec=True)
  @mock.patch('service.get_async', autospec=True)
  def test_schedule_with_template_build_id(self, get_async, add_async):
    get_async.return_value = future(
        test_util.build(
            id=44,
            builder=dict(project='chromium', bucket='try', builder='linux'),
            canary=common_pb2.YES,
            input=build_pb2.Build.Input(
                experimental=common_pb2.NO,
                experiments=[
                    "experiment.foo",
                    "experiment.bar",
                    experiments.CANARY,
                ],
                properties=test_util.create_struct({
                    'property_key': 'property_value_from_build',
                    'another_property_key': 'another_property_value',
                }),
                gitiles_commit=common_pb2.GitilesCommit(
                    host='host', project='proj', ref="refs/from_host"
                ),
                gerrit_changes=[
                    common_pb2.GerritChange(
                        project='proj', host='host', change=1, patchset=1
                    ),
                    common_pb2.GerritChange(
                        project='proj', host='host', change=1, patchset=1
                    ),
                ],
            ),
            tags=[
                common_pb2.StringPair(
                    key='tag_key', value='tag_value_from_build'
                ),
                common_pb2.StringPair(
                    key='another_tag_key', value='another_tag_value'
                ),
            ],
            critical=common_pb2.YES,
            exe=common_pb2.Executable(cipd_package='package_from_host'),
            infra=build_pb2.BuildInfra(
                swarming=build_pb2.BuildInfra.Swarming(
                    parent_run_id='id_from_build'
                )
            ),
        ),
    )
    add_async.return_value = future(
        test_util.build(
            id=54,
            builder=dict(project='chromium', bucket='try', builder='linux'),
            canary=common_pb2.NO,
            input=build_pb2.Build.Input(
                experimental=common_pb2.YES,
                experiments=[
                    "experiment.bar",
                    "experiment.baz",
                    experiments.NON_PROD,
                ],
                properties=test_util.create_struct({
                    'property_key': 'property_value_from_req',
                }),
                gitiles_commit=common_pb2.GitilesCommit(
                    host='host', project='proj', ref="refs/from_req"
                ),
                gerrit_changes=[
                    common_pb2.GerritChange(
                        project='proj', host='host', change=2, patchset=2
                    ),
                ],
            ),
            tags=[
                common_pb2.StringPair(
                    key='tag_key', value='tag_value_from_req'
                ),
            ],
            critical=common_pb2.NO,
            exe=common_pb2.Executable(cipd_package=''),
            infra=build_pb2.BuildInfra(
                swarming=build_pb2.BuildInfra.Swarming(
                    parent_run_id='id_from_req'
                )
            ),
        ),
    )
    req = rpc_pb2.ScheduleBuildRequest(
        template_build_id=44,
        builder=dict(project='chromium', bucket='try', builder='linux'),
        canary=common_pb2.NO,
        experimental=common_pb2.YES,
        experiments={
            "experiment.bar": True,
            "experiment.baz": True,
            "experiment.foo": False,
            experiments.CANARY: True,
            experiments.NON_PROD: True,
        },
        properties=test_util.create_struct({
            'property_key': 'property_value_from_req',
        }),
        gitiles_commit=common_pb2.GitilesCommit(
            host='host', project='proj', ref="refs/from_req"
        ),
        gerrit_changes=[
            common_pb2.GerritChange(
                project='proj', host='host', change=2, patchset=2
            ),
        ],
        tags=[
            common_pb2.StringPair(key='tag_key', value='tag_value_from_req'),
        ],
        critical=common_pb2.NO,
        exe=common_pb2.Executable(cipd_package=''),
        swarming=rpc_pb2.ScheduleBuildRequest.Swarming(
            parent_run_id='id_from_req'
        ),
        notify=notification_pb2.NotificationConfig(pubsub_topic='topic'),
        fields=field_mask_pb2.FieldMask(),
    )
    res = self.call(self.api.ScheduleBuild, req)
    self.assertEqual(res.id, 54)

    add_async.assert_called_once_with(mock.ANY)
    actual_req = add_async.mock_calls[0][1][0].schedule_build_request
    self.assertEqual(actual_req, req)

  @mock.patch('creation.add_async', autospec=True)
  @mock.patch('service.get_async', autospec=True)
  def test_schedule_with_only_template_build_id(self, get_async, add_async):
    build_tags = [
        common_pb2.StringPair(key='tag_key', value='tag_value'),
        common_pb2.StringPair(key='another_tag_key', value='another_tag_value'),
    ]
    template_build = test_util.build(
        id=44,
        builder=dict(project='chromium', bucket='try', builder='linux'),
        canary=common_pb2.YES,
        input=build_pb2.Build.Input(
            experimental=common_pb2.NO,
            properties=test_util.create_struct({
                'property_key': 'property_value',
                'another_property_key': 'another_property_value',
            }),
            gitiles_commit=common_pb2.GitilesCommit(
                host='host', project='proj', ref="refs/ref"
            ),
            gerrit_changes=[
                common_pb2.GerritChange(
                    project='proj', host='host', change=1, patchset=1
                ),
                common_pb2.GerritChange(
                    project='proj', host='host', change=1, patchset=1
                ),
            ],
        ),
        tags=build_tags,
        critical=common_pb2.YES,
        exe=common_pb2.Executable(cipd_package='package'),
        infra=build_pb2.BuildInfra(
            swarming=build_pb2.BuildInfra.Swarming(parent_run_id='id')
        ),
    )
    get_async.return_value = future(template_build)
    add_async.return_value = future(
        test_util.build(
            id=54,
            builder=dict(project='chromium', bucket='try', builder='linux'),
            canary=common_pb2.NO,
            input=build_pb2.Build.Input(
                experimental=common_pb2.YES,
                properties=test_util.create_struct({
                    'property_key': 'property_value',
                }),
                gitiles_commit=common_pb2.GitilesCommit(
                    host='host', project='proj', ref="refs/ref"
                ),
                gerrit_changes=[
                    common_pb2.GerritChange(
                        project='proj', host='host', change=2, patchset=2
                    ),
                ],
            ),
            tags=build_tags,
            critical=common_pb2.NO,
            exe=common_pb2.Executable(cipd_package=''),
            infra=build_pb2.BuildInfra(
                swarming=build_pb2.BuildInfra.Swarming(parent_run_id='id')
            ),
        ),
    )
    req = rpc_pb2.ScheduleBuildRequest(template_build_id=44)
    res = self.call(self.api.ScheduleBuild, req)
    self.assertEqual(res.id, 54)

    add_async.assert_called_once_with(mock.ANY)
    actual_req = add_async.mock_calls[0][1][0].schedule_build_request
    self.assertEqual(actual_req.builder, template_build.proto.builder)

    self.assertEqual(actual_req.canary, common_pb2.UNSET)
    self.assertEqual(actual_req.experimental, common_pb2.UNSET)
    self.assertEqual(
        actual_req.experiments, {
            experiments.CANARY: True,
            experiments.NON_PROD: True,
        }
    )

    self.assertEqual(
        actual_req.properties, template_build.proto.input.properties
    )
    self.assertEqual(
        actual_req.gitiles_commit, template_build.proto.input.gitiles_commit
    )
    self.assertEqual(
        actual_req.gerrit_changes, template_build.proto.input.gerrit_changes
    )
    self.assertTrue(all(tag in actual_req.tags for tag in build_tags))
    self.assertEqual(actual_req.critical, template_build.proto.critical)
    self.assertEqual(actual_req.exe, template_build.proto.exe)
    self.assertEqual(actual_req.swarming.parent_run_id, '')

  @mock.patch('creation.add_async', autospec=True)
  @mock.patch('service.get_async', autospec=True)
  def test_schedule_build_doesnt_inject_empty_structs(
      self, get_async, add_async
  ):
    template_build = test_util.build(
        id=44,
        builder=dict(project='chromium', bucket='try', builder='linux'),
        canary=common_pb2.YES,
        input=build_pb2.Build.Input(),
        critical=common_pb2.YES,
        infra=build_pb2.BuildInfra(),
    )
    template_build.proto.ClearField('exe')
    get_async.return_value = future(template_build)
    add_async.return_value = future(
        test_util.build(
            id=54,
            builder=dict(project='chromium', bucket='try', builder='linux'),
            canary=common_pb2.NO,
            input=build_pb2.Build.Input(),
            critical=common_pb2.NO,
            infra=build_pb2.BuildInfra(),
        ),
    )
    req = rpc_pb2.ScheduleBuildRequest(template_build_id=44)
    self.call(self.api.ScheduleBuild, req)
    add_async.assert_called_once_with(mock.ANY)
    actual_req = add_async.mock_calls[0][1][0].schedule_build_request
    self.assertFalse(actual_req.HasField('properties'))
    self.assertFalse(actual_req.HasField('gitiles_commit'))
    self.assertFalse(actual_req.HasField('exe'))

  @mock.patch('service.get_async', autospec=True)
  def test_schedule_with_template_build_id_not_found(self, get_async):
    get_async.return_value = future(None)
    req = rpc_pb2.ScheduleBuildRequest(template_build_id=44,)
    self.call(
        self.api.ScheduleBuild, req, expected_code=prpc.StatusCode.NOT_FOUND
    )

  @mock.patch('service.get_async', autospec=True)
  def test_schedule_with_unauthorized_template_build_id(self, get_async):
    get_async.return_value = future(
        test_util.build(
            id=44,
            builder=dict(project='chromium', bucket='another', builder='linux'),
        ),
    )
    req = rpc_pb2.ScheduleBuildRequest(template_build_id=44)
    self.call(
        self.api.ScheduleBuild,
        req,
        expected_code=prpc.StatusCode.PERMISSION_DENIED,
    )

  def test_forbidden(self):
    self.perms['chromium/try'].remove(user.PERM_BUILDS_ADD)
    req = rpc_pb2.ScheduleBuildRequest(
        builder=dict(project='chromium', bucket='try', builder='linux'),
    )
    self.call(
        self.api.ScheduleBuild,
        req,
        expected_code=prpc.StatusCode.PERMISSION_DENIED
    )


class BatchTests(BaseTestCase):

  @mock.patch('service.get_async', autospec=True)
  def test_errors(self, get_async):
    get_async.return_value = future(None)

    req = rpc_pb2.BatchRequest(
        requests=[
            dict(get_build=dict(id=1)),
            dict(),
        ],
    )
    self.assertEqual(
        self.call(self.api.Batch, req),
        rpc_pb2.BatchResponse(
            responses=[
                dict(
                    error=dict(
                        code=prpc.StatusCode.UNIMPLEMENTED.value,
                        message='unimplemented',
                    ),
                ),
                dict(
                    error=dict(
                        code=prpc.StatusCode.INVALID_ARGUMENT.value,
                        message='request is not specified',
                    ),
                ),
            ]
        )
    )

  @mock.patch('creation.add_many_async', autospec=True)
  @mock.patch('service.get_async', autospec=True)
  def test_schedule_build_requests(self, get_async, add_many_async):
    linux_builder = dict(project='chromium', bucket='try', builder='linux')
    win_builder = dict(project='chromium', bucket='try', builder='windows')

    get_async.return_value = future(
        test_util.build(id=23, builder=linux_builder),
    )

    add_many_async.return_value = future([
        (test_util.build(id=42), None),
        (test_util.build(id=43), None),
        (test_util.build(id=44), None),
        (None, errors.InvalidInputError('bad')),
        (None, Exception('unexpected')),
        (None, auth.AuthorizationError('bad')),
    ])

    req = rpc_pb2.BatchRequest(
        requests=[
            dict(schedule_build=dict(builder=linux_builder)),
            dict(
                schedule_build=dict(
                    builder=linux_builder, fields=dict(paths=['tags'])
                )
            ),
            dict(schedule_build=dict(template_build_id=23)),
            dict(
                schedule_build=dict(
                    builder=linux_builder, fields=dict(paths=['wrong-field'])
                )
            ),
            dict(schedule_build=dict(builder=win_builder)),
            dict(schedule_build=dict(builder=win_builder)),
            dict(schedule_build=dict(builder=win_builder)),
            dict(
                schedule_build=dict(
                    builder=dict(
                        project='chromium', bucket='forbidden', builder='nope'
                    ),
                )
            ),
            dict(
                schedule_build=dict(),  # invalid request
            ),
        ],
    )

    res = self.call(self.api.Batch, req)

    codes = [r.error.code for r in res.responses]
    self.assertEqual(
        codes, [
            prpc.StatusCode.OK.value,
            prpc.StatusCode.OK.value,
            prpc.StatusCode.OK.value,
            prpc.StatusCode.INVALID_ARGUMENT.value,
            prpc.StatusCode.INVALID_ARGUMENT.value,
            prpc.StatusCode.INTERNAL.value,
            prpc.StatusCode.PERMISSION_DENIED.value,
            prpc.StatusCode.PERMISSION_DENIED.value,
            prpc.StatusCode.INVALID_ARGUMENT.value,
        ]
    )
    self.assertEqual(res.responses[0].schedule_build.id, 42)
    self.assertFalse(len(res.responses[0].schedule_build.tags))
    self.assertTrue(len(res.responses[1].schedule_build.tags))
    self.assertEqual(res.responses[2].schedule_build.id, 44)

  @mock.patch('service.get_async', autospec=True)
  def test_schedule_build_requests_error(self, get_async):
    get_async.return_value = future(None)
    req = rpc_pb2.BatchRequest(
        requests=[
            dict(schedule_build=dict(template_build_id=121)),
        ],
    )
    self.assertEqual(
        self.call(self.api.Batch, req),
        rpc_pb2.BatchResponse(
            responses=[
                dict(
                    error=dict(
                        code=prpc.StatusCode.NOT_FOUND.value,
                        message='build 121 is not found',
                    ),
                ),
            ]
        )
    )


class BuildPredicateToSearchQueryTests(BaseTestCase):

  def test_project(self):
    predicate = rpc_pb2.BuildPredicate(builder=dict(project='chromium'),)
    q = api.build_predicate_to_search_query(predicate)
    self.assertEqual(q.project, 'chromium')
    self.assertFalse(q.bucket_ids)
    self.assertFalse(q.tags)

  def test_project_bucket(self):
    predicate = rpc_pb2.BuildPredicate(
        builder=dict(project='chromium', bucket='try'),
    )
    q = api.build_predicate_to_search_query(predicate)
    self.assertFalse(q.project)
    self.assertEqual(q.bucket_ids, ['chromium/try'])
    self.assertFalse(q.tags)

  def test_project_bucket_builder(self):
    predicate = rpc_pb2.BuildPredicate(
        builder=dict(project='chromium', bucket='try', builder='linux-rel'),
    )
    q = api.build_predicate_to_search_query(predicate)
    self.assertFalse(q.project)
    self.assertEqual(q.bucket_ids, ['chromium/try'])
    self.assertEqual(q.builder, 'linux-rel')

  def test_create_time(self):
    predicate = rpc_pb2.BuildPredicate()
    predicate.create_time.start_time.FromDatetime(datetime.datetime(2018, 1, 1))
    predicate.create_time.end_time.FromDatetime(datetime.datetime(2018, 1, 2))
    q = api.build_predicate_to_search_query(predicate)
    self.assertEqual(q.create_time_low, datetime.datetime(2018, 1, 1))
    self.assertEqual(q.create_time_high, datetime.datetime(2018, 1, 2))

  def test_build_range(self):
    predicate = rpc_pb2.BuildPredicate(
        build=rpc_pb2.BuildRange(start_build_id=100, end_build_id=90),
    )
    q = api.build_predicate_to_search_query(predicate)
    self.assertEqual(q.build_low, 89)
    self.assertEqual(q.build_high, 101)

  def test_canary(self):
    predicate = rpc_pb2.BuildPredicate(canary=common_pb2.YES)
    q = api.build_predicate_to_search_query(predicate)
    self.assertEqual(q.canary, True)

  def test_non_canary(self):
    predicate = rpc_pb2.BuildPredicate(canary=common_pb2.NO)
    q = api.build_predicate_to_search_query(predicate)
    self.assertEqual(q.canary, False)
