# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""A set of functions that integrate the GAE search index with Monorail."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import collections
import datetime
import logging
import re
import time

from google.appengine.api import search

from proto import ast_pb2
from proto import tracker_pb2


# TODO(jrobbins): Consider re-implementing this whole file by using a
# BNF syntax specification and a parser generator or library.

# encodings
UTF8 = 'utf-8'

# Operator used for OR statements.
OR_SYMBOL = ' OR '

# Token types used for parentheses parsing.
SUBQUERY = ast_pb2.TokenType.SUBQUERY
LEFT_PAREN = ast_pb2.TokenType.LEFT_PAREN
RIGHT_PAREN = ast_pb2.TokenType.RIGHT_PAREN
OR = ast_pb2.TokenType.OR

# Field types and operators
BOOL = tracker_pb2.FieldTypes.BOOL_TYPE
DATE = tracker_pb2.FieldTypes.DATE_TYPE
NUM = tracker_pb2.FieldTypes.INT_TYPE
TXT = tracker_pb2.FieldTypes.STR_TYPE
APPROVAL = tracker_pb2.FieldTypes.APPROVAL_TYPE

EQ = ast_pb2.QueryOp.EQ
NE = ast_pb2.QueryOp.NE
LT = ast_pb2.QueryOp.LT
GT = ast_pb2.QueryOp.GT
LE = ast_pb2.QueryOp.LE
GE = ast_pb2.QueryOp.GE
TEXT_HAS = ast_pb2.QueryOp.TEXT_HAS
NOT_TEXT_HAS = ast_pb2.QueryOp.NOT_TEXT_HAS
IS_DEFINED = ast_pb2.QueryOp.IS_DEFINED
IS_NOT_DEFINED = ast_pb2.QueryOp.IS_NOT_DEFINED
KEY_HAS = ast_pb2.QueryOp.KEY_HAS

# Mapping from user query comparison operators to our internal representation.
OPS = {
    ':': TEXT_HAS,
    '=': EQ,
    '!=': NE,
    '<': LT,
    '>': GT,
    '<=': LE,
    '>=': GE,
}

# When the query has a leading minus, switch the operator for its opposite.
NEGATED_OPS = {
    EQ: NE,
    NE: EQ,
    LT: GE,
    GT: LE,
    LE: GT,
    GE: LT,
    TEXT_HAS: NOT_TEXT_HAS,
    # IS_DEFINED is handled separately.
    }

# This is a partial regular expression that matches all of our comparison
# operators, such as =, 1=, >, and <.  Longer ones listed first so that the
# shorter ones don't cause premature matches.
OPS_PATTERN = '|'.join(
    map(re.escape, sorted(list(OPS.keys()), key=lambda op: -len(op))))

# This RE extracts search terms from a subquery string.
TERM_RE = re.compile(
    r'(-?"[^"]+")|'  # E.g., ["division by zero"]
    r'(\S+(%s)[^ "]+)|'  # E.g., [stars>10]
    r'(\w+(%s)"[^"]+")|'  # E.g., [summary:"memory leak"]
    r'(-?[._\*\w][-._\*\w]+)'  # E.g., [-workaround]
    % (OPS_PATTERN, OPS_PATTERN), flags=re.UNICODE)

# This RE is used to further decompose a comparison term into prefix, op, and
# value.  E.g., [stars>10] or [is:open] or [summary:"memory leak"].  The prefix
# can include a leading "-" to negate the comparison.
OP_RE = re.compile(
    r'^(?P<prefix>[-_.\w]*?)'
    r'(?P<op>%s)'
    r'(?P<value>([-@\w][-\*,./:<=>@\w]*|"[^"]*"))$' %
    OPS_PATTERN,
    flags=re.UNICODE)


