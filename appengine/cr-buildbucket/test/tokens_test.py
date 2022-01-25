# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from components import auth
from components.auth import b64
from go.chromium.org.luci.buildbucket.proto import token_pb2
from testing_utils import testing

import tokens


def validate_build_token(token):
  """Raises auth.InvalidTokenError if the token is invalid."""
  try:
    binary = b64.decode(token)
    env = token_pb2.TokenEnvelope()
    env.MergeFromString(binary)
    body = token_pb2.TokenBody()
    body.MergeFromString(env.payload)
  except:  # pragma: no cover.
    raise auth.InvalidTokenError('failed to deserialize %r' % token)


class BuildTokenTests(testing.AppengineTestCase):

  def test_roundtrip_simple(self):
    build_id = 1234567890
    token = tokens.generate_build_token(build_id)
    validate_build_token(token)
