# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""This module is for communicating with ResultDB."""
import logging
import os
import sys

from components.prpc import client as prpc_client
from go.chromium.org.luci.resultdb.proto.v1 import resultdb_pb2
from go.chromium.org.luci.resultdb.proto.v1 import resultdb_prpc_pb2
from google.protobuf.field_mask_pb2 import FieldMask

RESULTDB_HOSTNAME = "results.api.cr.dev"


def query_resultdb(inv_id):
  # TODO(crbug.com/981066): Add pagination
  logging.info("Query test results for invocation %s from resultdb", inv_id)
  client = resultdb_client(RESULTDB_HOSTNAME)
  inv_name = "invocations/" + inv_id
  req = resultdb_pb2.QueryTestResultsRequest(
      invocations=[inv_name],
      read_mask=FieldMask(paths=["*"]),
      page_size=1000,
  )
  resp = client.QueryTestResults(
      req,
      credentials=prpc_client.service_account_credentials(),
  )
  logging.info("Got %d test results", len(resp.test_results))


def resultdb_client(hostname):
  return prpc_client.Client(hostname,
                            resultdb_prpc_pb2.ResultDBServiceDescription)