# Predefined issue fields passed to the query parser.
_ISSUE_FIELDS_LIST = [
    (ast_pb2.ANY_FIELD, TXT),
    ('attachment', TXT),  # attachment file names
    ('attachments', NUM),  # number of attachment files
    ('blocked', BOOL),
    ('blockedon', TXT),
    ('blockedon_id', NUM),
    ('blocking', TXT),
    ('blocking_id', NUM),
    ('cc', TXT),
    ('cc_id', NUM),
    ('comment', TXT),
    ('commentby', TXT),
    ('commentby_id', NUM),
    ('component', TXT),
    ('component_id', NUM),
    ('description', TXT),
    ('gate', TXT),
    ('hotlist', TXT),
    ('hotlist_id', NUM),
    ('id', NUM),
    ('is_spam', BOOL),
    ('label', TXT),
    ('label_id', NUM),
    ('mergedinto', NUM),
    ('mergedinto_id', NUM),
    ('open', BOOL),
    ('owner', TXT),
    ('ownerbouncing', BOOL),
    ('owner_id', NUM),
    ('project', TXT),
    ('reporter', TXT),
    ('reporter_id', NUM),
    ('spam', BOOL),
    ('stars', NUM),
    ('starredby', TXT),
    ('starredby_id', NUM),
    ('status', TXT),
    ('status_id', NUM),
    ('summary', TXT),
    ]

_DATE_FIELDS = (
    'closed',
    'modified',
    'opened',
    'ownermodified',
    'ownerlastvisit',
    'statusmodified',
    'componentmodified',
    )

# Add all _DATE_FIELDS to _ISSUE_FIELDS_LIST.
_ISSUE_FIELDS_LIST.extend((date_field, DATE) for date_field in _DATE_FIELDS)

_DATE_FIELD_SUFFIX_TO_OP = {
    '-after': '>',
    '-before': '<',
}

SET_BY_SUFFIX = '-by'
SET_ON_SUFFIX = '-on'
APPROVER_SUFFIX = '-approver'
STATUS_SUFFIX = '-status'

_APPROVAL_SUFFIXES = (
    SET_BY_SUFFIX,
    SET_ON_SUFFIX,
    APPROVER_SUFFIX,
    STATUS_SUFFIX,
)

BUILTIN_ISSUE_FIELDS = {
    f_name: tracker_pb2.FieldDef(field_name=f_name, field_type=f_type)
    for f_name, f_type in _ISSUE_FIELDS_LIST}


# Do not treat strings that start with the below as key:value search terms.
# See bugs.chromium.org/p/monorail/issues/detail?id=419 for more detail.
NON_OP_PREFIXES = (
    'http:',
    'https:',
)


def ParseUserQuery(
    query, scope, builtin_fields, harmonized_config, warnings=None,
    now=None):
  # type: (str, str, Mapping[str, proto.tracker_pb2.FieldDef],
  #   proto.tracker_pb2.ProjectIssueConfig, Sequence[str], int) ->
  #     proto.ast_pb2.QueryAST
  """Parse a user query and return a set of structure terms.

  Args:
    query: string with user's query.  E.g., 'Priority=High'.
    scope: string search terms that define the scope in which the
        query should be executed.  They are expressed in the same
        user query language.  E.g., adding the canned query.
    builtin_fields: dict {field_name: FieldDef(field_name, type)}
        mapping field names to FieldDef objects for built-in fields.
    harmonized_config: config for all the projects being searched.
        @@@ custom field name is not unique in cross project search.
         - custom_fields = {field_name: [fd, ...]}
         - query build needs to OR each possible interpretation
         - could be label in one project and field in another project.
        @@@ what about searching across all projects?
    warnings: optional list to accumulate warning messages.
    now: optional timestamp for tests, otherwise time.time() is used.

  Returns:
    A QueryAST with conjunctions (usually just one), where each has a list of
    Condition PBs with op, fields, str_values and int_values.  E.g., the query
    [priority=high leak OR stars>100] over open issues would return
    QueryAST(
      Conjunction(Condition(EQ, [open_fd], [], [1]),
                  Condition(EQ, [label_fd], ['priority-high'], []),
                  Condition(TEXT_HAS, any_field_fd, ['leak'], [])),
      Conjunction(Condition(EQ, [open_fd], [], [1]),
                  Condition(GT, [stars_fd], [], [100])))

  Raises:
    InvalidQueryError: If a problem was detected in the user's query.
  """
  if warnings is None:
    warnings = []

  # Convert the overall query into one or more OR'd subqueries.
  subqueries = QueryToSubqueries(query)

  # Make a dictionary of all fields: built-in + custom in each project.
  combined_fields = collections.defaultdict(
      list, {field_name: [field_def]
             for field_name, field_def in builtin_fields.items()})
  for fd in harmonized_config.field_defs:
    if fd.field_type != tracker_pb2.FieldTypes.ENUM_TYPE:
      # Only do non-enum fields because enums are stored as labels
      combined_fields[fd.field_name.lower()].append(fd)
      if fd.field_type == APPROVAL:
        for approval_suffix in _APPROVAL_SUFFIXES:
          combined_fields[fd.field_name.lower() + approval_suffix].append(fd)

  conjunctions = [
      _ParseConjunction(sq, scope, combined_fields, warnings, now=now)
      for sq in subqueries]
  return ast_pb2.QueryAST(conjunctions=conjunctions)


