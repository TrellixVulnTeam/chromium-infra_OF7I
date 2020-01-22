# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd

"""Functions to help rerank issues in a lit.

This file contains methods that implement a reranking algorithm for
issues in a list.
"""

from __future__ import division
from __future__ import print_function
from __future__ import absolute_import

import sys

from framework import exceptions

MAX_RANKING = sys.maxint
MIN_RANKING = 0

def GetHotlistRerankChanges(hotlist_items, moved_issue_ids, target_position):
  # type: (int, Sequence[int], int) -> Collection[Tuple[int, int]]
  """Computes the changed ranks from a reranking of HotlistItems.

  Args:
    hotlist_items: The list of HotlistItems to rerank.
    moved_issue_ids: A list of issue IDs to be moved together, in the order
      they should have the reranking.
    target_position: The index, starting at 0, of the new position the
      first issue in moved_issue_ids should occupy in the updated ordering.
      Therefore this value cannot be greater than
      (len(hotlist.items) - len(moved_issue_ids)).

  Returns:
    A list of [(issue_id, rank), ...] of HotlistItems that need to be updated.

  Raises:
    InputException: If the target_position or moved_issue_ids are not valid.
  """
  # Sort hotlist items by rank.
  sorted_hotlist_items = sorted(hotlist_items, key=lambda item: item.rank)
  hotlist_issue_ids = [item.issue_id for item in sorted_hotlist_items]
  if not set(moved_issue_ids).issubset(set(hotlist_issue_ids)):
    raise exceptions.InputException('An issue to move is not in the hotlist')
  unmoved_hotlist_items = [
      item for item in sorted_hotlist_items
      if item.issue_id not in moved_issue_ids]
  if target_position < 0:
    raise exceptions.InputException(
        'given `target_position`: %d, must be non-negative')
  if target_position > len(unmoved_hotlist_items):
    raise exceptions.InputException(
        '`target_position` %d is higher than maximum allowed (%d) for'
        'moving %d items in a hotlist with %d items total.' % (
            target_position, len(unmoved_hotlist_items),
            len(moved_issue_ids), len(sorted_hotlist_items)))
  lower, higher = [], []
  for i, item in enumerate(unmoved_hotlist_items):
    item_tuple = (item.issue_id, item.rank)
    if i < target_position:
      lower.append(item_tuple)
    else:
      higher.append(item_tuple)

  return GetInsertRankings(lower, higher, moved_issue_ids)

def GetInsertRankings(lower, higher, moved_ids):
  """Compute rankings for moved_ids to insert between the
  lower and higher rankings

  Args:
    lower: a list of [(id, rank),...] of blockers that should have
      a lower rank than the moved issues. Should be sorted from highest
      to lowest rank.
    higher: a list of [(id, rank),...] of blockers that should have
      a higher rank than the moved issues. Should be sorted from highest
      to lowest rank.
    moved_ids: a list of global IDs for issues to re-rank.

  Returns:
    a list of [(id, rank),...] of blockers that need to be updated. rank
    is the new rank of the issue with the specified id.
  """
  if lower:
    lower_rank = lower[-1][1]
  else:
    lower_rank = MIN_RANKING

  if higher:
    higher_rank = higher[0][1]
  else:
    higher_rank = MAX_RANKING

  slot_count = higher_rank - lower_rank - 1
  if slot_count >= len(moved_ids):
    new_ranks = _DistributeRanks(lower_rank, higher_rank, len(moved_ids))
    return list(zip(moved_ids, new_ranks))
  else:
    new_lower, new_higher, new_moved_ids = _ResplitRanks(
        lower, higher, moved_ids)
    if not new_moved_ids:
      return None
    else:
      return GetInsertRankings(new_lower, new_higher, new_moved_ids)


def _DistributeRanks(low, high, rank_count):
  """Compute evenly distributed ranks in a range"""
  bucket_size = (high - low) // rank_count
  first_rank = low + (bucket_size + 1) // 2
  return list(range(first_rank, high, bucket_size))


def _ResplitRanks(lower, higher, moved_ids):
  if not (lower or higher):
    return None, None, None

  if not lower:
    take_from = 'higher'
  elif not higher:
    take_from = 'lower'
  else:
    next_lower = lower[-2][1] if len(lower) >= 2 else MIN_RANKING
    next_higher = higher[1][1] if len(higher) >= 2 else MAX_RANKING
    if (lower[-1][1] - next_lower) > (next_higher - higher[0][1]):
      take_from = 'lower'
    else:
      take_from = 'higher'

  if take_from == 'lower':
    return (lower[:-1], higher, [lower[-1][0]] + moved_ids)
  else:
    return (lower, higher[1:], moved_ids + [higher[0][0]])
