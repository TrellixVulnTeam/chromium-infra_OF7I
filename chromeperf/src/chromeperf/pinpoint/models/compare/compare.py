# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# TODO: replace most of this package with SciPy.

import dataclasses

import scipy.stats

from chromeperf.pinpoint import comparison_pb2

from chromeperf.pinpoint.models.compare import thresholds

DIFFERENT = 'DIFFERENT'
SAME = 'SAME'
UNKNOWN = 'UNKNOWN'
PENDING = 'PENDING'


@dataclasses.dataclass
class ComparisonResult:
    result: str  # one of DIFFERENT, PENDING, SAME, UNKNOWN.
    p_value: float
    low_threshold: float
    high_threshold: float

    def to_proto(self) -> comparison_pb2.Comparison:
        return comparison_pb2.Comparison(
            result=comparison_pb2.Comparison.CompareResult.Value(self.result),
            p_value=self.p_value,
            low_threshold=self.low_threshold,
            high_threshold=self.high_threshold)

    @classmethod
    def from_proto(cls, proto: comparison_pb2.Comparison):
        if proto is None: return None
        if (proto.result == comparison_pb2.Comparison.CompareResult.COMPARE_RESULT_UNSPECIFIED
                and proto.p_value == 0.0 and proto.low_threshold == 0.0
                and proto.high_threshold == 0.0):
            return None
        return cls(
            result=comparison_pb2.Comparison.CompareResult.Name(proto.result),
            p_value=proto.p_value,
            low_threshold=proto.low_threshold,
            high_threshold=proto.high_threshold)


def _mannwhitneyu(values_a, values_b):
  try:
      return scipy.stats.mannwhitneyu(values_a, values_b, use_continuity=True,
                                      alternative='two-sided')[1]
  except ValueError:
      # Catch 'All numbers are identical in mannwhitneyu' errors.  For our
      # purposes that means p-value 1.0.
      return 1.0


# TODO(https://crbug.com/1051710): Make this return all the values useful in
# decision making (and display).
def compare(values_a, values_b, attempt_count, mode, magnitude
            ) -> ComparisonResult:
    """Decide whether two samples are the same, different, or unknown.

    Arguments:
      values_a: A list of sortable values. They don't need to be numeric.
      values_b: A list of sortable values. They don't need to be numeric.
      attempt_count: The average number of attempts made.
      mode: 'functional' or 'performance'. We use different significance
        thresholds for each type.
      magnitude: An estimate of the size of differences to look for. We need
        more values to find smaller differences. If mode is 'functional', this
        is the failure rate, a float between 0 and 1. If mode is 'performance',
        this is a multiple of the interquartile range (IQR).

    Returns:
      A tuple `ComparisonResults` which contains the following elements:
        * result: one of the following values:
            DIFFERENT: The samples are unlikely to come from the same
                       distribution, and are therefore likely different. Reject
                       the null hypothesis.
            SAME     : The samples are unlikely to come from distributions that
                       differ by the given magnitude. Cannot reject the null
                       hypothesis.
            UNKNOWN  : Not enough evidence to reject either hypothesis. We
                       should collect more data before making a final decision.
        * p_value: the consolidated p-value for the statistical tests used in
                   the implementation.
        * low_threshold: the `alpha` where if the p-value is lower means we can
                         reject the null hypothesis.
        * high_threshold: the `alpha` where if the p-value is lower means we
                          need more information to make a definitive judgement.
    """
    low_threshold = thresholds.low_threshold()
    high_threshold = thresholds.high_threshold(mode, magnitude, attempt_count)

    if not (len(values_a) > 0 and len(values_b) > 0):
        # A sample has no values in it.
        return ComparisonResult(UNKNOWN, None, low_threshold, high_threshold)

    # MWU is bad at detecting changes in variance, and K-S is bad with discrete
    # distributions. So use both. We want low p-values for the below examples.
    #        a                     b               MWU(a, b)  KS(a, b)
    # [0]*20            [0]*15+[1]*5                0.0097     0.4973
    # range(10, 30)     range(10)+range(30, 40)     0.4946     0.0082
    p_value = min(
        scipy.stats.ks_2samp(values_a, values_b)[1],
        _mannwhitneyu(values_a, values_b))

    if p_value <= low_threshold:
        # The p-value is less than the significance level. Reject the null
        # hypothesis.
        return ComparisonResult(DIFFERENT, p_value, low_threshold,
                                high_threshold)

    if p_value <= thresholds.high_threshold(mode, magnitude, attempt_count):
        # The p-value is not less than the significance level, but it's small
        # enough to be suspicious. We'd like to investigate more closely.
        return ComparisonResult(UNKNOWN, p_value, low_threshold, high_threshold)

    # The p-value is quite large. We're not suspicious that the two samples
    # might come from different distributions, and we don't care to investigate
    # more.
    return ComparisonResult(SAME, p_value, low_threshold, high_threshold)
