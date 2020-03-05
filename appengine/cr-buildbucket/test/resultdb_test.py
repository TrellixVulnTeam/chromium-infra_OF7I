# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from components import utils
utils.fix_protobuf_package()

from components import net
from components.prpc import client
from testing_utils import testing

from go.chromium.org.luci.resultdb.proto.rpc.v1 import invocation_pb2
from test.test_util import build_bundle, future, future_exception
import resultdb


def _make_build(build_id, hostname='rdb.example.com', invocation=None):
  bundle = build_bundle(
      id=build_id,
      for_creation=True,
      infra=dict(
          resultdb=dict(
              hostname=hostname or '',
              invocation=invocation or '',
          )
      )
  )
  bundle.put()
  return bundle.build


def _mock_create_request_async(response, update_token):

  def inner(*_, **kwargs):
    ret = future(response.SerializeToString())
    kwargs['response_headers']['update-token'] = update_token
    return ret

  return inner


class ResultDBTest(testing.AppengineTestCase):

  def setUp(self):
    super(ResultDBTest, self).setUp()
    self.patch('components.net.request_async')
    self.build = None

  def test_no_hostname(self):
    self.build = _make_build(1, hostname=None)
    self.assertFalse(resultdb.sync(self.build))
    self.assertFalse(net.request_async.called)

  def test_has_invocation(self):
    self.build = _make_build(2, invocation='invocations/build:2')
    self.assertFalse(resultdb.sync(self.build))
    self.assertFalse(net.request_async.called)

  def test_cannot_create_invocation(self):
    self.build = _make_build(3)
    net.request_async.side_effect = [
        future_exception(
            net.Error(
                'Internal Error',
                500,
                'Internal Error',
                headers={'X-Prpc-Grpc-Code': '2'}
            )
        )
    ]
    with self.assertRaises(client.RpcError):
      resultdb.sync(self.build)

  def test_invocation_created(self):
    self.build = _make_build(4)
    response = invocation_pb2.Invocation()
    response.name = 'invocations/build:4'
    net.request_async.side_effect = _mock_create_request_async(
        response, 'FakeUpdateToken'
    )
    self.assertTrue(resultdb.sync(self.build))
    # if called a second time there should be no changes written to datastore.
    self.assertFalse(resultdb.sync(self.build))
