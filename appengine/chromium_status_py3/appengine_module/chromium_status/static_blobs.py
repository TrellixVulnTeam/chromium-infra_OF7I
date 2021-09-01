# Copyright (c) 2012 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Serve static images."""

import logging
from flask import abort, redirect

VALID_RESOURCES = ['favicon.ico', 'logo.png']


class ServeHandler:
  """Serves a static file."""

  def get(self, resource):
    if not resource in VALID_RESOURCES:
      logging.warning('Unknown resource "%s"' % resource)
      abort(404)
    # TODO (crbug.com/1121016): Perhaps support caching?
    return redirect('/static/' + resource)
