import json
import os
from enum import IntEnum
from typing import List, NamedTuple, Tuple

from chromite.lib import osutils
from chromite.lib import portage_util

import lib.constants as constants
from .logger import g_logger
from .util import Setup


class PackageSupport(IntEnum):

  SUPPORTED = 0
  # Package does not have local sources and is being downloaded.
  NO_LOCAL_SOURCE = 2
  # Package is not built with gn.
  NO_GN_BUILD = 3
  # There are some temporary issues with package that should be resolved.
  TEMP_NO_SUPPORT = 4


class PackagePathException(Exception):
  """Exception indicating some troubles while looking for packages dirs."""

  def __init__(self,
               package,
               message: str,
               first_dir: str = None,
               second_dir: str = None):
    if not first_dir:
      super(PackagePathException, self).__init__(f"{package.name}: {message}")
    elif not second_dir:
      super(PackagePathException,
            self).__init__(f"{package.name}: {message}: '{first_dir}'")
    else:
      super(PackagePathException, self).__init__(
          f"{package.name}: {message}: {first_dir} vs {second_dir}")


class PackageDependency(NamedTuple):
  name: str
  types: List[str]


def _CheckEbuildVar(ebuild_file: str,
                    var: str,
                    temp_src_basedir: str = '') -> str:
  """Returns a variable's value in ebuild file."""

  env = {'CROS_WORKON_ALWAYS_LIVE': '', 'S': temp_src_basedir}
  settings = osutils.SourceEnvironment(ebuild_file, (var,),
                                       env=env,
                                       multiline=True)
  if var in settings:
    return settings[var]

  return None


def IsPackageSupported(ebuild: portage_util.EBuild,
                       setup: Setup) -> PackageSupport:
  """
  Performs checks that the package can be processed:
    * package has *-9999.ebuild file
    * package has local sources:
      * CROS_WORKON_LOCALNAME is set
      * CROS_WORKON_REPO is not set
      * EGIT_REPO_URI is not set
    * package is built with gn (CROS_WORKON_SUBTREE contains '.gn')

  Returns corresponding PackageSupport enum value.
  """

  ebuild_file = ebuild._unstable_ebuild_path
  ebuild_source_info = ebuild.GetSourceInfo(setup.src_dir, setup.manifest)

  def HasLocalSource():
    # Project is CROS_WORKON_PROJECT in ebuild file.
    # Srcdir is CROS_WORKON_LOCALNAME in ebuild file.
    # If package does not have project and srcdir - it's downloaded.
    # If package has project or srcdir being empty-project - it's downloaed.
    if not ebuild_source_info.srcdirs or not ebuild_source_info.projects:
      return False
    if ebuild_source_info.projects and len(
        ebuild_source_info.projects
    ) == 1 and ebuild_source_info.projects[0].endswith('empty-project'):
      return False
    if ebuild_source_info.srcdirs and len(
        ebuild_source_info.srcdirs
    ) == 1 and ebuild_source_info.srcdirs[0].endswith('empty-project'):
      return False

    # If package has platform2 subdir and it does not exist and there's no other
    #  src dir but platform2 - it's downloaded.
    # Downloadable examples:
    # * chromeos-base/intel-nnha: platform2 with non-existing PLATFORM_SUBDIR.
    # * chromeos-base/quipper: platform2 with non-existing PLATFORM_SUBDIR.
    # * dev-libs/marisa-aosp: platform2 with non-existing PLATFORM_SUBDIR.
    # With local source:
    # * dev-libs/libtextclassifier: not pltform2 with non-existing
    #   PLATFORM_SUBDIR.
    platform_subdir = _CheckEbuildVar(ebuild_file, 'PLATFORM_SUBDIR')
    if platform_subdir and not os.path.isdir(
        os.path.join(setup.platform2_dir, platform_subdir)):
      if not any((os.path.isdir(srcdir)
                  for srcdir in ebuild_source_info.srcdirs
                  if srcdir != setup.platform2_dir)):
        return False

    return True

  def IsBuiltWithGn():
    # Subtrees is CROS_WORKON_SUBTREE in ebuild file.
    # If none of subtrees is .gn - package is not built with gn.
    if all((not st.endswith('.gn') for st in ebuild_source_info.subtrees)):
      return False

    if _CheckEbuildVar(ebuild_file, 'CROS_RUST_SUBDIR'):
      return False

    # TODO: Returns true for config packages (should be false):
    # * chromeos-base/arc-common-scripts
    # * chromeos-base/arc-myfiles
    # * chromeos-base/arc-removable-media
    # TODO: Returns true for makefile packages (should be false):
    # * chromeos-base/avtest_label_detect

    return True

  if not HasLocalSource():
    return PackageSupport.NO_LOCAL_SOURCE

  if not IsBuiltWithGn():
    return PackageSupport.NO_GN_BUILD

  if ebuild.package in constants.TEMPORARY_UNSUPPORTED_PACKAGES:
    return PackageSupport.TEMP_NO_SUPPORT

  if setup.with_tests and ebuild.package in constants.TEMPORARY_UNSUPPORTED_PACKAGES_WITH_TESTS:
    return PackageSupport.TEMP_NO_SUPPORT

  if ebuild.package in setup.skip_packages:
    return PackageSupport.TEMP_NO_SUPPORT

  return PackageSupport.SUPPORTED


