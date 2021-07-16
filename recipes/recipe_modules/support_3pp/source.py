# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Implements the source version checking and acquisition logic."""

import functools
import itertools
import operator
import re

from pkg_resources import parse_version

from .run_script import run_script

from PB.recipe_modules.infra.support_3pp.spec import LT, LE, GT, GE, EQ, NE


def _to_versions(raw_ls_remote_lines, version_join, tag_re):
  """Converts raw ls-remote output lines to a sorted (descending)
  list of (Version, v_str, git_hash) objects.

  This is used for source:git method to find latest version and git hash.
  """
  ret = []
  for line in raw_ls_remote_lines:
    git_hash, ref = line.split('\t')
    if ref.startswith('refs/tags/'):
      tag = ref[len('refs/tags/'):]
      m = tag_re.match(tag)
      if not m:
        continue

      v_str = m.group(1)
      if version_join:
        v_str = '.'.join(v_str.split(version_join))

      ret.append((parse_version(v_str), v_str, git_hash))
  return sorted(ret, reverse=True)


# Maps the operator OP(a, b) to the reverse function. For example:
#
#    A < B   maps to   B > A
#
# This will allow us to use functools.partial to pre-fill the value of B.
FILTER_TO_REVERSE_OP = {
    LT: operator.gt,
    LE: operator.ge,
    GT: operator.lt,
    GE: operator.le,

    # EQ and NE are commutative comparisons, so they map directly to their
    # equivalent function in `operator`.
    EQ: operator.eq,
    NE: operator.ne,
}

def _filters_to_func(filters):
  restrictions = [
    functools.partial(FILTER_TO_REVERSE_OP[f.op], parse_version(f.val))
    for f in filters
  ]
  def _apply_filter(candidate_version):
    for restriction in restrictions:
      if not restriction(candidate_version):
        return False
    return True
  return _apply_filter


def _filter_versions(version_strs, filters):
  if not filters:
    return version_strs
  filt_fn = _filters_to_func(filters)
  return [
    (vers, vers_s, git_hash)
    for vers, vers_s, git_hash in version_strs
    if filt_fn(vers)
  ]


def resolve_latest(api, spec):
  """Resolves the latest available version given a ResolvedSpec.

  This usually involves doing network operations, depending on the `source`
  type of the ResolvedSpec.

  Args:
    * api - The module injection site for support_3pp recipe module
      (i.e. `self.m`)
    * spec (ResolvedSpec) - The spec to resolve.

  Returns (str, str) the symver for the latest version of this package, e.g.
  '1.2.3'. This should always use '.' as the digit separator. And checksum for
  resolved source, e.g. git_hash for source:git method.
  """
  # TODO(iannucci): when we can put annotations on nest steps, put the 'resolved
  # version' there.

  method_name, source_method_pb = spec.source_method
  source_hash = ''
  if method_name == 'git':
    # We need to transform the tag_pattern (which is a python format-string
    # lookalike with `%s` in it) into a regex which we can use to scan over the
    # repo's tags.
    tag_re = re.escape(
      source_method_pb.tag_pattern if source_method_pb.tag_pattern else '%s')
    tag_re = '^%s$' % (tag_re.replace('\\%s', '(.*)'),)

    step = api.git('ls-remote', '-t', source_method_pb.repo,
                   stdout=api.raw_io.output(),
                   step_test_data=lambda: api.raw_io.test_api.stream_output(
                     '\n'.join([
                       'hash\trefs/tags/unrelated',
                       'hash\trefs/tags/v1.0.0-a2',
                       'hash\trefs/tags/v1.3.0',
                       'hash\trefs/tags/v1.4.0',
                       'hash\trefs/tags/v1.4.1',
                       'hash\trefs/tags/v1.5.0-rc1',
                     ])))

    versions = _to_versions(
        step.stdout.splitlines(),
        source_method_pb.version_join,
        re.compile(tag_re))

    versions = _filter_versions(
        versions, source_method_pb.version_restriction)

    highest_cmp = parse_version('0')
    highest_str = ''
    git_tree_hash = ''
    for vers, v_str, git_hash in versions:
      if vers > highest_cmp:
        highest_cmp = vers
        highest_str = v_str
        git_tree_hash = git_hash

    assert highest_str
    version = highest_str

    source_hash = git_tree_hash
    api.step.active_result.presentation.step_text = (
      'resolved version: %s' % (version,))

  # TODO(akashmukherjee): Get/compute hash for script method.
  elif method_name == 'script':
    script = spec.pkg_dir.join(source_method_pb.name[0])
    args = list(map(str, source_method_pb.name[1:])) + ['latest']
    version = run_script(api,
      script, *args,
      stdout=api.raw_io.output(),
      step_test_data=lambda: api.raw_io.test_api.stream_output('2.0.0')
    ).stdout.strip()
    api.step.active_result.presentation.step_text = (
      'resolved version: %s' % (version,))

  elif method_name == 'cipd':
    version = source_method_pb.default_version
    # We don't actually run a real step here, so we can't put the 'resolved
    # version' anywhere :(. See TODO at top.

    # If "latest", try to resolve it from the tag "version" of the cipd
    # instance that has the ref "latest".
    if version == 'latest':
      desc = api.cipd.describe(source_method_pb.pkg, 'latest')
      for tag in desc.tags:
        if tag.tag.startswith('version:'):
          version = tag.tag[len('version:'):]
      if version == 'latest':
        raise AssertionError(
            'Failed to resolve the latest version for CIPD package %s' %
            source_method_pb.pkg)

  elif method_name == 'url':
    version = source_method_pb.version

  else: # pragma: no cover
    assert False, '"latest" version resolution not defined for %r' % method_name

  return version, source_hash


