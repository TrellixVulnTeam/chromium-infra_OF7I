# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from parameterized import parameterized

from testing_utils import testing

import experiments


class CheckInvalidExperimentNames(testing.AppengineTestCase):

  @parameterized.expand([
      ('cool.experiment'),
      ('single'),
      ('luci.use_realms'),
  ])
  def test_valid(self, exp_name):
    self.assertIsNone(experiments.check_invalid_name(exp_name))

  @parameterized.expand([
      ('bad!name'),
      ('not experiment'),
  ])
  def test_bad_name(self, exp_name):
    self.assertTrue(
        experiments.check_invalid_name(exp_name).startswith('does not match')
    )

  @parameterized.expand([
      ('luci.use_ralms'),
  ])
  def test_reserved_name(self, exp_name):
    self.assertTrue(
        experiments.check_invalid_name(exp_name)
        .startswith('unknown experiment has reserved prefix')
    )
