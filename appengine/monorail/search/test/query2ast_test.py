# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Tests for the query2ast module."""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import datetime
import time
import unittest
import mock

from proto import ast_pb2
from proto import tracker_pb2
from search import query2ast
from tracker import tracker_bizobj

BOOL = query2ast.BOOL
DATE = query2ast.DATE
NUM = query2ast.NUM
TXT = query2ast.TXT

SUBQUERY = query2ast.SUBQUERY
LEFT_PAREN = query2ast.LEFT_PAREN
RIGHT_PAREN = query2ast.RIGHT_PAREN
OR = query2ast.OR

BUILTIN_ISSUE_FIELDS = query2ast.BUILTIN_ISSUE_FIELDS
ANY_FIELD = query2ast.BUILTIN_ISSUE_FIELDS['any_field']

EQ = query2ast.EQ
NE = query2ast.NE
LT = query2ast.LT
GT = query2ast.GT
LE = query2ast.LE
GE = query2ast.GE
TEXT_HAS = query2ast.TEXT_HAS
NOT_TEXT_HAS = query2ast.NOT_TEXT_HAS
IS_DEFINED = query2ast.IS_DEFINED
IS_NOT_DEFINED = query2ast.IS_NOT_DEFINED
KEY_HAS = query2ast.KEY_HAS

MakeCond = ast_pb2.MakeCond
NOW = 1277762224