def fetch_source(api,
                 workdir,
                 spec,
                 version,
                 source_hash,
                 spec_lookup,
                 ensure_built,
                 skip_upload=False):
  """Prepares a checkout in `workdir` to build `spec` at `version`.

  Args:
    * api - The module injection site for support_3pp recipe module
      (i.e. `self.m`)
    * workdir (Workdir) - The working directory object we're going to build the
      spec in. This function will create the checkout in `workdir.checkout`.
    * spec (ResolvedSpec) - The package we want to build.
    * version (str) - The symver of the package we want to build (e.g. '1.2.0').
    * source_hash (str) - source_hash returned from resolved version. This is
      external hash of the source.
    * spec_lookup ((package_name, platform) -> ResolvedSpec) - A function to
      lookup (possibly cached) ResolvedSpec's for things like dependencies and
      tools.
    * ensure_built ((ResolvedSpec, version) -> CIPDSpec) - A function to ensure
      that a given ResolvedSpec is actually fully built and return a CIPDSpec to
      retrieve it's output package.
    * skip_upload (bool) - When True, skip uploading the source to CIPD.
      Default to False.
  """
  def _ensure_installed(root, cipd_pkgs):
    # TODO(iannucci): once `cipd ensure` supports local package installation,
    # use that.
    for pkg in cipd_pkgs:
      pkg.deploy(root)

  if spec.create_pb.build.tool:
    with api.step.nest('installing tools'):
      # ensure all our dependencies are built (should be handled by
      # ensure_uploaded, but just in case).
      _ensure_installed(workdir.tools_prefix, [
        ensure_built(tool, 'latest')
        for tool in spec.unpinned_tools
      ] + [
        spec_lookup(tool, spec.tool_platform).cipd_spec(tool_version)
        for tool, tool_version in spec.pinned_tool_info
      ])

  if spec.create_pb.build.dep:
    with api.step.nest('installing deps'):
      _ensure_installed(workdir.deps_prefix, [
        ensure_built(dep, 'latest')
        for dep in spec.all_possible_deps
      ])

  # Building a CIPDSpec for source package at resolved version
  source_cipd_spec = spec.source_cipd_spec(version)
  if source_hash:
    # See if source is already cached in cipd
    try:
      all_tags = source_cipd_spec.resolve_remote_tags()
    # When source cache miss happens, set step status SUCCESS
    except api.step.StepFailure:  # pragma: no cover
      api.step.active_result.presentation.status = api.step.SUCCESS
    else:
      external_hash = all_tags.get('external_hash', None)
      step_hash = api.step('Verify External Hash', None)
      if external_hash != source_hash:
        step_hash.presentation.status = 'FAILURE'
        step_hash.presentation.step_text = (
          'resolved version: %s has moved, current hash: %s, stored hash: %s, '
          'please verify.' % (version, source_hash, external_hash))
        raise AssertionError(
            'External hash verification failed, please check the third party'
            ' git repository for any security incidents.'
        )
      else:  # pragma: no cover
        step_hash.presentation.step_text = (
          'external source verification successful.')

  _source_checkout(
      api,
      workdir,
      spec,
      source_cipd_spec,
      skip_upload,
      source_hash=source_hash)

  # Iff we are going to do the 'build' operation, copy all the necessary package
  # definition scripts into the checkout. If no build message is provided,
  # then we're planning to directly package the result of the checkout, and
  # we don't want to include these scripts.
  if spec.create_pb.HasField("build"):
    # Copy all the necessary package definitions into the checkout
    # We want to uniquify by package name, even if the same package is
    # present for more than one platform (in the case of cross-compiling).
    unique_pkgs = {
        s.cipd_pkg_name: s
        for s in itertools.chain([spec], spec.all_possible_deps_and_tools)
    }
    # Sort to make sure the packages are in order, otherwise it will produce
    # inconsistent recipe train result.
    for pkg in sorted(unique_pkgs.values()):
      api.file.copytree(
        'copy package definition %s' % pkg.cipd_pkg_name,
        pkg.pkg_dir,
        workdir.script_dir(pkg))


