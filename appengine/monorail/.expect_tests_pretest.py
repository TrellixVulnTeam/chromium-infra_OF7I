# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# pylint: disable=undefined-variable

import os
import sys
lib_path = os.path.join(
    os.path.dirname(os.path.realpath(pretest_filename)), 'lib')
sys.path.insert(0, lib_path)
import google
google.__path__.insert(0, os.path.join(lib_path, 'google'))
import google.auth
import google.auth.transport.requests


def _fix_sys_path_for_appengine(pretest_filename):
  """Adds the App Engine built-in libraries to sys.path."""
  # Scan the path to this file to locate the infra repo base directory.
  infra_base_dir = os.path.abspath(pretest_filename)
  pos = infra_base_dir.rfind('/infra/appengine')
  if pos == -1:
    return
  infra_base_dir = infra_base_dir[:pos + len('/infra')]

  # Remove the base infra directory from the path, since this isn't available
  # on appengine.
  sys.path.remove(infra_base_dir)

  # Add the google_appengine directory.
  pretest_APPENGINE_ENV_PATH = os.path.join(
      os.path.dirname(infra_base_dir), 'gcloud', 'platform', 'google_appengine')
  sys.path.insert(0, pretest_APPENGINE_ENV_PATH)

  # Unfortunate hack, because of appengine.
  import dev_appserver as pretest_dev_appserver
  pretest_dev_appserver.fix_sys_path()

  # Remove google_appengine SDK from sys.path after use.
  sys.path.remove(pretest_APPENGINE_ENV_PATH)

  # This is not added by fix_sys_path.
  sys.path.append(os.path.join(pretest_APPENGINE_ENV_PATH, 'lib', 'mox'))


def _load_appengine_config(pretest_filename):
  """Runs appengine_config.py to reproduce the App Engine environment."""
  app_dir = os.path.abspath(os.path.dirname(pretest_filename))

  # Add the application directory to sys.path.
  inserted = False
  if app_dir not in sys.path:
    sys.path.insert(0, app_dir)
    inserted = True

  # import appengine_config.py, thus executing its contents.
  import appengine_config  # Unused Variable pylint: disable=W0612

  # Clean up.
  if inserted:
    sys.path.remove(app_dir)


# Using pretest_filename is magic, because it is available in the locals() of
# the script which execfiles this file.
_fix_sys_path_for_appengine(pretest_filename)

os.environ['SERVER_SOFTWARE'] = 'test ' + os.environ.get('SERVER_SOFTWARE', '')
os.environ['CURRENT_VERSION_ID'] = 'test.123'
os.environ.setdefault('NO_GCE_CHECK', 'True')

# Load appengine_config from the appengine project to ensure that any changes to
# configuration there are available to the tests (e.g. sys.path modifications,
# namespaces, etc.). This is according to
# https://cloud.google.com/appengine/docs/python/tools/localunittesting
_load_appengine_config(pretest_filename)
