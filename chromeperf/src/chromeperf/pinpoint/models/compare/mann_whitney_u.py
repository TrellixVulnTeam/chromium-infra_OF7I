# Copyright 2016 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
"""Pure Python implementation of the Mann-Whitney U test.

This code is adapted from SciPy:
  https://github.com/scipy/scipy/blob/master/scipy/stats/stats.py
Which is provided under a BSD-style license.

There is also a JavaScript version in Catapult:
  https://github.com/catapult-project/catapult/blob/master/tracing/third_party/mannwhitneyu/mannwhitneyu.js
"""

import itertools
import math


def mann_whitney_u(x, y):
    """Computes the Mann-Whitney rank test on samples x and y.

    The distribution of U is approximately normal for large samples. This
    implementation uses the normal approximation, so it's recommended to have
    sample sizes > 20.
    """
    n1 = len(x)
    n2 = len(y)
    ranked = _rank_data(x + y)
    rankx = ranked[0:n1]  # get the x-ranks
    u1 = n1 * n2 + n1 * (n1 + 1) / 2.0 - sum(rankx)  # calc U for x
    u2 = n1 * n2 - u1  # remainder is U for y
    t = _tie_correction_factor(ranked)
    if t == 0:
        return 1.0
    sd = math.sqrt(t * n1 * n2 * (n1 + n2 + 1) / 12.0)

    mean_rank = n1 * n2 / 2.0 + 0.5
    big_u = max(u1, u2)

    z = (big_u - mean_rank) / sd
    return 2 * _norm_sf(abs(z))


def _rank_data(a):
    """Assigns ranks to data. Ties are given the mean of the ranks of the items.

    This is called "fractional ranking":
        https://en.wikipedia.org/wiki/Ranking
    """
    sorter = _arg_sort_reverse(a)
    ranked_min = [0] * len(sorter)
    for i, j in reversed(list(enumerate(sorter))):
        ranked_min[j] = i

    sorter = _arg_sort(a)
    ranked_max = [0] * len(sorter)
    for i, j in enumerate(sorter):
        ranked_max[j] = i

    return [1 + (x + y) / 2.0 for x, y in zip(ranked_min, ranked_max)]


def _arg_sort(a):
    """Returns the indices that would sort an array.

    Ties are given indices in ordinal order."""
    return sorted(range(len(a)), key=a.__getitem__)


def _arg_sort_reverse(a):
    """Returns the indices that would sort an array.

    Ties are given indices in reverse ordinal order."""
    return list(
        reversed(sorted(range(len(a)), key=a.__getitem__, reverse=True)))


def _tie_correction_factor(rankvals):
    """Tie correction factor for ties in the Mann-Whitney U test."""
    arr = sorted(rankvals)
    cnt = [len(list(group)) for _, group in itertools.groupby(arr)]
    size = len(arr)
    if size < 2:
        return 1.0
    else:
        return 1.0 - sum(x**3 - x for x in cnt) / float(size**3 - size)


def _norm_sf(x):
    """Survival function of the standard normal distribution. (1 - cdf)"""
    return (1 - math.erf(x / math.sqrt(2))) / 2
