#!/usr/bin/env python
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import logging
import os
import sys

from install_cipd_packages import cipd_ensure_list


BOOTSTRAP_DIR = os.path.dirname(os.path.abspath(__file__))
BASE_DIR = os.path.dirname(BOOTSTRAP_DIR)


def main():
  parser = argparse.ArgumentParser(prog='python -m %s' % __package__)
  parser.add_argument('-v', '--verbose', action='store_true')
  parser.add_argument(
      '-d', '--dest', default=os.path.dirname(BASE_DIR), help='Output')
  options = parser.parse_args()

  if options.verbose:
    logging.getLogger().setLevel(logging.DEBUG)

  cipd_ensure_list(os.path.join(os.path.abspath(options.dest), 'gcloud'), [
    (
      'infra/3pp/tools/gcloud/${os=mac,linux}-${arch=amd64}',
      'version:272.0.0.chromium0',
    ),
  ])
  return 0


if __name__ == '__main__':
  sys.exit(main())
