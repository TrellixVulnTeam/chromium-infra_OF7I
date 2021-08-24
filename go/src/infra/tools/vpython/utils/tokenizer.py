# pylint: skip-file
import logging
import pprint
import re

"""
This class defines a simple token, storing:
  * The token type (e.g.: Number, word, etc)
  * The token value (the actual string representation)
"""
class Token:
  def __init__(self, token_type: str, token_value: str):
    self.type = token_type
    self.value = token_value

  def __str__(self) -> str:
    return pprint.pformat({'type': self.type, 'value': self.value}, indent=2)

  def __repr__(self) -> str:
    return self.__str__()


"""
The DependencyTokenizer class holds a number of methods which help tokenize
python dependencies as found via a pypi query. This is meant to be initialized
with the dependency raw string, and methods should be called based on expected
inputs. Stored here is:
  * The Token list for the raw input string.
"""
class DependencyTokenizer:
  """
   Expected as follows (here [] denotes optional, and $ denotes variable, ""
   denotes description):
     $name [$version_range] [$metadata]
   Where name is:
     "a string starting with a character"
   Where version_range is:
     ($comparator$version[,$comparator$version)]
   Where comparator is:
     "any of ==, !=, <=, >=, <, >"
   Where version is:
     "'Defined' by PEP 440 (though most do not comply)"
   Where metadata is:
     ; [(] $key_value [$more][)] [$more]
   Where key_value is:
     $key $comparator $value
   Where key is:
     $name
   Where value is:
     "any of $name or $version"
   Where more is:
     and [(] $key_value [$more] [)] [$more]
  """
  # TOKENS
  VERSION_TOKEN = 'VERSION'
  NAME_TOKEN = 'NAME'
  COMPARATOR_TOKEN = 'COMPARATOR'
  METADATA_SEPARATOR_TOKEN = 'METADATA_SEPARATOR'
  STRING_TOKEN = 'STRING'
  VERSION_START_TOKEN = 'VERSION_START'
  VERSION_END_TOKEN = 'VERSION_END'
  VERSION_SEPARATOR_TOKEN = 'VERSION_SEPARATOR'


  def __init__(self, raw_requirement_string: str):
    self.tokens = self._tokenize_requirement_metadata_string(raw_requirement_string)
    logging.debug("Tokens: {}".format(self.tokens))

  def is_token_type(self, expected_type: str):
    """Whether the next token is of the expected type."""
    if not self.has_next():
      return False
    return expected_type == self.peek_next_token().type

  def get_next_token(self) -> Token:
    """Retrieves the next token effectively popping it from the list."""
    token = self.tokens.pop(0)
    return token

  def peek_next_token(self) -> Token:
    """Retrieves the next token without removing it from the list."""
    if not self.has_next():
      return None
    return self.tokens[0]

  def has_next(self) -> bool:
    """Whether there is another token left to process."""
    return len(self.tokens) != 0

  def drop_next(self) -> None:
    """Removes the next token from the list."""
    self.tokens.pop(0)

  def drop_next_if_type(self, expected_type: str) -> None:
    """Removes the next token from the list if it matches the given type."""
    if self.is_token_type(expected_type):
      self.drop_next()

  def parse_token_type(self, expected_type: str) -> str:
    """Returns the next token value if of expected_type, else throw Error."""
    token = self.get_next_token()
    if token.type != expected_type:
      raise TypeError(
          "Token Mismatch: Expected: {}, instead got: {}({})".format(
              expected_type,
              token.value,
              token.type))
    return token.value

  def parse_name(self) -> str:
    """Retrieves the next token if it's a name type."""
    logging.debug("Parsing Library name!")
    return self.parse_token_type(DependencyTokenizer.NAME_TOKEN)

  def parse_versions(self) -> dict:
    """Retrieves the next token if it's a version type."""
    logging.debug("Parsing Library versions!")
    versions = {}

    # Dealing here with specific case where version start token
    # (and end token) are not present. This should not happen, but it does.
    if not (self.is_token_type(DependencyTokenizer.VERSION_START_TOKEN)
            or self.is_token_type(DependencyTokenizer.COMPARATOR_TOKEN)):
      return None
    if self.is_token_type(DependencyTokenizer.VERSION_START_TOKEN):
      self.drop_next()

    while True:
      version_comparator = self.parse_token_type(DependencyTokenizer.COMPARATOR_TOKEN)
      version = self.parse_token_type(DependencyTokenizer.VERSION_TOKEN)
      versions.update({version_comparator: version})
      if not self.is_token_type(DependencyTokenizer.VERSION_SEPARATOR_TOKEN):
        break
      self.drop_next()
    self.drop_next_if_type(DependencyTokenizer.VERSION_END_TOKEN)
    return versions

  def parse_kv(self) -> (str, str, str):
    """Retrieves a tuple equivalent of the next token if it's a key-value."""
    logging.debug("Parsing Key Value!")
    self.drop_next_if_type(DependencyTokenizer.VERSION_START_TOKEN)
    key = self.parse_token_type(DependencyTokenizer.NAME_TOKEN)
    comparator = self.parse_token_type(DependencyTokenizer.COMPARATOR_TOKEN)
    self.parse_token_type(DependencyTokenizer.STRING_TOKEN)
    value = self.get_next_token().value
    self.parse_token_type(DependencyTokenizer.STRING_TOKEN)
    self.drop_next_if_type(DependencyTokenizer.VERSION_END_TOKEN)
    return key, comparator, value

  def parse_metadata(self) -> dict:
    """Retrieves a dict equivalent of the next token if it's a metadata type."""
    logging.debug("Parsing Library extras!")
    metadata = {}
    if not self.is_token_type(DependencyTokenizer.METADATA_SEPARATOR_TOKEN):
      return None
    try:
      while True:
        self.drop_next()
        key, comparator, value = self.parse_kv()
        metadata[key] = {'value': value, 'comparator': comparator}
        if not self.is_token_type(DependencyTokenizer.NAME_TOKEN):
          # Didn't find an 'and'
          break

    except Exception as err:
      logging.warning(err)
      return {'extra': {'value': 'unparsable'}}
    return metadata

  @staticmethod
  def _tokenize_requirement_metadata_string(raw_requirement_string: str) -> list:
    """Tokenize the input raw requirement string."""
    logging.info("Tokenizing {}".format(raw_requirement_string))
    scanner=re.Scanner([
      (r"[0-9]+[.A-Za-z0-9]*",            lambda scanner,token:Token(DependencyTokenizer.VERSION_TOKEN, token)),
      (r"[A-Za-z][\[\]A-Za-z0-9\._\-]+",  lambda scanner,token:Token(DependencyTokenizer.NAME_TOKEN, token)),
      (r"[<>=!]={0,1}",                   lambda scanner,token:Token(DependencyTokenizer.COMPARATOR_TOKEN, token)),
      (r";",                              lambda scanner,token:Token(DependencyTokenizer.METADATA_SEPARATOR_TOKEN, token)),
      (r"['\"]",                          lambda scanner,token:Token(DependencyTokenizer.STRING_TOKEN, token)),
      (r"\(",                             lambda scanner,token:Token(DependencyTokenizer.VERSION_START_TOKEN, token)),
      (r"\)",                             lambda scanner,token:Token(DependencyTokenizer.VERSION_END_TOKEN, token)),
      (r",",                              lambda scanner,token:Token(DependencyTokenizer.VERSION_SEPARATOR_TOKEN, token)),
      (r"\s+",                            None), # None == skip token.
    ])
    results, _ = scanner.scan(raw_requirement_string)
    return results
