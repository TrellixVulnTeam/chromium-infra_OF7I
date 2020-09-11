# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections


# TODO(abennetts): Move this to use protobufs instead.
class States(object):
  __slots__ = ()
  PENDING = 0
  ONGOING = 1
  COMPLETED = 2
  FAILED = 3


# TODO(abennetts): Move this to use protobufs instead.
class Job(collections.namedtuple('Job', ('id', 'state'))):
  __slots__ = ()
