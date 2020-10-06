# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Helpers for defining LED Swarming permissions."""

def _users(*, groups, task_realm, pool_realm = None):
    """Defines permissions to directly launch tasks on Swarming.

    Args:
      groups: a list of groups to grant permissions to run tasks.
      task_realm: a realm or a list of realms with to-be-launched tasks.
      pool_realm: a realm with ACLs for the Swarming pool to run tasks in or
        None if the pool ACLs are defined in some other project.
    """
    luci.binding(
        realm = task_realm,
        roles = "role/swarming.taskTriggerer",
        groups = groups,
    )
    if pool_realm:
        luci.binding(
            realm = pool_realm,
            roles = "role/swarming.poolUser",
            groups = groups,
        )

led = struct(
    users = _users,
)
