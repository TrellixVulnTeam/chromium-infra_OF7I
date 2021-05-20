# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import re


# The epoch is prepended to the version when constructing the version: tag
# for both the source package and the final built package. It must be
# incremented in the case of incompatible changes to the source package
# format, or any time that we need to force new builds of all packages.
PACKAGE_EPOCH = '2'


def parse_name_version(name_version):
  """Parses a package 'name', or 'name@version'.

  Returns (name, version). If the input was just 'name' then the version is
  'latest'.
  """
  if '@' in name_version:
    name, version = name_version.split('@')
  else:
    name, version = name_version, 'latest'
  return name, version


def platform_for_host(api):
  """This returns a cipd platform name for the current host, derived from the
  `platform` recipe_module.
  """
  return '%s-%s' % (
      {
          'win': 'windows',
          'linux': 'linux',  # not actually used, but for completeness
          'mac': 'mac',
      }[api.platform.name],
      {
          ('intel', 32): '386',
          ('intel', 64): 'amd64',
          ('arm', 64): 'arm64',
      }[api.platform.arch, api.platform.bits])


def tool_platform(api, platform, _spec_pb):
  """Returns the target platform for tools needed to build the provided
  `platform`. E.g. if we're targeting `linux-arm64` the toolchain might be
  `linux-amd64`, regardless of the host platform (because we use docker to build
  for linux-arm64, and so the tools need to run in the docker container).

  When not cross-compiling, this returns a cipd platform name for the current
  host, derived from the `platform` recipe_module.
  """
  if platform.startswith('linux-'):
    # TODO(iannucci): When we can control the toolchains more precisely in
    # `spec_pb`, make this contingent on the selection of dockcross. Until
    # then, hardcode the dockcross host type.
    return 'linux-amd64'
  return platform_for_host(api)


def get_cipd_pkg_name(pkg_pieces):
  """Return the CIPD package name with given list of package pieces.

  Pieces identified as False (like None, or empty string) will be skipped.
  This is to make sure the returned cipd pacakge name
    * Does not have leading & tailing '/'
    * Does not have string like '//' inside
  """
  return '/'.join([str(piece) for piece in pkg_pieces if piece])


