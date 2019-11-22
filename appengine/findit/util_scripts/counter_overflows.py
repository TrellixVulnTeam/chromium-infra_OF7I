#!/usr/bin/python
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Find out which targets/files/functions are causing counter overflows.

Download the merge script logs from a code coverage build and parse them to find
whether any raw profiles contain any counter overflows, output these findings to
stdout.

Optionally, download the offending profraw files.

Requires `bb` and `swarming.py` to be accessible on PATH.
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import tempfile

MERGE_SCRIPT_LOG_NAME = 'Merge script log'


def _DumpToLocalStorage(data, path, options):
  path = os.path.join(options.out_dir, path)
  if not os.path.exists(os.path.dirname(path)):
    os.makedirs(os.path.dirname(path))
  with open(path, 'w') as f:
    f.write(data)


def _ExtractIndividuaOutputsFromMergeScriptOutput(lines, path, options,
                                                  results):
  """Extracts logs for individual llvm-profdata runs from merge script log.

  The merge script in chromium/src/testing/merge_scripts/code_coverage checks
  that individual .profraw files are valid by merging them by themselves.

  If a counter overflow is detected (via output parsing), the whole output is
  put in a single line of the log (with newlines escaped).

  This function undoes this and dumps the original output to local storage.
  """
  OVERFLOW_PREFIX = 'Counter overflow: '
  OUTPUT_PREFIX = 'output: '
  i = 0
  for line in lines:
    if OVERFLOW_PREFIX in line:
      relevant_bit = line.split(OVERFLOW_PREFIX)[1].split(OUTPUT_PREFIX)[1]
      unescaped_log = relevant_bit.replace('\\n', '\n')
      current_path = '%s_%d.log' % (path, i)
      if options.save_profile_merge_output:
        _DumpToLocalStorage(unescaped_log, current_path, options)
      for source, func_name in ProcessProfdataLog(unescaped_log, options):
        results.setdefault(source, set())
        results[source].add(func_name)
      i += 1


def _FindOverflowsInStep(build_id, step_name, options, results):
  short_step_names = {}

  def _Deduplicate(s):
    """Deduplicates strings by adding a numerical suffix if repeated."""
    if s in short_step_names:
      short_step_names[s] += 1
      return s + '_%d' % short_step_names[s]
    short_step_names[s] = 0
    return s

  command = ['bb', 'log', build_id, step_name, MERGE_SCRIPT_LOG_NAME]
  log = subprocess.check_output(command)
  build_number = build_id.split('/')[-1]
  short_step_name = _Deduplicate(step_name.split()[0])
  if options.save_script_output:
    _DumpToLocalStorage(
        log, 'merge-script-out/%s/%s.log' % (build_number, short_step_name),
        options)
  _ExtractIndividuaOutputsFromMergeScriptOutput(
      log.splitlines(), 'overflows/%s/%s' % (build_number, short_step_name),
      options, results)


def _FindOverflowsInBuild(builder, build_number, options, results):
  build_id = '%s/%d' % (builder, build_number)
  command = ['bb', 'get', build_id, '-A', '-json']
  raw_build = subprocess.check_output(command)
  build = json.loads(raw_build)
  if options.save_build_json:
    _DumpToLocalStorage(raw_build, 'builds/%d.json' % build_number, options)
  for step in build['steps']:
    if any(
        log['name'] == MERGE_SCRIPT_LOG_NAME for log in step.get('logs', [])):
      _FindOverflowsInStep(build_id, step['name'], options, results)


def _DownloadIsolatedOutput(task_id, options):
  command = [
      'swarming.py', 'collect', '--swarming', 'chromium-swarm.appspot.com',
      '--task-output-dir=%s/swarmout/%s' % (options.temp_dir, task_id), task_id
  ]
  try:
    _ = subprocess.check_output(command, stderr=subprocess.STDOUT)
  except subprocess.CalledProcessError as cpe:
    print 'Swarming.py failed to do %r' % command
    print cpe.output


def _ParseBadProfileLog(log):
  # This is how llvm-profdata output is formatted for counter overflows.
  OVERFLOW_RE = re.compile(r'(.*): (.*):(.*): Counter overflow')
  for line in log.splitlines():
    line = line.strip().strip("'")
    match = OVERFLOW_RE.match(line)
    if match:
      profraw_path, file_name, function_name = match.groups()
      yield profraw_path, file_name, function_name


def _ParseProfilePath(profraw_path):
  task_id, dirname, filename = profraw_path.split('/')[-3:]
  return task_id, dirname, filename


def _DownloadRawProfile(task_id, dirname, filename, options):
  """Downloads the isolated output if needed, then extracts the raw profile."""
  # /0/ below is needed because swarming.py creates subdirs for every shard.
  # Apparently each shard triggered by the recipe is a one-shard task as far as
  # swarming is concerned.
  local_path = '%s/swarmout/%s/0/%s/%s' % (options.temp_dir, task_id, dirname,
                                           filename)
  if not os.path.exists(local_path):
    _DownloadIsolatedOutput(task_id, options)
  if os.path.exists(local_path):
    out_dir = os.path.join(options.out_dir, 'profiles', task_id)
    if not os.path.exists(out_dir):
      os.makedirs(out_dir)
    shutil.copy(local_path, out_dir)


def ProcessProfdataLog(log, options):
  for profile_path, sourcefile_name, function_name in _ParseBadProfileLog(log):
    if options.save_bad_raw_profiles:
      task_id, profile_dir_name, profile_file_name = _ParseProfilePath(
          profile_path)
      _DownloadRawProfile(task_id, profile_dir_name, profile_file_name, options)
    yield sourcefile_name, function_name


def GetOptions():
  parser = argparse.ArgumentParser(
      description=
      'Download info about code coverage counter overflows from a luci build',
      epilog=__doc__)
  parser.add_argument('-b', '--save-build-json', action='store_true')
  parser.add_argument('-s', '--save-script-output', action='store_true')
  parser.add_argument('-p', '--save-profile-merge-output', action='store_true')
  parser.add_argument('-r', '--save-bad-raw-profiles', action='store_true')
  parser.add_argument(
      '-c',
      '--clean-task-outputs',
      action='store_true',
      help='Delete the temp directory the isolated output is written to')
  parser.add_argument(
      '--out-dir', default='overflow-out/', help='Directory to save data to')
  parser.add_argument(
      'builds', help='E.g. chromium/ci/mac-code-coverage/352', nargs='+')
  parser.add_argument('--temp-dir')
  options = parser.parse_args()
  return options


def main():
  options = GetOptions()
  # This will map a source file name to a set of function names.
  results = {}

  for build in options.builds:
    builder, build_number = build.rsplit('/', 1)
    # Validate input format.
    _project, _bucket, _builder_name = builder.split('/')
    build_number_int = int(build_number)
    delete_temp_dir = False
    if options.temp_dir:
      print 'Leaking isolated outputs to %s' % options.temp_dir
    else:
      options.temp_dir = tempfile.mkdtemp()
      # Only delete the temp dir if we create it here in the line above.
      delete_temp_dir = True
    _FindOverflowsInBuild(builder, build_number_int, options, results)
    if delete_temp_dir:
      shutil.rmtree(options.temp_dir)
  for source in sorted(results.keys()):
    print '%s:' % source
    for fn in sorted(results[source]):
      print '    %s' % fn


if __name__ == '__main__':
  main()
