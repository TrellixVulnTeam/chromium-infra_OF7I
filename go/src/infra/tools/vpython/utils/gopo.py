# pylint: skip-file
""" Good Old Python Object (Python POJO equivalent)
Here we house all simple unemcumbered objects.
"""
import pprint
import re

#CONSTANTS
ANY_VERSION =  {'>': '-1'}

"""
The dependency class holds all metadata relative to a single pypi dependency.
This includes:
  * The library name (including the displayed name and a cleaned-up version)
  * The library versions (stored as a dict of version constraints)
  * Extra dependency metadata (includes tags, python version, etc.)
"""
class Dependency:
  _BRACKET_OPEN = '['
  _BRACKET_CLOSE = ']'
  _INTERNAL_SEPARATOR = '.'
  _EXTERNAL_SEPARATOR = '-'
  _CANONICAL_SEPARATOR = '_'

  def __init__(self,
               library_name: str,
               versions: dict=None,
               metadata: dict=None):
    self.library_name = self.clean_library_name(library_name, only_brackets=True)
    self.canonical_name = self.clean_library_name(library_name)
    self.versions = versions or ANY_VERSION
    self.metadata = metadata or {}

  @staticmethod
  def clean_library_name(name: str, only_brackets: bool=False) -> str:
    """Returns the effective library name from the name string.

    Replaces all instances of '.' and '-' with '_' and removes all content
    within brackets. If only brackets is set to True, remove only bracket-
    related content. This is necessary as library names across pip and wheel
    are not consistent (in fact they are not consistent within pip itself).

    Args:
      name: The library name.
      only_brackets: Whether to only strip brackets.
    Returns:
      The sanitized library name.
    """
    result = ''
    inside_parens = False
    for character in name:
      if character == Dependency._BRACKET_OPEN:
        inside_parens = True
      elif character == Dependency._BRACKET_CLOSE:
        inside_parens = False
      elif inside_parens == False:
        if not only_brackets and (character == Dependency._EXTERNAL_SEPARATOR
                                  or character == Dependency._INTERNAL_SEPARATOR):
          result += Dependency._CANONICAL_SEPARATOR
        else:
          result += character
    return result

  def __str__(self) -> str:
    return pprint.pformat({'library_name': self.library_name,
                           'canonical_name': self.canonical_name,
                           'versions': self.versions,
                           'metadata': self.metadata})

  def __repr__(self) -> str:
    return self.__str__()


"""
Version holds a pypi version (as defined in PEP 440). It is of note that pypi
does not actually conform to PEP 440 in all instances, so we can't rely on
provided regexes or methods for the sanitization. Here we store:
  * The version input string.
  * A sanitized representation of the version in the form of a list, delimited
    by version category.
"""
class Version:
  _SEPARATOR_TOKEN = '.'
  _NEGATIVE_TOKEN = '-'

  def __init__(self, version: str):
    # Unfortunately, not all PIP versions follow PEP 440, so we can't validate
    # correctly here.
    self._raw_definition = version
    self.definition = self._sanitize_version(version).split(self._SEPARATOR_TOKEN)

  def get_relevant_version(self) -> str:
    """Returns the major and minor version as a string."""
    return "{}{}".format(
        self.get_major_version(),
       ".{}".format(self.get_minor_version()) if self.has_minor_version() else "")

  def get_major_version(self) -> str:
    """Returns the major version only."""
    return self.definition[0]

  def get_minor_version(self) -> str:
    """Returns the minor version only, if it exists."""
    return self.definition[1] if self.has_minor_version() else None

  def has_minor_version(self) -> bool:
    """Determines if this version has a minor component."""
    return len(self.definition) > 1

  @staticmethod
  def is_canonical_python_version(version: str) -> bool:
    """Determines if this version is canonical as per PEP 440.

    Applies the PEP-supplied RegEx which should match if this is a PEP 440-compliant version.

    Args:
      version: The raw version string.
    Returns:
      True if this is a PEP 440-compliant version. False otherwise.
    """
    return re.match(r'^([1-9][0-9]*!)?(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))*((a|b|rc)(0|[1-9][0-9]*))?(\.post(0|[1-9][0-9]*))?(\.dev(0|[1-9][0-9]*))?$', version) is not None

  @staticmethod
  def _sanitize_version(version: str) -> str:
    """Returns a cleaned-up representation of the input raw version in pypi.

    This simple sanitization method keeps numbers, dots and hyphens, but strips
    all else (in fact at the first non-compliant match, stop parsing).
    Versioning is a complex item, here. As such we admit that we won't cover
    all cases and instead choose to ignore cases such as dev etc, and just
    drop non-numbers, non-dots (including epochs). This is necessary as a lot
    of versions do not follow PEP 440, so no matter what, this will be broken
    in one way or another.

    Args:
      version: The raw version string.
    Returns:
      A sanitized version of that raw string.
    """
    result = ''
    for character in version:
      if character in [Version._SEPARATOR_TOKEN, Version._NEGATIVE_TOKEN] or character.isdigit():
        result += character
      else:
        if result[-1] == Version._SEPARATOR_TOKEN:
          result = result[:-1]
        break
    return result

  # Following are a number of version comparators to aid in version match
  # determination.
  def __lt__(self, other: 'Version') -> bool:
    comparisons = min(len(self.definition), len(other.definition))

    for current_index in range(comparisons):
      if int(self.definition[current_index]) < int(other.definition[current_index]):
        return True
      elif int(self.definition[current_index]) > int(other.definition[current_index]):
        return False
      # if equal keep going

    return False

  def __gt__(self, other: 'Version') -> bool:
    comparisons = min(len(self.definition), len(other.definition))

    for current_index in range(comparisons):
      if int(self.definition[current_index]) < int(other.definition[current_index]):
        return False
      elif int(self.definition[current_index]) > int(other.definition[current_index]):
        return True
      # if equal keep going

    return False

  def __eq__(self, other: 'Version') -> bool:
    return self._raw_definition == other._raw_definition

  def __ne__(self, other: 'Version') -> bool:
    return not self.__eq(other)

  def __ge__(self, other: 'Version') -> bool:
    return self.__eq__(other) or self.__gt__(other)

  def __le__(self, other: 'Version') -> bool:
    return self.__eq__(other) or self.__lt__(other)

  def __hash__(self) -> str:
    return hash(self._raw_definition)

  def __str__(self) -> str:
    return self._raw_definition

  def __repr__(self) -> str:
    return self.__str__()