class Package:
  """"
  Represents portage package. Gives an access to paths associated with the
  package. Fields:
    * setup.
    * name: package's full name, e.g. chromeos-base/cryptohome.
    * simple_name: last part of the name, e.g. cryptohome.
    * is_highly_volatile: bool indicating if package's sources are patched on
      build. If true, one should not expect exact match between temp and
      actual sources.
    * temp_dir: base path to a dir with all temporary sources.
    * build_dir: path to a dir with build. Is expected to contain args.gn.
    * src_dir_matches: list of tuples (temp, actual). Represents a possbile
      match between temporary and actual source dirs/files. The list is sorted
      by depth: match is better when closer to desired path.
    * additional_include_paths: list of actual paths to be added to include
      path arguments.
    * dependencies: list of package names on which this package depends on.

  Raises:
    * UnsupportedPackageException upon construction if package is not supported.

  NOTE: all dir fields are expected to exist when Initialize is called.
  NOTE: only packages built with gn are supported.
  """

  class UnsupportedPackageException(Exception):
    """Exception indicating attempt to create unsupported package."""

    def __init__(self, package_name, reason: PackageSupport):
      self.package_name = package_name
      self.reason = reason
      super(Package.UnsupportedPackageException,
            self).__init__(f"{package_name}: Not supported due to: {reason}")

  class DirsException(PackagePathException):
    """Exception indicating some troubles while looking for packages dirs."""

  class TempActualDichotomy(NamedTuple):
    temp: str
    actual: str

  @staticmethod
  def Serialize(package: 'Package') -> str:
    package_as_dict = {
        'ebuild': package.ebuild_file,
        'deps': [dep._asdict() for dep in package.dependencies]
    }
    return json.dumps(package_as_dict)

  @staticmethod
  def Deserialize(data: bytes, setup: Setup) -> 'Package':
    package_as_dict = json.loads(data)
    assert 'ebuild' in package_as_dict
    assert 'deps' in package_as_dict

    ebuild = portage_util.EBuild(package_as_dict['ebuild'])
    deps = [PackageDependency(**dep) for dep in package_as_dict['deps']]
    return Package(setup, ebuild, deps)

  g_highly_volatile_packages = [
      # Libchrome has a number of patches applied on top of checkout.
      'chromeos-base/libchrome'
  ]

  def __init__(self,
               setup: Setup,
               ebuild: portage_util.EBuild,
               deps: List[PackageDependency] = []):

    is_supported = IsPackageSupported(ebuild, setup)
    if is_supported != PackageSupport.SUPPORTED:
      raise Package.UnsupportedPackageException(ebuild.package, is_supported)

    self.setup = setup
    self.name = ebuild.package
    self.simple_name = ebuild.pkgname
    self.ebuild_file = ebuild._unstable_ebuild_path

    self.is_highly_volatile = os.path.isdir(
        os.path.join(
            os.path.dirname(self.ebuild_file),
            'files')) or self.name in Package.g_highly_volatile_packages
    self.dependencies = deps if deps else []

  def __eq__(self, other) -> bool:
    if isinstance(other, str):
      return self.name == other
    elif isinstance(other, Package):
      return self.name == other.name

    raise NotImplementedError('Can comparte only with Package or string')

  def Initialize(self) -> None:
    """
    Find directories associated with the package and check they exist.

    This method will fail not-yet-built package, so make sure you've built
    the package with FEATURES=noclean flag.

    Raises:
      * DirsException if build, source or temp source dir(s) is not found.
    """
    g_logger.debug('%s: Initializing', self.name)

    self.temp_dir = self._GetTempDir()
    g_logger.debug('%s: Temp dir: %s', self.name, self.temp_dir)

    self.build_dir = self._GetBuildDir()
    g_logger.debug('%s: Build dir: %s', self.name, self.build_dir)

    src_dirs, temp_src_dirs = self._GetSourceDirs()

    if not src_dirs:
      raise Package.DirsException(self, 'Cannot find any src dirs')

    for src_dir in src_dirs:
      if not os.path.isdir(src_dir):
        raise Package.DirsException(self, 'Cannot find src dir', src_dir)

    if not temp_src_dirs:
      raise Package.DirsException(self, 'Cannot find any temps src dirs')

    for temp_src_dir in temp_src_dirs:
      if not os.path.isdir(temp_src_dir):
        raise Package.DirsException(self, 'Cannot find temp src dir',
                                    temp_src_dir)

    if len(src_dirs) != len(temp_src_dirs):
      raise Package.DirsException(self,
                                  'Different number of src and temp src dirs')

    self.src_dir_matches: List[Package.TempActualDichotomy] = []
    for src_dir, temp_src_dir in zip(src_dirs, temp_src_dirs):
      self.src_dir_matches.append(
          Package.TempActualDichotomy(temp=temp_src_dir, actual=src_dir))
      g_logger.debug('%s: Match between temp and actual: %s and %s', self.name,
                     temp_src_dir, src_dir)

    self.additional_include_paths = self.GetAdditionalIncludePaths()
    if self.additional_include_paths:
      for path in self.additional_include_paths:
        if not os.path.isdir(path):
          raise Package.DirsException(self,
                                      'Additional include path does not exist',
                                      path)

  def GetAdditionalIncludePaths(self) -> List[str]:
    """Returns a list of actual paths to be added as include path arguments."""

    # Special case for chromeos-base/update_engine which pretends to be in
    # platform2 and uses platform2 as include path. While the actual include
    # path is {src_dir}/aosp/system with update_engine inside.
    if self.name == 'chromeos-base/update_engine':
      return [os.path.dirname(self.src_dir_matches[0].actual)]

    return None

  def _IsOutOfTreeBuild(self) -> bool:
    """
    True if package has CROS_WORKON_OUTOFTREE_BUILD.

    If true, then package is built from local sources and has nothing in
    temp source dir (which does not exist).
    """
    out_of_tree_build = _CheckEbuildVar(self.ebuild_file,
                                        'CROS_WORKON_OUTOFTREE_BUILD')
    return out_of_tree_build and out_of_tree_build == '1'

  def _GetTempDir(self) -> str:
    """Returns path to base temp dir."""

    dir = os.path.join(self.setup.board_dir, 'tmp', 'portage',
                       self.name + '-9999', 'work')

    if not os.path.isdir(dir):
      raise Package.DirsException(self, 'Cannot find temp dir', dir)

    return dir

  def _GetBuildDir(self) -> str:
    """
    Returns path to dir with build metadata (were args.gn lives).

    Raises:
      * DirsException if build dir is not found.
      * DirsException if 'args.gn' file not found in supposed build dir.
    """
    build_dirs = [
        os.path.join(self.setup.board_dir, 'var', 'cache', 'portage', self.name,
                     'out', 'Default'),
        os.path.join(self.temp_dir, 'build', 'out', 'Default')
    ]

    build_dir = None
    for dir in build_dirs:
      if os.path.isdir(dir):
        build_dir = dir
        break

    if not build_dir:
      raise Package.DirsException(self, 'Cannot find build dir')

    if not os.path.isfile(os.path.join(build_dir, 'args.gn')):
      raise Package.DirsException(self, 'Build dir does not contain args.gn',
                                  build_dir)

    return build_dir

  def _GetSourceDirs(self) -> Tuple[str, str]:
    """
    Returns list of matches between actual src dirs and temp src dirs. Matches
    listed from deepest to the most common.
    """
    ebuild = portage_util.EBuild(self.ebuild_file)
    ebuild_src_dirs = ebuild.GetSourceInfo(self.setup.src_dir,
                                           self.setup.manifest).srcdirs

    if self._IsOutOfTreeBuild():
      src_dirs = ebuild_src_dirs
      temp_src_dirs = ebuild_src_dirs
    else:
      src_dirs = ebuild_src_dirs

      temp_src_basedir = os.path.join(self.temp_dir, self.simple_name + '-9999')
      temp_src_dirs = []

      dest_dirs = _CheckEbuildVar(self.ebuild_file, 'CROS_WORKON_DESTDIR',
                                  temp_src_basedir)
      if dest_dirs:
        dest_dirs = dest_dirs.split(',')
        temp_src_dirs.extend(dest_dirs)

    src_dirs.sort(key=len, reverse=True)
    temp_src_dirs.sort(key=len, reverse=True)

    return src_dirs, temp_src_dirs