class QueryParsingUnitTest(unittest.TestCase):

  def setUp(self):
    self.project_id = 789
    self.default_config = tracker_bizobj.MakeDefaultProjectIssueConfig(
        self.project_id)

  def testParseUserQuery_OrClause(self):
    # an "OR" query, which should look like two separate simple querys
    # joined together by a pipe.
    ast = query2ast.ParseUserQuery(
        'ham OR fancy', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    conj1 = ast.conjunctions[0]
    conj2 = ast.conjunctions[1]
    self.assertEqual([MakeCond(TEXT_HAS, [ANY_FIELD], ['ham'], [])],
                     conj1.conds)
    self.assertEqual([MakeCond(TEXT_HAS, [ANY_FIELD], ['fancy'], [])],
                     conj2.conds)

  def testParseUserQuery_Words(self):
    # an "ORTerm" is actually anything appearing on either side of an
    # "OR" operator. So this could be thought of as "simple" query parsing.

    # a simple query with no spaces
    ast = query2ast.ParseUserQuery(
        'hamfancy', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    fulltext_cond = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['hamfancy'], []), fulltext_cond)

    # negative word
    ast = query2ast.ParseUserQuery(
        '-hamfancy', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    fulltext_cond = ast.conjunctions[0].conds[0]
    self.assertEqual(
        # note: not NOT_TEXT_HAS.
        MakeCond(NOT_TEXT_HAS, [ANY_FIELD], ['hamfancy'], []),
        fulltext_cond)

    # invalid fulltext term
    ast = query2ast.ParseUserQuery(
        'ham=fancy\\', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    self.assertEqual([], ast.conjunctions[0].conds)

    # an explicit "AND" query in the "featured" context
    warnings = []
    query2ast.ParseUserQuery(
        'ham AND fancy', 'label:featured', BUILTIN_ISSUE_FIELDS,
        self.default_config, warnings=warnings)
    self.assertEqual(
      ['The only supported boolean operator is OR (all capitals).'],
      warnings)

    # an implicit "AND" query
    ast = query2ast.ParseUserQuery(
        'ham fancy', '-label:deprecated', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    scope_cond1, ft_cond1, ft_cond2 = ast.conjunctions[0].conds
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['deprecated'], []),
        scope_cond1)
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['ham'], []), ft_cond1)
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['fancy'], []), ft_cond2)

    # Use word with non-operator prefix.
    word_with_non_op_prefix = '%stest' % query2ast.NON_OP_PREFIXES[0]
    ast = query2ast.ParseUserQuery(
        word_with_non_op_prefix, '', BUILTIN_ISSUE_FIELDS, self.default_config)
    fulltext_cond = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['"%s"' % word_with_non_op_prefix], []),
        fulltext_cond)

    # mix positive and negative words
    ast = query2ast.ParseUserQuery(
        'ham -fancy', '-label:deprecated', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    scope_cond1, ft_cond1, ft_cond2 = ast.conjunctions[0].conds
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['deprecated'], []),
        scope_cond1)
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['ham'], []), ft_cond1)
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [ANY_FIELD], ['fancy'], []), ft_cond2)

    # converts terms to lower case
    ast = query2ast.ParseUserQuery(
        'AmDude', '-label:deprecated', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    scope_cond1, fulltext_cond = ast.conjunctions[0].conds
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['deprecated'], []),
        scope_cond1)
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['amdude'], []), fulltext_cond)

  def testParseUserQuery_Phrases(self):
    # positive phrases
    ast = query2ast.ParseUserQuery(
        '"one two"', '-label:deprecated', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    scope_cond1, fulltext_cond = ast.conjunctions[0].conds
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['deprecated'], []),
        scope_cond1)
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['"one two"'], []), fulltext_cond)

    # negative phrases
    ast = query2ast.ParseUserQuery(
        '-"one two"', '-label:deprecated', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    scope_cond1, fulltext_cond = ast.conjunctions[0].conds
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['deprecated'], []),
        scope_cond1)
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [ANY_FIELD], ['"one two"'], []), fulltext_cond)

    # multiple phrases
    ast = query2ast.ParseUserQuery(
        '-"a b" "x y"', '-label:deprecated', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    scope_cond1, ft_cond1, ft_cond2 = ast.conjunctions[0].conds
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['deprecated'], []),
        scope_cond1)
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [ANY_FIELD], ['"a b"'], []), ft_cond1)
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['"x y"'], []), ft_cond2)

  def testParseUserQuery_CodeSyntaxThatWeNeedToCopeWith(self):
    # positive phrases
    ast = query2ast.ParseUserQuery(
        'Base::Tuple', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD],
                 ['"base::tuple"'], []),
        cond)

    # stuff we just ignore
    ast = query2ast.ParseUserQuery(
        ':: - -- .', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    self.assertEqual([], ast.conjunctions[0].conds)

  def testParseUserQuery_IsOperator(self):
    """Test is:open, is:spam, and is:blocked."""
    for keyword in ['open', 'spam', 'blocked']:
      ast = query2ast.ParseUserQuery(
          'is:' + keyword, '', BUILTIN_ISSUE_FIELDS, self.default_config)
      cond1 = ast.conjunctions[0].conds[0]
      self.assertEqual(
          MakeCond(EQ, [BUILTIN_ISSUE_FIELDS[keyword]], [], []),
          cond1)
      ast = query2ast.ParseUserQuery(
          '-is:' + keyword, '', BUILTIN_ISSUE_FIELDS, self.default_config)
      cond1 = ast.conjunctions[0].conds[0]
      self.assertEqual(
          MakeCond(NE, [BUILTIN_ISSUE_FIELDS[keyword]], [], []),
          cond1)

  def testParseUserQuery_HasOperator(self):
    # Search for issues with at least one attachment
    ast = query2ast.ParseUserQuery(
        'has:attachment', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(IS_DEFINED, [BUILTIN_ISSUE_FIELDS['attachment']], [], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-has:attachment', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(IS_NOT_DEFINED, [BUILTIN_ISSUE_FIELDS['attachment']], [], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'has=attachment', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(IS_DEFINED, [BUILTIN_ISSUE_FIELDS['attachment']], [], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-has=attachment', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(IS_NOT_DEFINED, [BUILTIN_ISSUE_FIELDS['attachment']], [], []),
        cond1)

    # Search for numeric fields for searches with 'has' prefix
    ast = query2ast.ParseUserQuery(
        'has:attachments', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(IS_DEFINED, [BUILTIN_ISSUE_FIELDS['attachments']], [], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-has:attachments', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(IS_NOT_DEFINED, [BUILTIN_ISSUE_FIELDS['attachments']],
                 [], []),
        cond1)

    # If it is not a field, look for any key-value label.
    ast = query2ast.ParseUserQuery(
        'has:Size', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(IS_DEFINED, [BUILTIN_ISSUE_FIELDS['label']], ['size'], []),
        cond1)

  def testParseUserQuery_Phase(self):
    ast = query2ast.ParseUserQuery(
        'gate:Canary,Stable', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['gate']],
                 ['canary', 'stable'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-gate:Canary,Stable', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [BUILTIN_ISSUE_FIELDS['gate']],
                 ['canary', 'stable'], []),
        cond1)

  def testParseUserQuery_Components(self):
    """Parse user queries for components"""
    ast = query2ast.ParseUserQuery(
        'component:UI', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['component']],
                 ['ui'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'Component:UI>AboutBox', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['component']],
                 ['ui>aboutbox'], []),
        cond1)

  def testParseUserQuery_OwnersReportersAndCc(self):
    """Parse user queries for owner:, reporter: and cc:."""
    ast = query2ast.ParseUserQuery(
        'owner:user', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['owner']],
                 ['user'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'owner:user@example.com', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['owner']],
                 ['user@example.com'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'owner=user@example.com', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['owner']],
                 ['user@example.com'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-reporter=user@example.com', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(NE, [BUILTIN_ISSUE_FIELDS['reporter']],
                 ['user@example.com'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'cc=user@example.com,user2@example.com', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['cc']],
                 ['user@example.com', 'user2@example.com'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'cc:user,user2', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['cc']],
                 ['user', 'user2'], []),
        cond1)

  def testParseUserQuery_SearchWithinFields(self):
    # Search for issues with certain filenames
    ast = query2ast.ParseUserQuery(
        'attachment:filename', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['attachment']],
                 ['filename'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-attachment:filename', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(NOT_TEXT_HAS, [BUILTIN_ISSUE_FIELDS['attachment']],
                 ['filename'], []),
        cond1)

    # Search for issues with a certain number of attachments
    ast = query2ast.ParseUserQuery(
        'attachments:2', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['attachments']],
                 ['2'], [2]),
        cond1)

    # Searches with '=' syntax
    ast = query2ast.ParseUserQuery(
        'attachment=filename', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['attachment']],
                 ['filename'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-attachment=filename', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(NE, [BUILTIN_ISSUE_FIELDS['attachment']],
                 ['filename'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'milestone=2009', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['label']], ['milestone-2009'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-milestone=2009', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(NE, [BUILTIN_ISSUE_FIELDS['label']], ['milestone-2009'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'milestone=2009-Q1', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['label']],
                 ['milestone-2009-q1'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        '-milestone=2009-Q1', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(NE, [BUILTIN_ISSUE_FIELDS['label']],
                 ['milestone-2009-q1'], []),
        cond1)

    # Searches with ':' syntax
    ast = query2ast.ParseUserQuery(
        'summary:foo', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS,
                 [BUILTIN_ISSUE_FIELDS['summary']], ['foo'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'summary:"greetings programs"', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS,
                 [BUILTIN_ISSUE_FIELDS['summary']], ['greetings programs'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'summary:"&#1234;"', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS,
                 [BUILTIN_ISSUE_FIELDS['summary']], ['&#1234;'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'priority:high', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(KEY_HAS,
                 [BUILTIN_ISSUE_FIELDS['label']], ['priority-high'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'type:security', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(KEY_HAS,
                 [BUILTIN_ISSUE_FIELDS['label']], ['type-security'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'label:priority-high', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS,
                 [BUILTIN_ISSUE_FIELDS['label']], ['priority-high'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'blockedon:other:123', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['blockedon']],
                 ['other:123'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'cost=-2', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['label']],
                 ['cost--2'], []),
        cond1)

    # Searches with ':' and an email domain only.
    ast = query2ast.ParseUserQuery(
        'reporter:@google.com', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS,
                 [BUILTIN_ISSUE_FIELDS['reporter']], ['@google.com'], []),
        cond1)

    # Search for issues in certain user hotlists.
    ast = query2ast.ParseUserQuery(
        'hotlist=gatsby@chromium.org:Hotlist1', '',
        BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(
            EQ, [BUILTIN_ISSUE_FIELDS['hotlist']],
            ['gatsby@chromium.org:hotlist1'], []),
        cond1)

    # Search for 'Hotlist' labels.
    ast = query2ast.ParseUserQuery(
        'hotlist:sublabel', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(KEY_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['hotlist-sublabel'], []),
        cond1)

  def testParseUserQuery_SearchWithinCustomFields(self):
    """Enums are treated as labels, other fields are kept as fields."""
    fd1 = tracker_bizobj.MakeFieldDef(
        1, self.project_id, 'Size', tracker_pb2.FieldTypes.ENUM_TYPE,
        'applic', 'applic', False, False, False, None, None, None, False, None,
        None, None, 'no_action', 'doc', False)
    fd2 = tracker_bizobj.MakeFieldDef(
        1, self.project_id, 'EstDays', tracker_pb2.FieldTypes.INT_TYPE,
        'applic', 'applic', False, False, False, None, None, None, False, None,
        None, None, 'no_action', 'doc', False)
    self.default_config.field_defs.extend([fd1, fd2])
    ast = query2ast.ParseUserQuery(
        'Size:Small EstDays>3', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    cond2 = ast.conjunctions[0].conds[1]
    self.assertEqual(
        MakeCond(KEY_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['size-small'], []),
        cond1)
    self.assertEqual(
        MakeCond(GT, [fd2], ['3'], [3]),
        cond2)

  @mock.patch('time.time', return_value=NOW)
  def testParseUserQuery_Approvals(self, _mock_time):
    """Test approval queries are parsed correctly."""
    fd1 = tracker_bizobj.MakeFieldDef(
        1, self.project_id, 'UIReview', tracker_pb2.FieldTypes.APPROVAL_TYPE,
        'applic', 'applic', False, False, False, None, None, None, False, None,
        None, None, 'no_action', 'doc', False)
    fd2 = tracker_bizobj.MakeFieldDef(
        2, self.project_id, 'EstDays', tracker_pb2.FieldTypes.INT_TYPE,
        'applic', 'applic', False, False, False, None, None, None, False, None,
        None, None, 'no_action', 'doc', False)
    fd3 = tracker_bizobj.MakeFieldDef(
        3, self.project_id, 'UXReview', tracker_pb2.FieldTypes.APPROVAL_TYPE,
        'applic', 'applic', False, False, False, None, None, None, False, None,
        None, None, 'no_action', 'doc', False)
    self.default_config.field_defs.extend([fd1, fd2, fd3])
    ast = query2ast.ParseUserQuery(
        'UXReview-approver:user1@mail.com,user2@mail.com UIReview:Approved '
        'UIReview-on>today-7', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    cond2 = ast.conjunctions[0].conds[1]
    cond3 = ast.conjunctions[0].conds[2]
    self.assertEqual(MakeCond(TEXT_HAS, [fd3],
                              ['user1@mail.com', 'user2@mail.com'], [],
                              key_suffix='-approver'), cond1)
    self.assertEqual(MakeCond(TEXT_HAS, [fd1], ['approved'], []), cond2)
    self.assertEqual(
        cond3,
        MakeCond(
            GT, [fd1], [], [query2ast._CalculatePastDate(7, NOW)],
            key_suffix='-on'))

  def testParseUserQuery_PhaseFields(self):
    fd = tracker_bizobj.MakeFieldDef(
        1, self.project_id, 'EstDays', tracker_pb2.FieldTypes.INT_TYPE,
        'applic', 'applic', False, False, False, None, None, None, False, None,
        None, None, 'no_action', 'doc', False, is_phase_field=True)
    self.default_config.field_defs.append(fd)
    ast = query2ast.ParseUserQuery(
        'UXReview.EstDays>3', '', BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(GT, [fd], ['3'], [3], phase_name='uxreview'),
        cond1)

  def testParseUserQuery_QuickOr(self):
    # quick-or searches
    ast = query2ast.ParseUserQuery(
        'milestone:2008,2009,2010', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(KEY_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['milestone-2008', 'milestone-2009', 'milestone-2010'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'label:milestone-2008,milestone-2009,milestone-2010', '',
        BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(TEXT_HAS, [BUILTIN_ISSUE_FIELDS['label']],
                 ['milestone-2008', 'milestone-2009', 'milestone-2010'], []),
        cond1)

    ast = query2ast.ParseUserQuery(
        'milestone=2008,2009,2010', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['label']],
                 ['milestone-2008', 'milestone-2009', 'milestone-2010'], []),
        cond1)

    # Duplicated and trailing commas are ignored.
    ast = query2ast.ParseUserQuery(
        'milestone=2008,,2009,2010,', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['label']],
                 ['milestone-2008', 'milestone-2009', 'milestone-2010'], []),
        cond1)

  def testParseUserQuery_Dates(self):
    # query with a daterange
    ast = query2ast.ParseUserQuery(
        'modified>=2009-5-12', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    ts1 = int(time.mktime(datetime.datetime(2009, 5, 12).timetuple()))
    self.assertEqual(
        MakeCond(GE, [BUILTIN_ISSUE_FIELDS['modified']], [], [ts1]), cond1)

    # query with quick-or
    ast = query2ast.ParseUserQuery(
        'modified=2009-5-12,2009-5-13', '', BUILTIN_ISSUE_FIELDS,
        self.default_config)
    cond1 = ast.conjunctions[0].conds[0]
    ts1 = int(time.mktime(datetime.datetime(2009, 5, 12).timetuple()))
    ts2 = int(time.mktime(datetime.datetime(2009, 5, 13).timetuple()))
    self.assertEqual(
        MakeCond(EQ, [BUILTIN_ISSUE_FIELDS['modified']], [], [ts1, ts2]), cond1)

    # query with multiple dateranges
    ast = query2ast.ParseUserQuery(
        'modified>=2009-5-12 opened<2008/1/1', '',
        BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1, cond2 = ast.conjunctions[0].conds
    ts1 = int(time.mktime(datetime.datetime(2009, 5, 12).timetuple()))
    self.assertEqual(
        MakeCond(GE, [BUILTIN_ISSUE_FIELDS['modified']], [], [ts1]), cond1)
    ts2 = int(time.mktime(datetime.datetime(2008, 1, 1).timetuple()))
    self.assertEqual(
        MakeCond(LT, [BUILTIN_ISSUE_FIELDS['opened']], [], [ts2]), cond2)

    # query with multiple dateranges plus a search term
    ast = query2ast.ParseUserQuery(
        'one two modified>=2009-5-12 opened<2008/1/1', '',
        BUILTIN_ISSUE_FIELDS, self.default_config)
    ft_cond1, ft_cond2, cond1, cond2 = ast.conjunctions[0].conds
    ts1 = int(time.mktime(datetime.datetime(2009, 5, 12).timetuple()))
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['one'], []), ft_cond1)
    self.assertEqual(
        MakeCond(TEXT_HAS, [ANY_FIELD], ['two'], []), ft_cond2)
    self.assertEqual(
        MakeCond(GE, [BUILTIN_ISSUE_FIELDS['modified']], [], [ts1]), cond1)
    ts2 = int(time.mktime(datetime.datetime(2008, 1, 1).timetuple()))
    self.assertEqual(
        MakeCond(LT, [BUILTIN_ISSUE_FIELDS['opened']], [], [ts2]), cond2)

    # query with a date field compared to "today"
    ast = query2ast.ParseUserQuery(
        'modified<today', '', BUILTIN_ISSUE_FIELDS,
        self.default_config, now=NOW)
    cond1 = ast.conjunctions[0].conds[0]
    ts1 = query2ast._CalculatePastDate(0, now=NOW)
    self.assertEqual(MakeCond(LT, [BUILTIN_ISSUE_FIELDS['modified']],
                              [], [ts1]),
                     cond1)

    # query with a daterange using today-N alias
    ast = query2ast.ParseUserQuery(
        'modified>=today-13', '', BUILTIN_ISSUE_FIELDS,
        self.default_config, now=NOW)
    cond1 = ast.conjunctions[0].conds[0]
    ts1 = query2ast._CalculatePastDate(13, now=NOW)
    self.assertEqual(MakeCond(GE, [BUILTIN_ISSUE_FIELDS['modified']],
                              [], [ts1]),
                     cond1)

    ast = query2ast.ParseUserQuery(
        'modified>today-13', '', BUILTIN_ISSUE_FIELDS, self.default_config,
        now=NOW)
    cond1 = ast.conjunctions[0].conds[0]
    ts1 = query2ast._CalculatePastDate(13, now=NOW)
    self.assertEqual(MakeCond(GT, [BUILTIN_ISSUE_FIELDS['modified']],
                              [], [ts1]),
                     cond1)

    # query with multiple old date query terms.
    ast = query2ast.ParseUserQuery(
        'modified-after:2009-5-12 opened-before:2008/1/1 '
        'closed-after:2007-2-1', '',
        BUILTIN_ISSUE_FIELDS, self.default_config)
    cond1, cond2, cond3 = ast.conjunctions[0].conds
    ts1 = int(time.mktime(datetime.datetime(2009, 5, 12).timetuple()))
    self.assertEqual(
        MakeCond(GT, [BUILTIN_ISSUE_FIELDS['modified']], [], [ts1]), cond1)
    ts2 = int(time.mktime(datetime.datetime(2008, 1, 1).timetuple()))
    self.assertEqual(
        MakeCond(LT, [BUILTIN_ISSUE_FIELDS['opened']], [], [ts2]), cond2)
    ts3 = int(time.mktime(datetime.datetime(2007, 2, 1).timetuple()))
    self.assertEqual(
        MakeCond(GT, [BUILTIN_ISSUE_FIELDS['closed']], [], [ts3]), cond3)

  def testCalculatePastDate(self):
    ts1 = query2ast._CalculatePastDate(0, now=NOW)
    self.assertEqual(NOW, ts1)

    ts2 = query2ast._CalculatePastDate(13, now=NOW)
    self.assertEqual(ts2, NOW - 13 * 24 * 60 * 60)

    # Try it once with time.time() instead of a known timestamp.
    ts_system_clock = query2ast._CalculatePastDate(13)
    self.assertTrue(ts_system_clock < int(time.time()))

  def testParseUserQuery_BadDates(self):
    bad_dates = ['today-13h', 'yesterday', '2/2', 'm/y/d',
                 '99/99/1999', '0-0-0']
    for val in bad_dates:
      with self.assertRaises(query2ast.InvalidQueryError) as cm:
        query2ast.ParseUserQuery(
            'modified>=' + val, '', BUILTIN_ISSUE_FIELDS,
            self.default_config)
      self.assertEqual('Could not parse date: ' + val, cm.exception.message)

  def testQueryToSubqueries_BasicQuery(self):
    self.assertEqual(['owner:me'], query2ast.QueryToSubqueries('owner:me'))

  def testQueryToSubqueries_EmptyQuery(self):
    self.assertEqual([''], query2ast.QueryToSubqueries(''))

  def testQueryToSubqueries_UnmatchedParenthesesThrowsError(self):
    with self.assertRaises(query2ast.InvalidQueryError):
      self.assertEqual(['Pri=1'], query2ast.QueryToSubqueries('Pri=1))'))
    with self.assertRaises(query2ast.InvalidQueryError):
      self.assertEqual(
          ['label:Hello'], query2ast.QueryToSubqueries('((label:Hello'))

    with self.assertRaises(query2ast.InvalidQueryError):
      self.assertEqual(
          ['owner:me'], query2ast.QueryToSubqueries('((((owner:me)))'))

    with self.assertRaises(query2ast.InvalidQueryError):
      self.assertEqual(
          ['test=What'], query2ast.QueryToSubqueries('(((test=What))))'))

  def testQueryToSubqueries_IgnoresEmptyGroups(self):
    self.assertEqual([''], query2ast.QueryToSubqueries('()(()(()))()()'))

    self.assertEqual(
        ['owner:me'], query2ast.QueryToSubqueries('()(()owner:me)()()'))

  def testQueryToSubqueries_BasicOr(self):
    self.assertEqual(
        ['owner:me', 'status:New', 'Pri=1'],
        query2ast.QueryToSubqueries('owner:me OR status:New OR Pri=1'))

  def testQueryToSubqueries_OrAtStartOrEnd(self):
    self.assertEqual(
        ['owner:me OR'], query2ast.QueryToSubqueries('owner:me OR'))

    self.assertEqual(
        ['OR owner:me'], query2ast.QueryToSubqueries('OR owner:me'))

  def testQueryToSubqueries_BasicParentheses(self):
    self.assertEqual(
        ['owner:me status:New'],
        query2ast.QueryToSubqueries('owner:me (status:New)'))

    self.assertEqual(
        ['owner:me status:New'],
        query2ast.QueryToSubqueries('(owner:me) status:New'))

    self.assertEqual(
        ['owner:me status:New'],
        query2ast.QueryToSubqueries('((owner:me) (status:New))'))

  def testQueryToSubqueries_ParenthesesWithOr(self):
    self.assertEqual(
        ['Pri=1 owner:me', 'Pri=1 status:New'],
        query2ast.QueryToSubqueries('Pri=1 (owner:me OR status:New)'))

    self.assertEqual(
        ['owner:me component:OhNo', 'status:New component:OhNo'],
        query2ast.QueryToSubqueries('(owner:me OR status:New) component:OhNo'))

  def testQueryToSubqueries_ParenthesesWithOr_Multiple(self):
    self.assertEqual(
        [
            'Pri=1 test owner:me', 'Pri=1 test status:New',
            'Pri=2 test owner:me', 'Pri=2 test status:New'
        ],
        query2ast.QueryToSubqueries(
            '(Pri=1 OR Pri=2)(test (owner:me OR status:New))'))

  def testQueryToSubqueries_OrNextToParentheses(self):
    self.assertEqual(['A', 'B'], query2ast.QueryToSubqueries('(A) OR (B)'))

    self.assertEqual(
        ['A B', 'A C E', 'A D E'],
        query2ast.QueryToSubqueries('A (B OR (C OR D) E)'))

    self.assertEqual(
        ['A B C', 'A B D', 'A E'],
        query2ast.QueryToSubqueries('A (B (C OR D) OR E)'))

  def testQueryToSubqueries_ExtraSpaces(self):
    self.assertEqual(
        ['A', 'B'], query2ast.QueryToSubqueries(' ( A )   OR  ( B ) '))

    self.assertEqual(
        ['A B', 'A C E', 'A D E'],
        query2ast.QueryToSubqueries(' A  ( B   OR   ( C  OR  D )  E )'))

  def testQueryToSubqueries_OrAtEndOfParentheses(self):
    self.assertEqual(['A B'], query2ast.QueryToSubqueries('(A OR )(B)'))
    self.assertEqual(
        ['A B', 'A C'], query2ast.QueryToSubqueries('( OR A)(B OR C)'))
    self.assertEqual(
        ['A B', 'A C'], query2ast.QueryToSubqueries(' OR A (B OR C)'))

  def testQueryToSubqueries_EmptyOrGroup(self):
    self.assertEqual(
        ['A C', 'C', 'B C'], query2ast.QueryToSubqueries('(A OR  OR B)(C)'))

  def testParseQuery_Basic(self):
    self.assertEqual(
        [
            'owner:me',
        ],
        query2ast._ParseQuery(
            query2ast.PeekIterator(
                [ast_pb2.QueryToken(token_type=SUBQUERY, value='owner:me')])))

  def testParseQuery_Complex(self):
    self.assertEqual(
        [
            'owner:me',
            'Pri=1',
            'label=test',
        ],
        query2ast._ParseQuery(
            query2ast.PeekIterator(
                [
                    ast_pb2.QueryToken(token_type=SUBQUERY, value='owner:me'),
                    ast_pb2.QueryToken(token_type=OR),
                    ast_pb2.QueryToken(token_type=LEFT_PAREN),
                    ast_pb2.QueryToken(token_type=SUBQUERY, value='Pri=1'),
                    ast_pb2.QueryToken(token_type=RIGHT_PAREN),
                    ast_pb2.QueryToken(token_type=OR),
                    ast_pb2.QueryToken(token_type=SUBQUERY, value='label=test'),
                ])))

  def testParseOrGroup_Basic(self):
    self.assertEqual(
        [
            'owner:me',
        ],
        query2ast._ParseOrGroup(
            query2ast.PeekIterator(
                [ast_pb2.QueryToken(token_type=SUBQUERY, value='owner:me')])))

  def testParseOrGroup_TwoAdjacentAndGroups(self):
    self.assertEqual(
        [
            'owner:me Pri=1',
            'owner:me label=test',
        ],
        query2ast._ParseOrGroup(
            query2ast.PeekIterator(
                [
                    ast_pb2.QueryToken(token_type=SUBQUERY, value='owner:me'),
                    ast_pb2.QueryToken(token_type=LEFT_PAREN),
                    ast_pb2.QueryToken(token_type=SUBQUERY, value='Pri=1'),
                    ast_pb2.QueryToken(token_type=OR),
                    ast_pb2.QueryToken(token_type=SUBQUERY, value='label=test'),
                    ast_pb2.QueryToken(token_type=RIGHT_PAREN),
                ])))

  def testParseAndGroup_Subquery(self):
    self.assertEqual(
        [
            'owner:me',
        ],
        query2ast._ParseAndGroup(
            query2ast.PeekIterator(
                [ast_pb2.QueryToken(token_type=SUBQUERY, value='owner:me')])))

  def testParseAndGroup_ParenthesesGroup(self):
    self.assertEqual(
        [
            'owner:me',
            'Pri=1',
        ],
        query2ast._ParseAndGroup(
            query2ast.PeekIterator(
                [
                    ast_pb2.QueryToken(token_type=LEFT_PAREN),
                    ast_pb2.QueryToken(token_type=SUBQUERY, value='owner:me'),
                    ast_pb2.QueryToken(token_type=OR),
                    ast_pb2.QueryToken(token_type=SUBQUERY, value='Pri=1'),
                    ast_pb2.QueryToken(token_type=RIGHT_PAREN),
                ])))

  def testParseAndGroup_Empty(self):
    self.assertEqual([], query2ast._ParseAndGroup(query2ast.PeekIterator([])))

  def testParseAndGroup_InvalidTokens(self):
    with self.assertRaises(query2ast.InvalidQueryError):
      query2ast._ParseAndGroup(
          query2ast.PeekIterator(
              [
                  ast_pb2.QueryToken(token_type=OR),
                  ast_pb2.QueryToken(token_type=SUBQUERY, value='Pri=1'),
                  ast_pb2.QueryToken(token_type=RIGHT_PAREN),
              ]))

    with self.assertRaises(query2ast.InvalidQueryError):
      query2ast._ParseAndGroup(
          query2ast.PeekIterator(
              [
                  ast_pb2.QueryToken(token_type=RIGHT_PAREN),
                  ast_pb2.QueryToken(token_type=OR),
                  ast_pb2.QueryToken(token_type=SUBQUERY, value='Pri=1'),
              ]))

  def testValidateAndTokenizeQuery_Basic(self):
    self.assertEqual(
        [
            ast_pb2.QueryToken(token_type=LEFT_PAREN),
            ast_pb2.QueryToken(token_type=SUBQUERY, value='owner:me'),
            ast_pb2.QueryToken(token_type=OR),
            ast_pb2.QueryToken(token_type=SUBQUERY, value='Pri=1'),
            ast_pb2.QueryToken(token_type=RIGHT_PAREN),
        ], query2ast._ValidateAndTokenizeQuery('(owner:me OR Pri=1)'))

  def testValidateAndTokenizeQuery_UnmatchedParentheses(self):
    with self.assertRaises(query2ast.InvalidQueryError):
      query2ast._ValidateAndTokenizeQuery('(owner:me')

    with self.assertRaises(query2ast.InvalidQueryError):
      query2ast._ValidateAndTokenizeQuery('owner:me)')

    with self.assertRaises(query2ast.InvalidQueryError):
      query2ast._ValidateAndTokenizeQuery('(()owner:me))')

    with self.assertRaises(query2ast.InvalidQueryError):
      query2ast._ValidateAndTokenizeQuery('(()owner:me)())')

  def testTokenizeSubqueryOnOr_NoOrOperator(self):
    self.assertEqual(
        [ast_pb2.QueryToken(token_type=SUBQUERY, value='owner:me')],
        query2ast._TokenizeSubqueryOnOr('owner:me'))

  def testTokenizeSubqueryOnOr_BasicOrOperator(self):
    self.assertEqual(
        [
            ast_pb2.QueryToken(token_type=SUBQUERY, value='A'),
            ast_pb2.QueryToken(token_type=OR),
            ast_pb2.QueryToken(token_type=SUBQUERY, value='B'),
            ast_pb2.QueryToken(token_type=OR),
            ast_pb2.QueryToken(token_type=SUBQUERY, value='C'),
        ], query2ast._TokenizeSubqueryOnOr('A OR B OR C'))

  def testTokenizeSubqueryOnOr_EmptyOrOperator(self):
    self.assertEqual(
        [ast_pb2.QueryToken(token_type=OR)],
        query2ast._TokenizeSubqueryOnOr(' OR '))

    self.assertEqual(
        [
            ast_pb2.QueryToken(token_type=SUBQUERY, value='A'),
            ast_pb2.QueryToken(token_type=OR),
        ], query2ast._TokenizeSubqueryOnOr('A OR '))

  def testMultiplySubqueries_Basic(self):
    self.assertEqual(
        ['owner:me Pri=1', 'owner:me Pri=2', 'test Pri=1', 'test Pri=2'],
        query2ast._MultiplySubqueries(['owner:me', 'test'], ['Pri=1', 'Pri=2']))

  def testMultiplySubqueries_OneEmpty(self):
    self.assertEqual(
        ['Pri=1', 'Pri=2'],
        query2ast._MultiplySubqueries([], ['Pri=1', 'Pri=2']))
    self.assertEqual(
        ['Pri=1', 'Pri=2'],
        query2ast._MultiplySubqueries([''], ['Pri=1', 'Pri=2']))

    self.assertEqual(
        ['Pri=1', 'Pri=2'],
        query2ast._MultiplySubqueries(['Pri=1', 'Pri=2'], []))
    self.assertEqual(
        ['Pri=1', 'Pri=2'],
        query2ast._MultiplySubqueries(['Pri=1', 'Pri=2'], ['']))

  def testPeekIterator_Basic(self):
    iterator = query2ast.PeekIterator([1, 2, 3])

    self.assertEqual(1, iterator.peek())
    self.assertEqual(1, iterator.next())

    self.assertEqual(2, iterator.next())

    self.assertEqual(3, iterator.peek())
    self.assertEqual(3, iterator.next())

    with self.assertRaises(StopIteration):
      iterator.peek()

    with self.assertRaises(StopIteration):
      iterator.next()
