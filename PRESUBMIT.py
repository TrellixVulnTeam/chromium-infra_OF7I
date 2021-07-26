# Copyright (c) 2014 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Top-level presubmit script for infra.

See http://dev.chromium.org/developers/how-tos/depottools/presubmit-scripts for
details on the presubmit API built into gcl.
"""

import os


DISABLED_TESTS = [
    '.*appengine/chromium_status/tests/main_test.py',
    '.*appengine/chromium_build/app_test.py',
]

DISABLED_PYLINT_WARNINGS = [
    'no-init',                # Class has no __init__ method
    'super-init-not-called',  # __init__ method from base class is not called
]

DISABLED_PROJECTS = [
    # Taken care of by appengine/PRESUBMIT.py
    'appengine/*',
    'infra/services/lkgr_finder',

    # Don't bother pylinting (these could also move to .gitignore):
    '.*/__pycache__',
    '\.git',
    '\.wheelcache',
    'bootstrap/virtualenv-ext',
]

# List of third_party direcories.
THIRD_PARTY_DIRS = [
  'appengine/third_party',
]

# List of directories to jshint.
JSHINT_PROJECTS = [
  'appengine/chromium_cq_status',
  'appengine/chromium_status',
  'appengine/milo',
]
# List of blacklisted directories that we don't want to jshint.
JSHINT_PROJECTS_BLACKLIST = THIRD_PARTY_DIRS

# Files that must not be modified (regex)
# Paths tested are relative to the directory containing this file.
# Ex: infra/libs/logs.py
NOFORK_PATHS = []

# This project is whitelisted to use Typescript on a trial basis.
ROTANG_DIR = os.path.join('go', 'src', 'infra', 'appengine', 'rotang')
CHOPSUI_DIR = os.path.join('crdx', 'chopsui-npm')


def CommandInGoEnv(input_api, output_api, name, cmd, kwargs):
  """Returns input_api.Command that wraps |cmd| with invocation to go/env.py.

  env.py makes golang tools available in PATH. It also bootstraps Golang dev
  environment if necessary.
  """
  if input_api.is_committing:
    error_type = output_api.PresubmitError
  else:
    error_type = output_api.PresubmitPromptWarning
  full_cmd = [
    'vpython',
    input_api.os_path.join(input_api.change.RepositoryRoot(), 'go', 'env.py'),
  ]
  full_cmd.extend(cmd)
  return input_api.Command(
      name=name,
      cmd=full_cmd,
      kwargs=kwargs,
      message=error_type)


def GoCheckers(input_api, output_api):
  affected_files = sorted([
    f.AbsoluteLocalPath()
    for f in input_api.AffectedFiles(include_deletes=False)
    if f.AbsoluteLocalPath().endswith('.go') and
    not any(f.AbsoluteLocalPath().endswith(x) for x in ('.pb.go', '.gen.go'))
  ])
  if not affected_files:
    return []
  stdin = '\n'.join(affected_files)

  tool_names = ['gofmt', 'govet']
  ret = []
  for tool_name in tool_names:
    cmd = [
      input_api.python_executable,
      input_api.os_path.join(input_api.change.RepositoryRoot(),
                             'go', 'check.py'),
      tool_name,
    ]
    if input_api.verbose:
      cmd.append("--verbose")
    ret.append(
      CommandInGoEnv(
          input_api, output_api,
          name='Check %s (%d files)' % (tool_name, len(affected_files)),
          cmd=cmd,
          kwargs={'stdin': stdin}),
    )
  return ret


def GoCheckGoModTidy(input_api, output_api):
  def is_interesting(p):
    return p == 'DEPS' or p.endswith(('.go', 'go.mod', 'go.sum'))

  run = any(
      is_interesting(f.LocalPath())
      for f in input_api.change.AffectedFiles(include_deletes=True))
  if not run:
    return []

  # Don't run "go install ...", it may blow up with untidy go.mod.
  env = os.environ.copy()
  env['INFRA_GO_SKIP_TOOLS_INSTALL'] = '1'

  root = input_api.change.RepositoryRoot()
  return input_api.RunTests([
      CommandInGoEnv(
          input_api,
          output_api,
          name='Check go.mod tidiness',
          cmd=[
              input_api.python_executable,
              os.path.join(
                  root,
                  'go',
                  'src',
                  'go.chromium.org',
                  'luci',
                  'scripts',
                  'check_go_mod_tidy.py',
              ),
              os.path.join(root, 'go', 'src', 'infra'),
          ],
          kwargs={'env': env}),
  ])


# Forked from depot_tools/presubmit_canned_checks._FetchAllFiles
def FetchAllFiles(input_api, files_to_check, files_to_skip):
  import datetime
  start_time = datetime.datetime.now()
  def Find(filepath, filters):
    return any(input_api.re.match(item, filepath) for item in filters)

  repo_path = input_api.PresubmitLocalPath()
  def MakeRootRelative(dirpath, item):
    path = input_api.os_path.join(dirpath, item)
    # Poor man's relpath:
    if path.startswith(repo_path):  # pragma: no cover
      return path[len(repo_path) + 1:]
    return path  # pragma: no cover

  dirs_walked = []

  files = []
  for dirpath, dirnames, filenames in input_api.os_walk(repo_path):
    dirs_walked.append(dirpath)
    for item in dirnames[:]:
      filepath = MakeRootRelative(dirpath, item)
      if Find(filepath, files_to_skip):
        dirnames.remove(item)
    for item in filenames:
      filepath = MakeRootRelative(dirpath, item)
      if Find(filepath, files_to_check) and not Find(filepath, files_to_skip):
        files.append(filepath)
  duration = datetime.datetime.now() - start_time
  input_api.logging.info('FetchAllFiles found %s files, searching '
      '%s directories in %ss' % (len(files), len(dirs_walked),
      duration.total_seconds()))
  return files


def EnvAddingPythonPath(input_api, extra_python_paths):
  # Copy the system path to the environment so pylint can find the right
  # imports.
  # FIXME: Is there no nicer way to pass a modified python path
  # down to subprocess?
  env = input_api.environ.copy()
  import sys
  env['PYTHONPATH'] = input_api.os_path.pathsep.join(
      extra_python_paths + sys.path).encode('utf8')
  return env


# Forked with prejudice from depot_tools/presubmit_canned_checks.py
def PylintFiles(input_api, output_api, files, pylint_root, disabled_warnings,
                extra_python_paths):  # pragma: no cover
  input_api.logging.debug('Running pylint on: %s', files)

  # FIXME: depot_tools should be right next to infra, however DEPS
  # recursion into build/DEPS does not seem to be working: crbug.com/410070
  canned_checks_path = input_api.canned_checks.__file__
  canned_checks_path = input_api.os_path.abspath(canned_checks_path)
  depot_tools_path = input_api.os_path.dirname(canned_checks_path)

  pylint_args = ['-d', ','.join(disabled_warnings)]

  env = EnvAddingPythonPath(input_api, extra_python_paths)

  pytlint_path = input_api.os_path.join(depot_tools_path, 'pylint-1.5')

  # Pass args via stdin, because windows (command line limit).
  return input_api.Command(
      name=('Pylint (%s files%s)' % (
            len(files), ' under %s' % pylint_root if pylint_root else '')),
      cmd=['vpython',
           pytlint_path,
           '--args-on-stdin'],
      kwargs={'env': env, 'stdin': '\n'.join(pylint_args + files)},
      message=output_api.PresubmitError)


def IgnoredPaths(input_api): # pragma: no cover
  # This computes the list if repository-root-relative paths which are
  # ignored by .gitignore files. There is probably a faster way to do this.
  status_output = input_api.subprocess.check_output(
      ['git', 'status', '--porcelain', '--ignored'])
  statuses = [(line[:2], line[3:]) for line in status_output.splitlines()]
  return [
    input_api.re.escape(path) for (mode, path) in statuses
    if mode in ('!!', '??') and not path.endswith('.pyc')
  ]


def PythonRootForPath(input_api, path):
  # For each path, walk up dirtories until find no more __init__.py
  # The directory with the last __init__.py is considered our root.
  root = input_api.os_path.dirname(path)
  while True:
    root_parent = input_api.os_path.dirname(root)
    parent_init = input_api.os_path.join(root_parent, '__init__.py')
    if not input_api.os_path.isfile(parent_init):
      break
    root = root_parent
  return root


def GroupPythonFilesByRoot(input_api, paths):
  sorted_paths = sorted(paths)
  import collections
  grouped_paths = collections.defaultdict(list)
  for path in sorted_paths:
    # FIXME: This doesn't actually need to touch the filesystem if we can
    # trust that 'paths' contains all __init__.py paths we care about.
    root = PythonRootForPath(input_api, path)
    grouped_paths[root].append(path)
  # Convert back to a normal dict before returning.
  return dict(grouped_paths)


def DirtyRootsFromAffectedFiles(changed_py_files, root_to_paths):
  # Compute root_groups for all python files
  path_to_root = {}
  for root, paths in root_to_paths.items():
    for path in paths:
      path_to_root[path] = root

  # Using the above mapping, compute the actual roots we need to run
  dirty_roots = set()
  for path in changed_py_files:
    dirty_roots.add(path_to_root[path])
  return dirty_roots


def NoForkCheck(input_api, output_api): # pragma: no cover
  """Warn when a file that should not be modified is modified.

  This is useful when a file is to be moved to a different place
  and is temporarily copied to preserve backward compatibility. We don't
  want the original file to be modified.
  """
  black_list_re = [input_api.re.compile(regexp) for regexp in NOFORK_PATHS]
  offending_files = []
  for filename in input_api.AffectedTextFiles():
    if any(regexp.search(filename.LocalPath()) for regexp in black_list_re):
      offending_files.append(filename.LocalPath())
  if offending_files:
    return [output_api.PresubmitPromptWarning(
      'You modified files that should not be modified. Look for a NOFORK file\n'
      + 'in a directory above those files to get more context:\n%s'
      % '\n'.join(offending_files)
      )]
  return []


def EmptiedFilesCheck(input_api, output_api): # pragma: no cover
  """Warns if a CL empties a file.

  This is not handled properly by apply_patch from depot_tools: the
  file would not exist at all on trybot checkouts.
  """
  empty_files = []
  infra_root = input_api.PresubmitLocalPath()
  for filename in input_api.AffectedTextFiles():
    fullname = input_api.os_path.join(infra_root, filename.LocalPath())
    if not input_api.os_stat(fullname).st_size:
      empty_files.append(filename.LocalPath())
  if empty_files:
    return [output_api.PresubmitPromptWarning(
      'Empty files found in the CL. This can cause trouble on trybots\n'
      + 'if your change depends on the existence of those files:\n%s'
      % '\n'.join(empty_files)
      )]
  return []


def BrokenLinksChecks(input_api, output_api):  # pragma: no cover
  """Complains if there are broken committed symlinks."""
  stdout = input_api.subprocess.check_output(['git', 'ls-files'])
  files = stdout.splitlines()
  output = []
  infra_root = input_api.PresubmitLocalPath()
  for filename in files:
    fullname = input_api.os_path.join(infra_root, filename)
    if (input_api.os_path.lexists(fullname)
        and not input_api.os_path.exists(fullname)):
      output.append(output_api.PresubmitError('Broken symbolic link: %s'
                                              % filename))
  return output


def PylintChecks(input_api, output_api, only_changed):  # pragma: no cover
  infra_root = input_api.PresubmitLocalPath()
  # DEPS specifies depot_tools, as sibling of infra.
  venv_path = input_api.os_path.join(infra_root, 'ENV', 'lib', 'python2.7')

  # Cause all pylint commands to execute in the virtualenv
  input_api.python_executable = (
    input_api.os_path.join(infra_root, 'ENV', 'bin', 'python'))

  files_to_check = ['.*\.py$']
  files_to_skip = list(input_api.DEFAULT_FILES_TO_SKIP)
  # FIXME: files_to_skip are regexes, but DISABLED_PROJECTS aren't.
  files_to_skip += DISABLED_PROJECTS
  files_to_skip += [
    '.*_pb2\.py',
    'chromeperf/.*',  # TODO(crbug.com/1123486): pylint for Python3
  ]
  # TODO(phajdan.jr): pylint recipes-py code (http://crbug.com/617939).
  files_to_skip += [r'^recipes/recipes\.py$']
  files_to_skip += IgnoredPaths(input_api)

  extra_syspaths = [venv_path]

  source_filter = lambda path: input_api.FilterSourceFile(
      path, files_to_check=files_to_check, files_to_skip=files_to_skip)
  changed_py_files = [f.LocalPath()
      for f in input_api.AffectedSourceFiles(source_filter)]

  if only_changed:
    if changed_py_files:
      input_api.logging.info('Running pylint on %d files',
                             len(changed_py_files))
      return [PylintFiles(input_api, output_api, changed_py_files, None,
                          DISABLED_PYLINT_WARNINGS, extra_syspaths)]

    return []

  all_python_files = FetchAllFiles(input_api, files_to_check, files_to_skip)
  root_to_paths = GroupPythonFilesByRoot(input_api, all_python_files)
  dirty_roots = DirtyRootsFromAffectedFiles(changed_py_files, root_to_paths)

  tests = []
  for root_path in sorted(dirty_roots):
    python_files = root_to_paths[root_path]
    if python_files:
      if root_path == '':
        root_path = input_api.PresubmitLocalPath()
      input_api.logging.info('Running pylint on %d files under %s',
          len(python_files), root_path)
      syspaths = extra_syspaths + [root_path]
      tests.append(PylintFiles(input_api, output_api, python_files, root_path,
        DISABLED_PYLINT_WARNINGS, syspaths))
  return tests


def GetAffectedJsFiles(input_api, include_deletes=False):
  """Returns a list of absolute paths to modified *.js files."""
  infra_root = input_api.PresubmitLocalPath()
  whitelisted_paths = [
      input_api.os_path.join(infra_root, path)
      for path in JSHINT_PROJECTS]
  blacklisted_paths = [
      input_api.os_path.join(infra_root, path)
      for path in JSHINT_PROJECTS_BLACKLIST]

  def keep_whitelisted_files(affected_file):
    return any([
        affected_file.AbsoluteLocalPath().startswith(whitelisted_path) and
        all([
            not affected_file.AbsoluteLocalPath().startswith(blacklisted_path)
            for blacklisted_path in blacklisted_paths
        ])
        for whitelisted_path in whitelisted_paths
    ])
  return sorted(
      f.AbsoluteLocalPath()
      for f in input_api.AffectedFiles(
          include_deletes=include_deletes, file_filter=keep_whitelisted_files)
      if f.AbsoluteLocalPath().endswith('.js'))


def JshintChecks(input_api, output_api):  # pragma: no cover
  """Runs Jshint on all .js files under appengine/."""
  infra_root = input_api.PresubmitLocalPath()
  node_jshint_path = input_api.os_path.join(infra_root, 'node', 'jshint.py')

  tests = []
  for js_file in GetAffectedJsFiles(input_api):
    cmd = [input_api.python_executable, node_jshint_path, js_file]
    tests.append(input_api.Command(
        name='Jshint %s' % js_file,
        cmd=cmd,
        kwargs={},
        message=output_api.PresubmitError))
  return tests


# string pattern, sequence of strings to show when pattern matches,
# error flag. True if match is a presubmit error, otherwise it's a warning.
_NON_INCLUSIVE_TERMS = (
    (
        # Note that \b pattern in python re is pretty particular. In this
        # regexp, 'class WhiteList ...' will match, but 'class FooWhiteList
        # ...' will not. This may require some tweaking to catch these cases
        # without triggering a lot of false positives. Leaving it naive and
        # less matchy for now.
        r'/\b(?i)((black|white)list|master|slave)\b',  # nocheck
        (
            'Please don\'t use blacklist, whitelist, '  # nocheck
            'master, or slave in your',  # nocheck
            'code and make every effort to use other terms. Using "// nocheck"',
            'at the end of the offending line will bypass this PRESUBMIT error',
            'but avoid using this whenever possible. Reach out to',
            'community@chromium.org if you have questions'),
        True),)


def CheckInclusiveLanguage(input_api, output_api):
  """Make sure that banned non-inclusive terms are not used."""
  results = []
  results.extend(
      input_api.canned_checks.CheckInclusiveLanguage(
          input_api,
          output_api,
          excluded_directories_relative_path=[
              'infra', 'inclusive_language_presubmit_exempt_dirs.txt'
          ],
          non_inclusive_terms=_NON_INCLUSIVE_TERMS))
  return results


def GoChecks(input_api, output_api):  # pragma: no cover
  # "go.mod" tidiness test needs to run first because if go.mod is untidy,
  # the rest of Go tests may blow up in a noisy way. Running this check also
  # bootstraps the go environment, but without tools. The tools are installed
  # later in `bootstrap go env` step.
  output = GoCheckGoModTidy(input_api, output_api)
  if any(x.fatal for x in output):
    return output, []

  # Collect all other potential Go tests
  tests = GoCheckers(input_api, output_api)
  if not tests:
    return output, []

  # depot_tools runs tests in parallel. If go env is not setup, each test will
  # attempt to bootstrap it simultaneously, which doesn't currently work
  # correctly.
  #
  # Because we use RunTests here, this will run immediately. The actual go
  # tests will run after this, assuming the bootstrap is successful.
  output.extend(
      input_api.RunTests([
          input_api.Command(
              name='bootstrap go env',
              cmd=[
                  'vpython',
                  input_api.os_path.join(input_api.change.RepositoryRoot(),
                                         'go', 'bootstrap.py')
              ],
              kwargs={},
              message=output_api.PresubmitError)
      ]))
  if any(x.fatal for x in output):
    return output, []
  return output, tests


def CommonChecks(input_api, output_api):  # pragma: no cover
  output, tests = GoChecks(input_api, output_api)

  file_filter = lambda x: x.LocalPath() == 'infra/config/recipes.cfg'
  output.extend(
      input_api.canned_checks.CheckJsonParses(
          input_api, output_api, file_filter=file_filter))

  # Add non-go tests
  tests += JshintChecks(input_api, output_api)

  # Run all the collected tests
  if tests:
    output.extend(input_api.RunTests(tests))

  output.extend(BrokenLinksChecks(input_api, output_api))

  third_party_filter = lambda path: input_api.FilterSourceFile(
      path, files_to_skip=THIRD_PARTY_DIRS)
  output.extend(input_api.canned_checks.CheckGenderNeutral(
      input_api, output_api, source_file_filter=third_party_filter))
  output.extend(
      input_api.canned_checks.CheckPatchFormatted(input_api, output_api))

  return output


def CheckChangeOnUpload(input_api, output_api):  # pragma: no cover
  output = CommonChecks(input_api, output_api)
  output.extend(input_api.RunTests(
    PylintChecks(input_api, output_api, only_changed=True)))
  output.extend(NoForkCheck(input_api, output_api))
  output.extend(EmptiedFilesCheck(input_api, output_api))
  output.extend(CheckInclusiveLanguage(input_api, output_api))
  return output


def CheckChangeOnCommit(input_api, output_api):  # pragma: no cover
  output = CommonChecks(input_api, output_api)
  output.extend(input_api.RunTests(
    PylintChecks(input_api, output_api, only_changed=False)))
  return output
