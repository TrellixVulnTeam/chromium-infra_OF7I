import filecmp
import json
import os
from typing import Dict, List

from .logger import g_logger
from .package import PackagePathException
from .package import Package
from .path_handler import PathHandler
from .util import CrosSdk
from .util import Setup


class GnTargets:
  """Responsible for fixing targets."""

  class TargetPathException(PackagePathException):
    """Exception to indicate failure while resolving paths for target."""

  # Extensions of files that most likely generated and not exist.
  g_ignorable_extensions = ['.typemap']

  def __init__(self,
               data: Dict,
               package: Package,
               setup: Setup,
               *,
               result_build_dir: str = None,
               file_conflicts: Dict = {}):
    """
    Cdb constructor.

    Arguments:
      * data: loaded of gn_targets.json for |package|.
      * package: package to work with.
      * setup: setup data (board, dirs, etc).
      * result_build_dir: path to result build dir simulating single result
        package.
      * file_conflicts: maps original build artifacts in chroot dir that
        conflict between packages to corresponding artfacts in
        |result_build_dir|.
    """
    self.data = data
    self.package = package
    self.setup = setup
    self.fields_to_resolve = {
        'args': GnTargets._FixArgsField,
        'cflags': GnTargets._FixArgList,
        'cflags_c': GnTargets._FixArgList,
        'cflags_cc': GnTargets._FixArgList,
        'include_dirs': GnTargets._FixPathList,
        'inputs': GnTargets._FixInputsField,
        'ldflags': GnTargets._FixArgList,
        'lib_dirs': GnTargets._FixPathList,
        'output_patterns': GnTargets._FixOutputPatternsField,
        'outputs': GnTargets._FixOutputsField,
        'response_file_contents': GnTargets._FixPathList,
        'sources': GnTargets._FixSourcesField,
        'script': GnTargets._FixScriptField,
    }
    self.path_handler = PathHandler(self.setup)
    if result_build_dir:
      self.build_dir = result_build_dir
    else:
      self.build_dir = self.package.build_dir

    self.file_conflicts = file_conflicts

  def Fix(self) -> 'GnTargets':
    """Go through targets and their fields, fix what you can."""

    for target in self.data:
      for field in self.data[target]:
        if field in self.fields_to_resolve:
          self.data[target][field] = self.fields_to_resolve[field](
              self, self.data[target][field])
    return self

  def _FixScriptField(self, script_file: str) -> str:
    """
    Fix script file path. Ensure it exist and is the same as |scrip_file|.

    Raises:
      * TargetPathException if actual script file not found.
      * TargetPathException if temp and actual script files have different
        data.
    """
    temp_script_file, actual_script_file = self.path_handler.FixPath(
        script_file, self.package, conflicting_paths=self.file_conflicts)
    if temp_script_file == actual_script_file:
      return actual_script_file

    if not filecmp.cmp(temp_script_file, actual_script_file):
      if self.package.is_highly_volatile:
        g_logger.debug(
            '%s: Temp and actual scripts differ. Possibly patches: %s vs %s',
            self.package.name, temp_script_file, actual_script_file)
      else:
        raise GnTargets.TargetPathException(self.package,
                                            'Temp and actual scripts differ',
                                            temp_script_file,
                                            actual_script_file)

    return actual_script_file

  def _FixArgsField(self, args_list: List[str]) -> List[str]:
    return self._FixArgList(args_list)

  def _FixSourcesField(self, path_list: List[str]) -> List[str]:
    return self._FixPathList(path_list)

  def _FixInputsField(self, path_list: List[str]) -> List[str]:
    return self._FixPathList(path_list)

  def _FixOutputsField(self, path_list: List[str]) -> List[str]:
    return self._FixPathList(path_list)

  def _FixOutputPatternsField(self, pattern_list: List[str]) -> List[str]:
    # File name is not actual file, but some pattern. Let's fix it's directory
    # instead.
    fixed_pattern_dirs = self._FixPathList(
        [os.path.dirname(p) for p in pattern_list])
    return [
        os.path.join(dir, os.path.basename(pattern))
        for dir, pattern in zip(fixed_pattern_dirs, pattern_list)
    ]

  def _FixPathList(self, path_list: List[str]) -> List[str]:
    return [self._FixPath(path).actual for path in path_list]

  def _FixArgList(self, args_list: List[str]) -> List[str]:
    """Fix paths in arguments. Ignores all misses."""

    actual_arg_list = []
    for arg in args_list:
      actual_arg = ','.join([
          ':'.join([self.FixArg(subsubarg)
                    for subsubarg in subarg.split(':')])
          for subarg in arg.split(',')
      ])
      actual_arg_list.append(actual_arg)

    return actual_arg_list

  def FixArg(self, arg: str) -> str:

    def Fixer(chroot_path):
      return self._FixPath(chroot_path).actual

    arg_prefix, actual_path = PathHandler.FixPathInArgument(arg, Fixer)
    return arg_prefix + actual_path

  def _FixPath(self, chroot_path: str) -> PathHandler.FixedPath:
    """
    Wrapper for |PathHandler.FixPathWithIgnores| with all ignores set and
    additional action to move path from |package.build_dir| to
    |self.result_build_dir|.
    """
    fixed_path = self.path_handler.FixPathWithIgnores(
        chroot_path,
        self.package,
        conflicting_paths=self.file_conflicts,
        ignore_highly_volatile=True,
        ignore_generated=True,
        ignorable_dirs=self.setup.ignorable_dirs,
        ignorable_extensions=GnTargets.g_ignorable_extensions)

    if fixed_path.actual.startswith(self.package.build_dir):
      return PathHandler.FixedPath(
          fixed_path.original,
          PathHandler.MovePath(fixed_path.actual, self.package.build_dir,
                               self.build_dir))
    return fixed_path


