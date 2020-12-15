# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import subprocess
import sys


def main():
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--server', default='https://gcr.io', help='Docker server to connect to')
  parser.add_argument(
      '--service-account-token-file',
      required=True,
      type=file,
      help='File containing service acccount token used to authenticate with '
      'Docker server')
  parser.add_argument(
      '--config-file',
      required=True,
      help='Location of the Docker config file.')
  args = parser.parse_args()

  token = args.service_account_token_file.read()
  try:
    subprocess.check_call([
        'docker', '--config', args.config_file, 'login', '--username',
        'oauth2accesstoken', '--password', token, args.server
    ])
  except subprocess.CalledProcessError as e:
    # Censor service account key to avoid leaking it to the logs.
    e.cmd = [arg if arg != token else '<task-account-token>' for arg in e.cmd]
    raise


if __name__ == '__main__':
  sys.exit(main())
