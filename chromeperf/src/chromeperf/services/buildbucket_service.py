# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Encapsulates a simplistic interface to the buildbucket service."""

import json

from chromeperf.services import request

API_BASE_URL = 'https://cr-buildbucket.appspot.com/api/buildbucket/v1/'


def put(bucket, tags, parameters, pubsub_callback=None):
  body = {
      'bucket': bucket,
      'tags': tags,
      'parameters_json': json.dumps(parameters, separators=(',', ':')),
  }
  if pubsub_callback:
    body['pubsub_callback'] = pubsub_callback
  return request.request_json(API_BASE_URL + 'builds', method='PUT', body=body)


def get(job_id):
  """Gets the details of a job via buildbucket's API."""
  return request.request_json(API_BASE_URL + 'builds/%s' % (job_id))