def _ParseConjunction(subquery, scope, fields, warnings, now=None):
  # type: (str, str, Mapping[str, proto.tracker_pb2.FieldDef], Sequence[str],
  #     int) -> proto.ast_pb2.Condition
  """Parse part of a user query into a Conjunction PB."""
  scoped_query = ('%s %s' % (scope, subquery)).lower()
  cond_strs = _ExtractConds(scoped_query, warnings)
  conds = [_ParseCond(cond_str, fields, warnings, now=now)
           for cond_str in cond_strs]
  conds = [cond for cond in conds if cond]
  return ast_pb2.Conjunction(conds=conds)


def _ParseCond(cond_str, fields, warnings, now=None):
  # type: (str, Mapping[str, proto.tracker_pb2.FieldDef], Sequence[str],
  #     int) -> proto.ast_pb2.Condition
  """Parse one user query condition string into a Condition PB."""
  op_match = OP_RE.match(cond_str)
  # Do not treat as key:value search terms if any of the special prefixes match.
  special_prefixes_match = any(
      cond_str.startswith(p) for p in NON_OP_PREFIXES)
  if op_match and not special_prefixes_match:
    prefix = op_match.group('prefix')
    op = op_match.group('op')
    val = op_match.group('value')
    # Special case handling to continue to support old date query terms from
    # code.google.com. See monorail:151 for more details.
    if prefix.startswith(_DATE_FIELDS):
      for date_suffix in _DATE_FIELD_SUFFIX_TO_OP:
        if prefix.endswith(date_suffix):
          prefix = prefix.rstrip(date_suffix)
          op = _DATE_FIELD_SUFFIX_TO_OP[date_suffix]
    return _ParseStructuredTerm(prefix, op, val, fields, now=now)

  # Treat the cond as a full-text search term, which might be negated.
  if cond_str.startswith('-'):
    op = NOT_TEXT_HAS
    cond_str = cond_str[1:]
  else:
    op = TEXT_HAS

  # Construct a full-text Query object as a dry-run to validate that
  # the syntax is acceptable.
  try:
    _fts_query = search.Query(cond_str)
  except search.QueryError:
    warnings.append('Ignoring full-text term: %s' % cond_str)
    return None

  # Flag a potential user misunderstanding.
  if cond_str.lower() in ('and', 'or', 'not'):
    warnings.append(
        'The only supported boolean operator is OR (all capitals).')

  return ast_pb2.MakeCond(
      op, [BUILTIN_ISSUE_FIELDS[ast_pb2.ANY_FIELD]], [cond_str], [])


