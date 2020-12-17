# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pylint: skip-file

import json
import os
import posixpath
import requests
import traceback

from google.cloud import storage


ISSUE = 'Issue'
PATCH_SET = 'PatchSet'
PATCH = 'Patch'
KNOWN_ENTITY_KINDS = (ISSUE, PATCH_SET, PATCH)


session = requests.Session()
client = storage.Client()


def process_page(request):
  """Fetch, process and upload Rietveld pages.

  Defines a cloud function to fetch pages from an existing Rietveld instance,
  process it to remove dynamic content, and upload the resulting page to Google
  Storage.

  Args:
    request: A dict containing entries for 'path', 'type' and 'private'.
      path: The page to fetch, e.g. '/1234/patchset/5'
      type: One of 'Issue', 'PatchSet' or 'Patch'.
      private: Whether this page is from a private Rietveld issue.
  """
  params = request.get_json(force=True, silent=True)
  try:
    _process_page(params)
    return '', 200

  except:
    print(json.dumps({
      'severity': 'ERROR',
      'message': traceback.format_exc(),
      'params': params,
    }))
    return '', 500


def _process_page(params):
  path = params['Path']
  entity_kind = params['EntityKind']
  private = params['Private']

  assert entity_kind in KNOWN_ENTITY_KINDS, (
      'Expected entity kind to be one of {}, got {}'.format(
          KNOWN_ENTITY_KINDS, entity_kind))

  if not path.startswith('/'):
    path = '/' + path
  response = session.get(
      os.getenv('RIETVELD_HOST') + path,
      headers=_get_auth_headers())

  # Upload page to Google Storage
  bucket = client.get_bucket(os.getenv('BUCKET_NAME'))
  blob = bucket.blob(path)
  blob.upload_from_string(response.text)
  blob.metadata = {
    'Rietveld-Private': private,
    'Status-Code': response.status_code,
  }
  blob.content_type = response.headers['content-type']
  blob.patch()

  # Forward transient errors to the client so tasks can be retried.
  # Content is stored anyways, since some pages consistently fail with internal
  # errors, e.g. https://codereview.chromium.org/135933002/diff/30003/.gitignore
  if response.status_code >= 500 or response.status_code == 429:
    response.raise_for_status()


def _get_auth_headers():
  # Fetch access token from metadata server.
  # https://cloud.google.com/run/docs/securing/service-identity#access_tokens
  TOKEN_URL = ('http://metadata.google.internal/computeMetadata/v1'
               '/instance/service-accounts/default/token')
  TOKEN_HEADERS = {'Metadata-Flavor': 'Google'}

  response = session.get(TOKEN_URL, headers=TOKEN_HEADERS)
  response.raise_for_status()

  # Extract the access token from the response.
  access_token = response.json()['access_token']

  return {'Authorization': 'Bearer {}'.format(access_token)}

