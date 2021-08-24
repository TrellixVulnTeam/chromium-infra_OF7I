# pylint: skip-file
import logging
import pprint
import utilities

from gopo import Version

"""
The Library class represents a library equivalent to a wheel. Contents are:
  * The library name.
  * A dict with the python versions this library is available for, and each
    individual version's wheel available for that python version.
"""
class Library:
  _UNIVERSAL_VERSION = 'universal'

  def __init__(self, name: str):
    self.name = name
    self.python_versions = {}

  def add_version(self,
                  python_version: str,
                  library_version: str,
                  wheel: dict) -> None:
    """Adds a specific version (both python and individual) to this library.

    Add a specific wheel definition (identified by python version and individual
    version) to this library.

    Args:
      python_version: The python version for this library.
      library_version: The individual (pypi) version for this library.
      wheel: The wheel descriptor for this version.
    """
    self.python_versions.setdefault(python_version, {})
    self.python_versions[python_version][Version(library_version)] = wheel

  def find_best_fit(self,
                    python_version: str,
                    library_version_range: dict) -> (Version, dict):
    """Finds the best fit given a python version and a range for this library.

    Tries to determine the best version of this library given the constraints
    of the python version required, as well as a dict keyed-in by a comparison
    operator, valued with a version number string.

    Args:
      python_version: The parsed python version desired.
      library_version_range: A dict keyed by a comparison operator, with version
        as value. Used to specify a range of versions desired.
    Returns:
      A tuple with the best fit version object and apposite wheel definition.
    """
    candidates = {}
    versions = [Library._UNIVERSAL_VERSION, python_version]
    for current_version in versions:
      if self.python_versions.get(current_version):
        current_wheel_dict = self.python_versions[current_version]
        candidates.update(current_wheel_dict)

    logging.debug("Candidates for best fit: {}".format(candidates))
    return self._find_best_fit(candidates, library_version_range)

  def _find_best_fit(self,
                     candidates: dict,
                     library_version_range: dict) -> (Version, dict):
    """Finds best-fit given a pypi version range from a list of candidates.

    Given a range of pypi versions and a set of candidates, returns the
    best match.

    Args:
      candidates: A dict of wheels to match from.
      library_version_range: A dict keyed by a comparison operator, with version
        as value. Used to specify a range of versions desired.
      Returns:
        A tuple with the best fit version object and apposite wheel definition.
    """
    for candidate_version, candidate_wheel in sorted(candidates.items(),
                                                     key=lambda x: x[0],
                                                     reverse=True):
      all_pass = True
      for comparison_operator, desired_version_number in library_version_range.items():
        if not utilities.OPERATIONS_MAP[comparison_operator](
            candidate_version,
            Version(desired_version_number)):
          all_pass = False
          break
      # If no comparisons fail, we can use this library!
      if all_pass:
        return candidate_version, candidate_wheel

    raise LookupError(
        "No versions match requirement: {} for library: {}"
        .format(library_version_range,
                self.name))

  def __str__(self) -> str:
    return pprint.pformat({
        'name': self.name,
        'versions': self.python_versions
    })

  def __repr__(self) -> str:
    return self.__str__()