def _ParseStructuredTerm(prefix, op_str, value, fields, now=None):
  # type: (str, str, str, Mapping[str, proto.tracker_pb2.FieldDef]) ->
  #     proto.ast_pb2.Condition
  """Parse one user structured query term into an internal representation.

  Args:
    prefix: The query operator, usually a field name.  E.g., summary. It can
      also be special operators like "is" to test boolean fields.
    op_str: the comparison operator.  Usually ":" or "=", but can be any OPS.
    value: the value to compare against, e.g., term to find in that field.
    fields: dict {name_lower: [FieldDef, ...]} for built-in and custom fields.
    now: optional timestamp for tests, otherwise time.time() is used.

  Returns:
    A Condition PB.
  """
  unquoted_value = value.strip('"')
  # Quick-OR is a convenient way to write one condition that matches any one of
  # multiple values, like set membership.  E.g., [Priority=High,Critical].
  # Ignore empty values caused by duplicated or trailing commas. E.g.,
  # [Priority=High,,Critical,] is equivalent to [Priority=High,Critical].
  quick_or_vals = [v.strip() for v in unquoted_value.split(',') if v.strip()]

  op = OPS[op_str]
  negate = False
  if prefix.startswith('-'):
    negate = True
    op = NEGATED_OPS.get(op, op)
    prefix = prefix[1:]

  if prefix == 'is' and unquoted_value in [
      'open', 'blocked', 'spam', 'ownerbouncing']:
    return ast_pb2.MakeCond(
        NE if negate else EQ, fields[unquoted_value], [], [])

  # Search entries with or without any value in the specified field.
  if prefix == 'has':
    op = IS_NOT_DEFINED if negate else IS_DEFINED
    if '.' in unquoted_value:  # Possible search for phase field with any value.
      phase_name, possible_field = unquoted_value.split('.', 1)
      if possible_field in fields:
        return ast_pb2.MakeCond(
            op, fields[possible_field], [], [], phase_name=phase_name)
    elif unquoted_value in fields:  # Look for that field with any value.
      return ast_pb2.MakeCond(op, fields[unquoted_value], [], [])
    else:  # Look for any label with that prefix.
      return ast_pb2.MakeCond(op, fields['label'], [unquoted_value], [])

  # Search entries with certain gates.
  if prefix == 'gate':
    return ast_pb2.MakeCond(op, fields['gate'], quick_or_vals, [])

  # Determine hotlist query type.
  # If prefix is not 'hotlist', quick_or_vals is empty, or qov
  # does not contain ':', is_fields will remain True
  is_fields = True
  if prefix == 'hotlist':
    try:
      if ':' not in quick_or_vals[0]:
        is_fields = False
    except IndexError:
      is_fields = False

  phase_name = None
  if '.' in prefix and is_fields:
    split_prefix = prefix.split('.', 1)
    if split_prefix[1] in fields:
      phase_name, prefix = split_prefix

  # search built-in and custom fields. E.g., summary.
  if prefix in fields and is_fields:
    # Note: if first matching field is date-type, we assume they all are.
    # TODO(jrobbins): better handling for rare case where multiple projects
    # define the same custom field name, and one is a date and another is not.
    first_field = fields[prefix][0]
    if first_field.field_type == DATE:
      date_values = [_ParseDateValue(val, now=now) for val in quick_or_vals]
      return ast_pb2.MakeCond(op, fields[prefix], [], date_values)
    elif first_field.field_type == APPROVAL and prefix.endswith(SET_ON_SUFFIX):
      date_values = [_ParseDateValue(val, now=now) for val in quick_or_vals]
      return ast_pb2.MakeCond(
          op,
          fields[prefix], [],
          date_values,
          key_suffix=SET_ON_SUFFIX,
          phase_name=phase_name)
    else:
      quick_or_ints = []
      for qov in quick_or_vals:
        try:
          quick_or_ints.append(int(qov))
        except ValueError:
          pass
      if first_field.field_type == APPROVAL:
        for approval_suffix in _APPROVAL_SUFFIXES:
          if prefix.endswith(approval_suffix):
            return ast_pb2.MakeCond(op, fields[prefix], quick_or_vals,
                                    quick_or_ints, key_suffix=approval_suffix,
                                    phase_name=phase_name)
      return ast_pb2.MakeCond(op, fields[prefix], quick_or_vals,
                              quick_or_ints, phase_name=phase_name)

  # Since it is not a field, treat it as labels, E.g., Priority.
  quick_or_labels = ['%s-%s' % (prefix, v) for v in quick_or_vals]
  # Convert substring match to key-value match if user typed 'foo:bar'.
  if op == TEXT_HAS:
    op = KEY_HAS
  return ast_pb2.MakeCond(op, fields['label'], quick_or_labels, [])


def _ExtractConds(query, warnings):
  # type: (str, Sequence[str]) -> Sequence[str]
  """Parse a query string into a list of individual condition strings.

  Args:
    query: UTF-8 encoded search query string.
    warnings: list to accumulate warning messages.

  Returns:
    A list of query condition strings.
  """
  # Convert to unicode then search for distinct terms.
  term_matches = TERM_RE.findall(query)

  terms = []
  for (phrase, word_label, _op1, phrase_label, _op2,
       word) in term_matches:
    # Case 1: Quoted phrases, e.g., ["hot dog"].
    if phrase_label or phrase:
      terms.append(phrase_label or phrase)

    # Case 2: Comparisons
    elif word_label:
      special_prefixes_match = any(
          word_label.startswith(p) for p in NON_OP_PREFIXES)
      match = OP_RE.match(word_label)
      if match and not special_prefixes_match:
        label = match.group('prefix')
        op = match.group('op')
        word = match.group('value')
        terms.append('%s%s"%s"' % (label, op, word))
      else:
        # It looked like a key:value cond, but not exactly, so treat it
        # as fulltext search.  It is probably a tiny bit of source code.
        terms.append('"%s"' % word_label)

    # Case 3: Simple words.
    elif word:
      terms.append(word)

    else:  # pragma: no coverage
      warnings.append('Unparsable search term')

  return terms


