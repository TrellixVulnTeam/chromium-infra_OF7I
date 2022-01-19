#!/usr/bin/env python3
# Copyright 2022 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Script to launch the Monorail release tarball builder.

It can be used to build a tarball with Monorail code based on a release branch
(i.e. `refs/releases/monorail/...`). It triggers a go/monorail-release-tarballs
build that uploads the release tarball and triggers its deployment to
monorail-dev, after which it can be promoted to monorail-prod.

See go/monorail-deploy for more details.
"""

import argparse
import json
import subprocess
import sys
import urllib.error
import urllib.request


INFRA_GIT = 'https://chromium.googlesource.com/infra/infra'
TARBALL_BUILDER = 'infra-internal/monorail-release/monorail-release-tarballs'


def resolve_commit(ref):
  """Queries gitiles for a commit hash matching the given infra.git ref.

  Args:
    ref: a `refs/...` ref to resolve into a commit.

  Returns:
    None if there's no such ref, a gitiles commit URL otherwise.
  """
  try:
    resp = urllib.request.urlopen('%s/+/%s?format=JSON' % (INFRA_GIT, ref))
  except urllib.error.HTTPError as exc:
    if exc.code == 404:
      return None
    raise

  # Gitiles JSON responses start with XSS-protection header.
  blob = resp.read()
  if blob.startswith(b')]}\''):
    blob = blob[4:]

  commit = json.loads(blob)['commit']
  return '%s/+/%s' % (INFRA_GIT, commit)


def ensure_logged_in():
  """Ensures `bb` tool is in PATH and the caller is logged in there.

  Returns:
    True if logged in, False if not and we should abort.
  """
  try:
    proc = subprocess.run(['bb', 'auth-info'], capture_output=True)
  except OSError:
    print(
        'Could not find `bb` tool in PATH. It comes with depot_tools. '
        'Make sure depot_tools is in PATH and up-to-date, then try again.')
    return False

  if proc.returncode == 0:
    return True  # already logged in

  # Launch interactive login process.
  proc = subprocess.run(['bb', 'auth-login'])
  if proc.returncode != 0:
    print('Failed to login')
    return False
  return True


def submit_build(ref, commit):
  """Submits a Monorail tarball builder build via `bb` tool.

  Args:
    ref: a `refs/...` ref with the code to build.
    commit: a gitiles commit matching this ref.

  Returns:
    None if failed, a URL to the pending build otherwise.
  """
  cmd = ['bb', 'add', '-json', '-ref', ref, '-commit', commit, TARBALL_BUILDER]
  proc = subprocess.run(cmd, capture_output=True)
  if proc.returncode != 0:
    print(
        'Failed to schedule the build:\n%s'
        % proc.stderr.decode('utf-8').strip())
    return None
  build_id = json.loads(proc.stdout)['id']
  return 'https://ci.chromium.org/b/%s' % build_id


def main():
  parser = argparse.ArgumentParser(
      description='Submits a request to build Monorail tarball for LUCI CD.')
  parser.add_argument(
      'branch', type=str,
      help='a branch to build from: refs/releases/monorail/<num> or just <num>')
  parser.add_argument(
      '--silent', action='store_true',
      help='disable interactive prompts')
  args = parser.parse_args()

  ref = args.branch
  if not ref.startswith('refs/'):
    ref = 'refs/releases/monorail/' + ref

  # `bb add` call wants a concrete git commit SHA1 as input.
  commit = resolve_commit(ref)
  if not commit:
    print('No such release branch: %s' % ref)
    return 1

  # Give a chance to confirm this is the commit we want to build.
  if not args.silent:
    print(
        'Will submit a request to build a Monorail code tarball from %s:\n'
        '  %s\n\n'
        'You may be asked to sign in with your google.com account if it is '
        'the first time you are using this script.\n'
        % (ref, commit)
    )
    if input('Proceed [Y/n]? ') not in ('', 'Y', 'y'):
      return 0

  # Submit the build via `bb` tool.
  if not args.silent and not ensure_logged_in():
    return 1
  build_url = submit_build(ref, commit)
  if not build_url:
    return 1

  print(
      '\nScheduled the build: %s\n'
      '\n'
      'When it completes it will trigger deployment of this release to '
      'monorail-dev. You can then promote it to production using the same '
      'procedure as with regular releases built from `main` branch.\n'
      '\n'
      'Note that if the produced release tarball is 100%% identical to any '
      'previously built tarball (e.g. there were no cherry-picks into the '
      'release branch since it was cut from `main`), an existing tarball and '
      'its version name will be reused.'
      % build_url
  )
  return 0


if __name__ == '__main__':
  sys.exit(main())
