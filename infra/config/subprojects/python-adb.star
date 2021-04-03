# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Definitions of presubmit/CQ for python-adb repo."""

load("//lib/build.star", "build")
load("//lib/infra.star", "infra")

infra.cq_group(
    name = "python-adb",
    repo = "https://chromium.googlesource.com/infra/luci/python-adb",
)
build.presubmit(
    name = "python-adb-presubmit",
    cq_group = "python-adb",
    repo_name = "python-adb",
    os = "Ubuntu",
)