def _ParseDateValue(val, now=None):
  # type: (str, int) -> int
  """Convert the user-entered date into timestamp."""
  # Support timestamp value such as opened>1437671476
  try:
    return int(val)
  except ValueError:
    pass

  # TODO(jrobbins): future: take timezones into account.
  # TODO(jrobbins): for now, explain to users that "today" is
  # actually now: the current time, not 12:01am in their timezone.
  # In fact, it is not very useful because everything in the system
  # happened before the current time.
  if val == 'today':
    return _CalculatePastDate(0, now=now)
  elif val.startswith('today-'):
    try:
      days_ago = int(val.split('-')[1])
    except ValueError:
      raise InvalidQueryError('Could not parse date: ' + val)
    return _CalculatePastDate(days_ago, now=now)

  try:
    if '/' in val:
      year, month, day = [int(x) for x in val.split('/')]
    elif '-' in val:
      year, month, day = [int(x) for x in val.split('-')]
    else:
      raise InvalidQueryError('Could not parse date: ' + val)
  except ValueError:
    raise InvalidQueryError('Could not parse date: ' + val)

  try:
    return int(time.mktime(datetime.datetime(year, month, day).timetuple()))
  except ValueError:
    raise InvalidQueryError('Could not parse date: ' + val)


def _CalculatePastDate(days_ago, now=None):
  # type: (int, int) -> int
  """Calculates the timestamp N days ago from now."""
  if now is None:
    now = int(time.time())
  ts = now - days_ago * 24 * 60 * 60
  return ts


def QueryToSubqueries(query):
  # type (str) -> Sequence[str]
  """Splits a query into smaller queries based on Monorail's search syntax.

  This function handles parsing parentheses and OR statements in Monorail's
  search syntax. By doing this parsing for OR statements and parentheses up
  front in FrontendSearchPipeline, we are able to convert complex queries
  with lots of ORs into smaller, more easily cacheable query terms.

  These outputted subqueries should collectively return the same query results
  as the initial input query without containing any ORs or parentheses,
  allowing later search layers to parse queries without worrying about ORs
  or parentheses.

  Some examples of possible queries and their expected output:

  - '(A OR B) (C OR D) OR (E OR F)' -> ['A C', 'A D', 'B C', 'B D', 'E', 'F']
  - '(A) OR (B)' -> ['A', 'B']
  - '(A ((C) OR (D OR (E OR F))))' -> ['A C', 'A D', 'A E', 'A F']

  Where A, B, C, D, etc could be any list of conjunctions. ie: "owner:me",
  "Pri=1", "hello world Hotlist=test", "label!=a11y", etc

  Note: Monorail implicitly ANDs any query terms separated by a space. For
  the most part, AND functionality is handled at a later layer in search
  processing. However, this case becomes important here when considering the
  fact that a prentheses group can either be ANDed or ORed with terms that
  surround it.

  The _MultiplySubqueries helper is used to AND the results of different
  groups together whereas concatenating lists is used to OR subqueries
  together.

  Args:
    query: The initial query that was sent to the search.

  Returns:
    List of query fragments to be independently processed as search terms.

  Raises:
    InvalidQueryError if parentheses are unmatched.
  """
  tokens = _ValidateAndTokenizeQuery(query)

  # Using an iterator allows us to keep our current loop position across
  # helpers. This makes recursion a lot easier.
  token_iterator = PeekIterator(tokens)

  subqueries = _ParseQuery(token_iterator)

  if not len(subqueries):
    # Several cases, such as an empty query or a query with only parentheses
    # will result in an empty set of subqueries. In these cases, we still want
    # to give the search pipeline a single empty query to process.
    return ['']

  return subqueries