# TODO(akashmukherjee): Reconstruct the manifest object to beautify.
class Manifest(object):
  """Implements a manifest object used for downloading remote source.

  Attributes:
    * protocol (required) - Protocol to use for downloading the artifact.
    * source_uri (required) - Remote artifact resource location(s) URL/URI.
        source_uri is a list of str.
    * path - Dictates where to download the sources to, e.g. checkout dir.
    * protocol_args (list(str)) - A list of str to be passed to protocol.
    * source_hash (str) - source_hash returned from resolved version. This is
      external hash of the GitSource.
    * ext - Extension used for downloading sources.
    * artifact_names - Corresponding to source_uri, optional name for downloaded
      source. It's a list of str of equal length to source_uri. This is
      optional, if passed will be used. For `pip_bootstrap`, name of the python
      wheel is important.
  """

  def __init__(self,
               protocol,
               source_uri,
               path,
               protocol_args=None,
               source_hash=None,
               ext=None,
               artifact_names=None):
    self.protocol = protocol
    self.source_uri = source_uri
    self.path = path
    self.protocol_args = protocol_args or []
    self.source_hash = source_hash
    self.ext = ext
    self.artifact_names = artifact_names


#### Private stuff


def _generate_download_manifest(api, spec, checkout_dir,
                                source_hash=None):
  """Generates download manifest object for current 3pp package.

  Args:
    * api - The module injection site for support_3pp recipe module
      (i.e. `self.m`)
    * spec (ResolvedSpec) - The package we want to build.
    * checkout_dir (Workdir) - The destination directory for remote sources.
    * source_hash - Optional source hash, used for git method.

  Returns a manifest class object with required attributes set for protocols.
  """
  method_name, source_method_pb = spec.source_method

  if method_name == 'git':
    return Manifest(
        'git', source_method_pb.repo, checkout_dir, source_hash=source_hash)

  elif method_name == 'url':
    return Manifest('url', [source_method_pb.download_url],
                    checkout_dir, ext=source_method_pb.extension or '.tar.gz')

  elif method_name == 'script':
    script = spec.pkg_dir.join(source_method_pb.name[0])
    if source_method_pb.use_fetch_checkout_workflow:
      # version is already in env as $_3PP_VERSION
      script_args = list(map(str, source_method_pb.name[1:])) + ['checkout']
      return Manifest('script', script, checkout_dir, protocol_args=script_args)
    else:
      # version is already in env as $_3PP_VERSION
      args = list(map(str, source_method_pb.name[1:])) + ['get_url']
      result = run_script(
          api,
          script,
          *args,
          stdout=api.json.output(),
          step_test_data=lambda: api.json.test_api.output_stream({
              'url': ['https://some.internet.example.com/%s' % (
                  spec.cipd_pkg_name,)],
              'ext': '.test',
              'name': ['test_source']
          }))
      source_uri, ext = result.stdout['url'], result.stdout['ext']
      # Setting source artifact name is optional, used by `pip_bootstrap`.
      artifact_names = result.stdout.get('name')
      # Verify source_uri and artifact_names are equal length, if present.
      if artifact_names:
        assert len(source_uri) == len(
            artifact_names
        ), 'Number of download URLs should be equal to number of artifacts.'
      return Manifest('url', source_uri, checkout_dir,
                      ext=ext, artifact_names=artifact_names)

  else:  # pragma: no cover
    assert False, 'Unknown source type %r' % (method_name,)


def _download_source(api, download_manifest):
  """Fetches the raw source from the given remote location.

  Args:
    * api - The module injection site for support_3pp recipe module
      (i.e. `self.m`)
    * download_manifest - A manifest object used for downloading remote source.
  """
  # Checkout a raw git source given remote git hash.
  if download_manifest.protocol == 'git':
    api.git.checkout(download_manifest.source_uri,
                     download_manifest.source_hash, download_manifest.path)
  elif download_manifest.protocol == 'url':
    for i, uri in enumerate(download_manifest.source_uri):
      if not download_manifest.artifact_names:
        artifact = 'raw_source_' + str(i) + download_manifest.ext
      else:  # pragma: no cover
        artifact = download_manifest.artifact_names[i]
      api.url.get_file(uri, api.path.join(download_manifest.path, artifact))
  elif download_manifest.protocol == 'script':
    script = download_manifest.source_uri
    args = download_manifest.protocol_args + [download_manifest.path]
    run_script(api, script, *args)
  else:  # pragma: no cover
    assert False, 'Unknown download protocol  %r' % (protocol,)


