# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Provides functions for handling tokens."""
import os

from components.auth import b64
from go.chromium.org.luci.buildbucket.proto import token_pb2


def generate_build_token(build_id):
  """Returns a token associated with the build."""
  body = token_pb2.TokenBody(
      build_id=build_id,
      purpose=token_pb2.TokenBody.BUILD,
      state=os.urandom(16),
  )
  env = token_pb2.TokenEnvelope(
      version=token_pb2.TokenEnvelope.UNENCRYPTED_PASSWORD_LIKE
  )
  env.payload = body.SerializeToString()
  return b64.encode(env.SerializeToString())