def _ParseQuery(token_iterator):
  # type (Sequence[proto.ast_pb2.QueryToken]) -> Sequence[str]
  """Recursive helper to convert query tokens into a list of subqueries.

  Parses a Query based on the following grammar (EBNF):

    Query             := OrGroup { [OrOperator] OrGroup }
    OrGroup           := AndGroup { AndGroup }
    AndGroup          := Subquery | ParenthesesGroup
    ParenthesesGroup  := "(" Query ")"
    Subquery          := /.+/
    OrOperator        := " OR "

  An important nuance is that two groups can be next to each other, separated
  only by a word boundary (ie: space or parentheses). In this case, they are
  implicitly ANDed. In practice, because unparenthesized fragments ANDed by
  spaces are stored as single tokens, we only need to handle the AND case when
  a parentheses group is implicitly ANDed with an adjacent group.

  Order of precedence is implemented by recursing through OR groups before
  recursing through AND groups.

  Args:
    token_iterator: Iterator over a list of query tokens.

  Returns:
    List of query fragments to be processed as search terms.

  Raises:
    InvalidQueryError if tokens were inputted in a format that does not follow
    our search grammar.
  """
  subqueries = []
  try:
    if token_iterator.peek().token_type == OR:
      # Edge case: Ignore empty OR groups at the starte of a ParenthesesGroup.
      # ie: "(OR A)" will be processed as "A"
      next(token_iterator)

    subqueries = _ParseOrGroup(token_iterator)

    while token_iterator.peek().token_type == OR:
      # Consume the OR tokens without doing anything with it.
      next(token_iterator)

      next_token = token_iterator.peek()
      if next_token.token_type == RIGHT_PAREN:
        # Edge case: Ignore empty OR groups at the end of a ParenthesesGroup.
        # ie: "(A OR)" will be processed as "A"
        return subqueries

      next_subqueries = _ParseOrGroup(token_iterator)

      # Concatenate results of OR groups together.
      subqueries = subqueries + next_subqueries

  except StopIteration:
    pass
  # Return when we've reached the end of the string.
  return subqueries


def _ParseOrGroup(token_iterator):
  # type (Sequence[proto.ast_pb2.QueryToken]) -> Sequence[str]
  """Recursive helper to convert a single "OrGroup" into subqueries.

  An OrGroup here is based on the following grammar:

    Query             := OrGroup { [OrOperator] OrGroup }
    OrGroup           := AndGroup { AndGroup }
    AndGroup          := Subquery | ParenthesesGroup
    ParenthesesGroup  := "(" Query ")"
    Subquery          := /.+/
    OrOperator        := " OR "

  Args:
    token_iterator: Iterator over a list of query tokens.

  Returns:
    List of query fragments to be processed as search terms.

  Raises:
    InvalidQueryError if tokens were inputted in a format that does not follow
    our search grammar.
  """
  subqueries = _ParseAndGroup(token_iterator)

  try:
    # Iterate until there are no more AND groups left to see.
    # Subquery or left parentheses are the possible starts of an AndGroup.
    while (token_iterator.peek().token_type == SUBQUERY or
           token_iterator.peek().token_type == LEFT_PAREN):

      # Find subqueries from the next AND group.
      next_subqueries = _ParseAndGroup(token_iterator)

      # Multiply all results across AND groups together.
      subqueries = _MultiplySubqueries(subqueries, next_subqueries)
  except StopIteration:
    pass

  return subqueries


def _ParseAndGroup(token_iterator):
  # type (Sequence[proto.ast_pb2.QueryToken]) -> Sequence[str]
  """Recursive helper to convert a single "AndGroup" into subqueries.

  An OrGroup here is based on the following grammar:

    Query             := OrGroup { [OrOperator] OrGroup }
    OrGroup           := AndGroup { AndGroup }
    AndGroup          := Subquery | ParenthesesGroup
    ParenthesesGroup  := "(" Query ")"
    Subquery          := /.+/
    OrOperator        := " OR "

  Args:
    token_iterator: Iterator over a list of query tokens.

  Returns:
    List of query fragments to be processed as search terms.

  Raises:
    InvalidQueryError if tokens were inputted in a format that does not follow
    our search grammar.
  """
  try:
    token = next(token_iterator)
    if token.token_type == LEFT_PAREN:
      if token_iterator.peek().token_type == RIGHT_PAREN:
        # Don't recurse into the ParenthesesGroup if there's nothing inside.
        next(token_iterator)
        return []

      # Recurse into the ParenthesesGroup.
      subqueries = _ParseQuery(token_iterator)

      # Next token should be a right parenthesis.
      next(token_iterator)

      return subqueries
    elif token.token_type == SUBQUERY:
      return [token.value]
    else:
      # This should not happen if other QueryToSubqueries helpers are working
      # properly.
      raise InvalidQueryError('Inputted tokens do not follow grammar.')
  except StopIteration:
    pass
  return []


