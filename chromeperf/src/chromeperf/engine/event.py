# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Event model for Task evaluation."""
import collections


class Event(
    collections.namedtuple('Event', ('type', 'target_task', 'payload'))):
  __slots__ = ()


def SelectEvent(target_task=None):
  return Event(type='select', target_task=target_task, payload={})