def _source_upload(api, source_cipd_spec, external_hash=None):
  """Uploads the copy of the source package we have on the local machine to the
  CIPD server.

  This method will upload source files into CIPD with
  `<package_prefix>/<source_cache_prefix>` prefix for provided version.
  Essentially, uploading the source only for the first time. In later builds,
  3pp recipe will use these uploaded artifacts when building the same version
  of a package.

  Args:
    * api - The module injection site for support_3pp recipe module
      (i.e. `self.m`)
    * source_cipd_spec (spec) - CIPDSpec obj for source.
    * external_hash - Tag the output package with this hash.
  """
  with api.step.nest('upload source to cipd') as upload_step:
    try:
      extra_tags = {'external_hash': external_hash} if external_hash else {}
      source_cipd_spec.ensure_uploaded(extra_tags=extra_tags)
    except api.step.StepFailure:  # pragma: no cover
      upload_step.status = api.step.FAILURE
      upload_step.step_text = 'Source upload failed.'
      raise


def _source_checkout(api,
                     workdir,
                     spec,
                     source_cipd_spec,
                     skip_upload,
                     source_hash=''):
  """Checks out source packages into checkout_dir.

  This method makes sure sources used in the current build are made available
  inside checkout_dir. Source checkout can done in two ways, if source is
  already cached (url, script, git), it will download the source cache cipd
  package, also true for cipd source. If not cached, downloader workflow is
  triggered which downloads and builds a source package for future use.

  Args:
    * api - The module injection site for support_3pp recipe module
      (i.e. `self.m`)
    * workdir (Workdir) - The working directory object we're going to build the
      spec in. This function will create the checkout in `workdir.checkout`.
    * spec (ResolvedSpec) - The package we want to build.
    * source_cipd_spec (spec) - CIPDSpec obj for source.
    * skip_upload (bool) - When True, skip uploading the source to CIPD.
    * source_hash (str) - source_hash returned from resolved version. This is
      external hash of the source.
  """
  method_name = spec.source_method[0]
  source_pb = spec.create_pb.source

  checkout_dir = workdir.checkout
  # Run checkout in this subdirectory of the install script's $CWD.
  if source_pb.subdir:
    checkout_dir = checkout_dir.join(*(source_pb.subdir.split('/')))

  api.file.ensure_directory(
      'mkdir -p [workdir]/checkout/%s' % (str(source_pb.subdir),), checkout_dir)

  if (method_name != 'cipd' and source_cipd_spec and
      not source_cipd_spec.check()):
    # If source is not cached already, downloads, builds and uploads source.
    download_manifest = _generate_download_manifest(api, spec, checkout_dir,
                                                    source_hash)
    _download_source(api, download_manifest)

    # building the source CIPDSpec into a source type package locally.
    source_cipd_spec.build(
        root=checkout_dir,
        install_mode='copy',
        # Some build systems overwrite files, make sure they're writable.
        preserve_writable=True,
        version_file=None,
        exclusions=['\.git'] if method_name == 'git' else [])

    if not skip_upload:
      _source_upload(api, source_cipd_spec, source_hash)

  # Fetches source from cipd (or existing local package) for all types.
  else:
    source_cipd_spec.deploy(checkout_dir)

  if source_pb.unpack_archive:
    with api.step.nest('unpack_archive'):
      paths = api.file.glob_paths('find archive to unpack', checkout_dir, '*.*')
      assert len(paths) == 1, (
          'unpack_archive==true - expected single archive file, '
          'but %s are extracted' % (paths,))

      archive = paths[0]
      archive_name = archive.pieces[-1]
      api.step.active_result.presentation.step_text = ('found %r' %
                                                       (archive_name,))

      tmpdir = api.path.mkdtemp()
      # Use copy instead of move because archive might be a symlink (e.g. when
      # using a "cipd" source mode).
      #
      # TODO(iannucci): Have a way for `cipd pkg-deploy` to always deploy in
      # copy mode and change this to a move.
      api.file.copy('cp %r [tmpdir]' % archive_name, archive,
                    tmpdir.join(archive_name))

      # blow away any other files (e.g. .git)
      api.file.rmtree('rm -rf [checkout_dir]', checkout_dir)

      api.archive.extract('extracting [tmpdir]/%s' % archive_name,
                          tmpdir.join(archive_name), checkout_dir)

      if not source_pb.no_archive_prune:
        api.file.flatten_single_directories('prune archive subdirs',
                                            checkout_dir)

  if source_pb.patch_dir:
    patches = []
    for patch_dir in source_pb.patch_dir:
      patch_dir = str(patch_dir)
      patches.extend(
          api.file.glob_paths('find patches in %s' % patch_dir,
                              spec.pkg_dir.join(*(patch_dir.split('/'))), '*'))
    with api.context(cwd=checkout_dir):
      api.git('apply', '-v', *patches)
