# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Testable functions for Bucket."""

import logging
import os
import sys
import subprocess

from infra.path_hacks.depot_tools import _depot_tools as depot_tools_path

# https://chromium.googlesource.com/infra/infra/+/master/infra_libs/logs/README.md
LOGGER = logging.getLogger(__name__)

PROJECT = 'chromium-archive'


class BucketExists(Exception):
  pass
class InvalidBucketName(Exception):
  pass


def gsutil(args):  # pragma: no cover
  target = os.path.join(depot_tools_path, 'gsutil.py')
  cmd = [sys.executable, target, '--']
  cmd.extend(args)
  print 'gsutil',
  print ' '.join(args)
  return subprocess.check_call(cmd, stderr=subprocess.PIPE)


def add_argparse_options(parser):
  """Define command-line arguments."""
  parser.add_argument('bucket', type=str, nargs='+')
  parser.add_argument(
      '--reader',
      '-r',
      type=str,
      action='append',
      default=[],
      help='Add this account as a Storage Legacy Bucket Reader')
  parser.add_argument(
      '--writer',
      '-w',
      type=str,
      action='append',
      default=[],
      help='Add this account as a Storage Legacy Bucket Writer')


def ensure_no_bucket_exists(bucket):
  """Raises an exception if the bucket exists."""
  try:
    gsutil(['ls', '-b', 'gs://%s' % bucket])
  except subprocess.CalledProcessError:
    return
  raise BucketExists('%s already exists.' % bucket)


def bucket_is_public(bucket_name):
  """Verify the name of the bucket and return whether it's public or not."""
  if bucket_name.startswith('chromium-'):
    return True
  elif bucket_name.startswith('chrome-'):
    return False
  else:
    raise InvalidBucketName(
        '%s does not start with either "chromium-" or "chrome-"' % bucket_name)


def run(bucket_name, readers, writers, public):
  ensure_no_bucket_exists(bucket_name)
  gsutil(['mb', '-p', PROJECT, 'gs://%s' % bucket_name])
  for reader in readers:
    gsutil(['acl', 'ch', '-u', '%s:r' % reader, 'gs://%s' % bucket_name])
  for writer in writers:
    gsutil(['acl', 'ch', '-u', '%s:w' % writer, 'gs://%s' % bucket_name])

  if public:
    reader = 'AllUsers'
  else:
    reader = 'google.com'
  gsutil(['acl', 'ch', '-g', '%s:R' % reader, 'gs://%s' % bucket_name])
  gsutil(['defacl', 'ch', '-g', '%s:R' % reader, 'gs://%s' % bucket_name])