class GnTargetsMerger:
  """Responsible for merging targets."""

  class GnTargetsMergeException(Exception):
    """Exception to indicate failure while merging gn targets."""

  def __init__(self):
    self.data = {}
    self.fields_to_resolve = {
        'all_dependent_configs': GnTargetsMerger._MergeLists,
        'defines': GnTargetsMerger._MergeLists,
        'deps': GnTargetsMerger._MergeLists,
        'cflags_c': GnTargetsMerger._MergeLists,
        'cflags_cc': GnTargetsMerger._MergeLists,
        'include_dirs': GnTargetsMerger._MergeLists,
        'inputs': GnTargetsMerger._MergeLists,
        'lib_dirs': GnTargetsMerger._MergeLists,
        'libs': GnTargetsMerger._MergeLists,
        'outputs': GnTargetsMerger._MergeLists,
        # Everthing else shall be either unique or equal.
    }

  @staticmethod
  def _MergeLists(first: List, second: List, field_name: str) -> List:
    return first + [element for element in second if element not in first]

  def Append(self, new_targets: GnTargets) -> None:
    """Add targets from |new_targets| to existing ones."""

    for target in new_targets.data:
      if not target in self.data:
        # Brand new target. Nothing to merge.
        self.data[target] = new_targets.data[target]
        continue

      g_logger.debug('%s: Merging existing target: %s',
                     new_targets.package.name, target)

      for field in new_targets.data[target]:
        if not field in self.data[target]:
          # Brand new field. Nothing to merge.
          self.data[target][field] = new_targets.data[target][field]
          continue

        field_data = self.data[target][field]
        new_field_data = new_targets.data[target][field]

        if field_data == new_field_data:
          # Fields equal. Nothing  to merge.
          continue

        g_logger.debug('%s: %s: Merging existing field: %s',
                       new_targets.package.name, target, field)

        if not field in self.fields_to_resolve:
          raise GnTargetsMerger.GnTargetsMergeException(
              f"{new_targets.package.name}: Unknown '{field}' in '{target}'")

        self.data[target][field] = self.fields_to_resolve[field](field_data,
                                                                 new_field_data,
                                                                 field)


class GnTargetsGenerator:
  """Generates and fixes output of gn desc command generating gn targets."""

  class RootDirException(PackagePathException):
    """Indicates troubles with root dir."""

  def __init__(self,
               setup: Setup,
               *,
               result_build_dir: str = None,
               file_conflicts: Dict = {},
               keep_going: bool = False):
    """
    GnTargetsGenerator constructor.

    Arguments:
      * setup: setup data (board, dirs, etc).
      * result_build_dir: path to result build dir simulating single result
        package.
      * file_conflicts: maps original build artifacts in chroot dir that
        conflict between packages to corresponding artfacts in
        |result_build_dir|.
    """
    self.setup = setup
    self.result_build_dir = result_build_dir
    self.file_conflicts = file_conflicts
    self.keep_going = keep_going

  def _FindRootDir(self, package: Package) -> str:
    """Returns a dir from which it's possible to generate gn targets."""

    for src_match in package.src_dir_matches:
      if os.path.isfile(os.path.join(src_match.temp, '.gn')):
        return src_match.temp

    raise GnTargetsGenerator.RootDirException(package, 'Cannot find root dir')

  def _GenerateTargetsForPackage(self, package: Package) -> GnTargets:
    path_handler = PathHandler(self.setup)
    chroot_targets_root_dir = path_handler.ToChroot(self._FindRootDir(package))
    chroot_build_dir = path_handler.ToChroot(package.build_dir)
    targets_str = CrosSdk(self.setup).GenerateGnTargets(chroot_targets_root_dir,
                                                        chroot_build_dir)
    targets_str = targets_str[targets_str.find('{'):targets_str.rfind('}') + 1]
    g_logger.debug('%s: Generated targets', package.name)

    targets_data = json.loads(targets_str)
    if not targets_data:
      g_logger.error('%s: gn targets are empty', package)

    if not isinstance(targets_data, Dict):
      raise NotImplementedError(
          f"gn targets are not dict for package: {package.name}")

    return GnTargets(targets_data,
                     package,
                     self.setup,
                     result_build_dir=self.result_build_dir,
                     file_conflicts=self.file_conflicts)

  def _GenerateResultTargets(self, packages: List[Package]) -> List:
    """Generates, fixes and merges gn_targets for given packages."""

    result_targets = GnTargetsMerger()

    for package in packages:
      try:
        new_targets = self._GenerateTargetsForPackage(package).Fix()
        result_targets.Append(new_targets)
        g_logger.debug('%s: targets merged', package.name)
      except (GnTargets.TargetPathException,
              GnTargetsMerger.GnTargetsMergeException,
              PackagePathException) as e:
        if self.keep_going:
          g_logger.error('%s: Failed to fix gn targets: %s', package.name, e)
        else:
          raise e

    return result_targets.data

  def Generate(self, packages: List[Package], result_targets_file: str) -> str:
    """
    Generates, fixes and merges gn_targets for given packages.

    Raises:
      * TargetPathException if failed to fix a target.
    """
    assert result_targets_file

    result_targets = self._GenerateResultTargets(packages)

    with open(result_targets_file, 'w') as output:
      json.dump(result_targets, output, indent=2)