def _ValidateAndTokenizeQuery(query):
  # type: (str) -> Sequence[proto.ast_pb2.QueryToken]
  """Converts the input query into a set of tokens for easier parsing.

  Tokenizing the query string before parsing allows us to not have to as many
  string manipulations while parsing, which simplifies our later code.

  Args:
    query: Query to tokenize.

  Returns:
    List of Token objects for use in query processing.

  Raises:
    InvalidQueryError if parentheses are unmatched.
  """
  tokens = []  # Function result
  count = 0  # Used for checking if parentheses are balanced
  s = ''  # Records current string fragment. Cleared when a token is added.

  for ch in query:
    if ch == '(':
      count += 1

      # Add subquery from before we hit this parenthesis.
      tokens.extend(_TokenizeSubqueryOnOr(s))
      s = ''

      tokens.append(ast_pb2.QueryToken(token_type=LEFT_PAREN))
    elif ch == ')':
      count -= 1

      if count < 0:
        # More closing parentheses then open parentheses.
        raise InvalidQueryError('Search query has unbalanced parentheses.')

      # Add subquery from before we hit this parenthesis.
      tokens.extend(_TokenizeSubqueryOnOr(s))
      s = ''

      tokens.append(ast_pb2.QueryToken(token_type=RIGHT_PAREN))
    else:
      s += ch

  if count != 0:
    raise InvalidQueryError('Search query has unbalanced parentheses.')

  # Add any trailing tokens.
  tokens.extend(_TokenizeSubqueryOnOr(s))

  return tokens


def _TokenizeSubqueryOnOr(subquery):
  # type: (str) -> Sequence[proto.ast_pb2.QueryToken]
  """Helper to split a subquery by OR and convert the result into tokens.

  Args:
    subquery: A string without parentheses to tokenize.

  Returns:
    Tokens for the subquery with OR tokens separating query strings if
    applicable.
  """
  if len(subquery) == 0:
    return []

  result = []
  fragments = subquery.split(OR_SYMBOL)
  for f in fragments:
    # Interleave the string fragments with OR tokens.
    result.append(ast_pb2.QueryToken(token_type=SUBQUERY, value=f.strip()))
    result.append(ast_pb2.QueryToken(token_type=OR))

  # Remove trailing OR.
  result.pop()

  # Trim empty strings at the beginning or end. ie: if subquery is ' OR ',
  # we want the list to be ['OR'], not ['', 'OR', ''].
  if len(result) > 1 and result[0].value == '':
    result.pop(0)
  if len(result) > 1 and result[-1].value == '':
    result.pop()
  return result


def _MultiplySubqueries(a, b):
  # type: (Sequence[str], Sequence[str]) -> Sequence[str]
  """Helper to AND subqueries from two separate lists.

  Args:
    a: First list of subqueries.
    b: Second list of subqueries.

  Returns:
    List with n x m subqueries.
  """
  if not len(a):
    return b
  if not len(b):
    return a
  res = []
  for q1 in a:
    for q2 in b:
      # AND two subqueries together by concatenating them.
      query = (q1.strip() + ' ' + q2.strip()).strip()
      res.append(query)
  return res


class PeekIterator:
  """Simple iterator with peek() functionality.

  Used by QueryToSubqueries to maintain state easily across recursive calls.
  """

  def __init__(self, source):
    # type: (Sequence[Any])
    self.__source = source
    self.__i = 0

  def peek(self):
    # type: () -> Any
    """Gets the next value in the iterator without side effects.

    Returns:
      Next value in iterator.

    Raises:
      StopIteration if you're at the end of the iterator.
    """
    if self.__i >= len(self.__source):
      raise StopIteration
    return self.__source[self.__i]

  def __iter__(self):
    # type: () -> Sequence[Any]
    """Return self to make iterator iterable."""
    return self

  def __repr__(self):
    # type: () -> str
    """Allow logging current iterator value for debugging."""
    try:
      return str(self.peek())
    except StopIteration:
      pass
    return 'End of PeekIterator'

  def next(self):
    # type: () -> Any
    """Gets the next value in the iterator and increments pointer.

    Returns:
      Next value in iterator.

    Raises:
      StopIteration if you're at the end of the iterator.
    """
    if self.__i >= len(self.__source):
      raise StopIteration
    value = self.__source[self.__i]
    self.__i += 1
    return value


class Error(Exception):
  """Base exception class for this package."""
  pass


class InvalidQueryError(Error):
  """Error raised when an invalid query is requested."""
  pass