class ResolvedSpec(object):
  """The ResolvedSpec represents a version of the Spec protobuf message, but
  resolved for a single target platform (e.g. "windows-amd64").

  It has helper methods and properties to read the resolved data.
  """
  def __init__(self, api, cipd_spec_pool, package_prefix, source_cache_prefix,
               cipd_pkg_name, platform, pkg_dir, spec, deps, unpinned_tools):
    self._api = api
    self._package_prefix = package_prefix.strip('/')
    self._source_cache_prefix = source_cache_prefix.strip('/')
    self._cipd_spec_pool = cipd_spec_pool

    # CIPD package name, excluding the package_prefix and platform suffix
    self._cipd_pkg_name = cipd_pkg_name
    self._platform = platform             # Platform resolved for
    # Path to the directory containing the package definition file
    self._pkg_dir = pkg_dir
    self._spec_pb = spec                  # spec_pb2.Spec
    self._deps = deps                     # list[ResolvedSpec]
    self._unpinned_tools = unpinned_tools # list[ResolvedSpec]

    self._all_deps_and_tools = set()
    for d in self._deps:
      self._all_deps_and_tools.add(d)
      self._all_deps_and_tools.update(d.all_possible_deps_and_tools)
    for d in self._unpinned_tools:
      self._all_deps_and_tools.add(d)
      self._all_deps_and_tools.update(d.all_possible_deps_and_tools)

  @property
  def cipd_pkg_name(self):
    """CIPD package name, excluding the package_prefix and platform suffix."""
    return self._cipd_pkg_name

  @property
  def source_cache(self):
    """Source cache location to fetch/upload in CIPD server.

    For git method, source cache will be independent of platform, however due to
    fine grained fetch method for script method, cache will be platform
    dependent. e.g. <package_prefix>/<source_cache_prefix>/git/repo_url or
    <package_prefix>/<source_cache_prefix>/script/pkg/platform.
    """
    method, source_method_pb = self.source_method
    if method == 'git':
      repo_url = source_method_pb.repo
      if repo_url.startswith('https://chromium.googlesource.com/external/'):
        repo_url = repo_url[len('https://chromium.googlesource.com/external/'):]
      elif repo_url.startswith('https://'):
        repo_url = repo_url[len('https://'):]
      elif repo_url.startswith('http://'):  # pragma: no cover
        repo_url = repo_url[len('http://'):]
      _source_cache = get_cipd_pkg_name([
          self._package_prefix, self._source_cache_prefix, method,
          repo_url.lower()])
    elif method == 'script' or method == 'url':
      _source_cache = get_cipd_pkg_name([
          self._package_prefix, self._source_cache_prefix, method,
          self._cipd_pkg_name_with_override, self._platform])
    else:
      _source_cache = None  # pragma: no cover
    return _source_cache

  @property
  def tool_platform(self):
    """The CIPD platform name for tools to build this ResolvedSpec.

    USUALLY, this is equivalent to the host platform (the machine running the
    recipe), but in the case of cross-compilation for linux, this will be the
    platform of the cross-compile docker container (i.e. 'linux-amd64').

    This is used to build programs that are used during the compilation of the
    package (e.g. `cmake`, `ninja`, etc.).
    """
    return tool_platform(self._api, self._platform, self._spec_pb)

  @staticmethod
  def _assert_resolve_for(condition):
    assert condition, 'Impossible; _resolve_for should have caught this'

  @property
  def create_pb(self):
    """The singular `spec_pb2.Spec.Create` message."""
    self._assert_resolve_for(len(self._spec_pb.create) == 1)
    return self._spec_pb.create[0]

  @property
  def source_method(self):
    """A tuple of (source_method_name, source_method_pb).

    These are the result of parsing the `method` field of the Spec.Create.Source
    message.
    """
    pb = self.create_pb.source
    method = pb.WhichOneof("method")
    self._assert_resolve_for(method is not None)
    self._assert_resolve_for(method in ('git', 'cipd', 'script', 'url'))
    return method, getattr(pb, method)

  @property
  def pkg_dir(self):
    """Path to the folder containing this spec on the host (i.e. not the
    version copied into the checkout)."""
    return self._pkg_dir

  @property
  def platform(self):
    """The CIPD platform that this ResolvedSpec was resolved for."""
    return self._platform

  @property
  def unpinned_tools(self):
    """The list of unpinned_tools as ResolvedSpec's.

    These packages must exist in order to build this ResolvedSpec, and may be
    implicitly built during the building of this ResolvedSpec.
    """
    return self._unpinned_tools

  @property
  def pinned_tool_info(self):
    """A generator of (package_name, version) for tools which this ResolvedSpec
    depends on, but which MUST ALREADY EXIST on the CIPD server.

    These will not be built implicitly when building this ResolvedSpec.
    """
    for t in self.create_pb.build.tool:
      name, version = parse_name_version(t)
      if version != 'latest':
        yield name, version

  @property
  def all_possible_deps(self):
    """Yields all the packages (ResolvedSpecs) that this ResolvedSpec depends
    on, which includes `deps` and their transitive dependencies.

    Infinite recursion is prevented by the _resolve_for function (which
    constructs all of the ResolvedSpec instances).
    """
    for dep in self._deps:
      for subdep in dep.all_possible_deps:
        yield subdep
      yield dep

  @property
  def all_possible_deps_and_tools(self):
    """Returns a set of all the packages (ResolvedSpecs) that this ResolvedSpec
    depends on, which includes both `deps` and `tools`, transitively.

    Infinite recursion is prevented by the _resolve_for function (which
    constructs all of the ResolvedSpec instances).
    """
    return self._all_deps_and_tools

  @property
  def disable_latest_ref(self):
    return self.create_pb.package.disable_latest_ref

  def cipd_spec(self, version):
    """Returns a CIPDSpec object for the result of building this ResolvedSpec's
    package/platform/version.

    Args:
      * version (str) - The symver of this package to get the CIPDSpec for.
    """
    cipd_pieces = [self._package_prefix, self._cipd_pkg_name_with_override]
    if not self._spec_pb.upload.universal:
      cipd_pieces.append(self.platform)
    full_cipd_pkg_name = get_cipd_pkg_name(cipd_pieces)
    patch_ver = self.create_pb.source.patch_version

    if self.create_pb.package.alter_version_re:
      version = re.sub(
          self.create_pb.package.alter_version_re,
          self.create_pb.package.alter_version_replace,
          version)
    symver = '%s@%s%s' % (PACKAGE_EPOCH, version,
                          '.' + patch_ver if patch_ver else '')
    return self._cipd_spec_pool.get(full_cipd_pkg_name, symver)

  def source_cipd_spec(self, version):
    """Returns a CIPDSpec object for the result of building this ResolvedSpec's
    source package.

    Args:
      * version (str) - The symver of this package to get the CIPDSpec for.
    """
    method, source_method_pb = self.source_method
    if method == 'cipd':
      pkg_name = source_method_pb.pkg
      # For the cipd source, we should use the given version as it is since:
      # * cipd api relies on this to deploy the package.
      # * cipd source will not be cached again in cipd.
      source_version = version
    else:
      pkg_name = self.source_cache
      source_version = '%s@%s' % (PACKAGE_EPOCH, version)

    return self._cipd_spec_pool.get(pkg_name,
                                    source_version) if pkg_name else None

  @property
  def _cipd_pkg_name_with_override(self):
    """Computes the true CIPD package name using any supplied override."""
    if not self._spec_pb.upload.pkg_name_override:
      return self._cipd_pkg_name
    return '/'.join(self._cipd_pkg_name.split('/')[:-1] +
                    [self._spec_pb.upload.pkg_name_override])

  @property
  def _sort_tuple(self):
    """Implementation detail of __cmp__, returns a sortable tuple that's
    used as a last resort when sorting by dependencies fails."""
    return (
      len(self.all_possible_deps_and_tools),
      self.cipd_pkg_name,
      self.platform,
      id(self),
    )

  def __cmp__(self, other):
    """This allows ResolvedSpec's to be sorted.

    ResolvedSpec's which depend on another ResolvedSpec will sort after it.
    """
    if self is other: # pragma: no cover
      return 0

    # self < other if other depends on self, OR
    #              if other uses self as a tool
    if self in other.all_possible_deps_and_tools:
      return -1

    # self > other if self depends on other, OR
    #              if self uses other as a tool
    if other in self.all_possible_deps_and_tools:
      return 1

    # Otherwise sort by #deps and package name.
    return cmp(self._sort_tuple, other._sort_tuple)
