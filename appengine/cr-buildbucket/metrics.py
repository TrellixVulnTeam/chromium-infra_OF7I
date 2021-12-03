# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import collections

import gae_ts_mon

from legacy import api_common
import buildtags
import config


BuildMetricField = collections.namedtuple(
    'BuildMetricField',
    [
        'value_type',  # e.g. str
        'field_type',  # e.g. gae_ts_mon.StringField
        'getter',  # function (model.Build) => value
    ]
)


def str_attr_field(attr):
  return BuildMetricField(
      value_type=str,
      field_type=gae_ts_mon.StringField,
      getter=lambda b: getattr(b, attr),
  )

_BUILD_FIELDS = {
    'bucket': BuildMetricField(
        value_type=str,
        field_type=gae_ts_mon.StringField,
        getter=lambda b: api_common.legacy_bucket_name(b.bucket_id, b.is_luci),
    ),
    'builder': BuildMetricField(
        value_type=str,
        field_type=gae_ts_mon.StringField,
        getter=lambda b: b.proto.builder.builder,
    ),
    'canary': BuildMetricField(
        value_type=bool,
        field_type=gae_ts_mon.BooleanField,
        getter=lambda b: b.canary,
    ),
    'cancelation_reason': str_attr_field('cancelation_reason'),
    'failure_reason': str_attr_field('failure_reason'),
    'result': str_attr_field('result'),
    'status': str_attr_field('status_legacy'),
    'user_agent': BuildMetricField(
        value_type=str,
        field_type=gae_ts_mon.StringField,
        getter=lambda b: (
            dict(buildtags.parse(t) for t in b.tags).get('user_agent')),
    ),
}
_METRIC_PREFIX_PROD = 'buildbucket/builds/'
_METRIC_PREFIX_EXPERIMENTAL = 'buildbucket/builds-experimental/'

BUCKETER_24_HR = gae_ts_mon.GeometricBucketer(growth_factor=10**0.05)
BUCKETER_48_HR = gae_ts_mon.GeometricBucketer(growth_factor=10**0.053)
BUCKETER_5_SEC = gae_ts_mon.GeometricBucketer(growth_factor=10**0.0374)
BUCKETER_1K = gae_ts_mon.GeometricBucketer(growth_factor=10**0.031)


def _fields_for(build, field_names):
  """Returns field values for a build"""
  result = {}
  for field_name in field_names:
    field = _BUILD_FIELDS.get(field_name)
    if not field:
      raise ValueError('invalid field %r' % field_name)
    val = field.getter(build) or field.value_type()
    result[field_name] = field.value_type(val)
  return result


def _fields_for_fn(fields):
  assert all(f.name in _BUILD_FIELDS for f in fields)
  field_names = [f.name for f in fields]
  return lambda b: _fields_for(b, field_names)  # pragma: no cover


def _build_fields(*names):
  return [_BUILD_FIELDS[n][1](n) for n in names]


def _incrementer(metric_suffix, description, fields):
  """Returns a function that increments a counter metric.

  Metric fields must conform _BUILD_FIELDS.

  The returned function accepts a build.
  """

  def mk_metric(metric_prefix):
    return gae_ts_mon.CounterMetric(
        metric_prefix + metric_suffix, description, fields
    )

  metric_prod = mk_metric(_METRIC_PREFIX_PROD)
  metric_exp = mk_metric(_METRIC_PREFIX_EXPERIMENTAL)
  fields_for = _fields_for_fn(fields)

  def inc(build):  # pragma: no cover
    metric = metric_exp if build.experimental else metric_prod
    metric.increment(fields_for(build))

  return inc


def _adder(metric_suffix, description, fields, bucketer, units, value_fn):
  """Returns a function that adds a build value to a cumulative distribution.

  Metric fields must conform _BUILD_FIELDS.
  value_fn accepts a build.

  The returned function accepts a build.
  """

  def mk_metric(metric_prefix):
    return gae_ts_mon.CumulativeDistributionMetric(
        metric_prefix + metric_suffix,
        description,
        fields,
        bucketer=bucketer,
        units=units
    )

  metric_prod = mk_metric(_METRIC_PREFIX_PROD)
  metric_exp = mk_metric(_METRIC_PREFIX_EXPERIMENTAL)
  fields_for = _fields_for_fn(fields)

  def add(build):  # pragma: no cover
    metric = metric_exp if build.experimental else metric_prod
    metric.add(value_fn(build.proto), fields_for(build))

  return add


def _duration_adder(metric_suffix, description, value_fn):
  return _adder(
      metric_suffix, description,
      _build_fields(
          'bucket', 'builder', 'result', 'failure_reason', 'cancelation_reason',
          'canary'
      ), BUCKETER_48_HR, gae_ts_mon.MetricsDataUnits.SECONDS, value_fn
  )


inc_created_builds = _incrementer(
    'created', 'Build creation',
    _build_fields('bucket', 'builder', 'user_agent')
)
inc_started_builds = _incrementer(
    'started', 'Build start', _build_fields('bucket', 'builder', 'canary')
)
inc_completed_builds = _incrementer(
    'completed',
    'Build completion, including success, failure and cancellation',
    _build_fields(
        'bucket', 'builder', 'result', 'failure_reason', 'cancelation_reason',
        'canary'
    )
)
inc_lease_expirations = _incrementer(
    'lease_expired', 'Build lease expirations',
    _build_fields('bucket', 'builder', 'status')
)
inc_leases = _incrementer(
    'leases', 'Successful build leases or lease extensions',
    _build_fields('bucket', 'builder')
)

inc_heartbeat_failures = gae_ts_mon.CounterMetric(
    'buildbucket/builds/heartbeats', 'Failures to extend a build lease', []
).increment


def _ts_delta_sec(start, end):  # pragma: no cover
  assert start.seconds
  assert end.seconds
  return end.seconds - start.seconds


# requires the argument to have create_time and end_time.
add_build_cycle_duration = _duration_adder(  # pragma: no branch
    'cycle_durations', 'Duration between build creation and completion',
    lambda b: _ts_delta_sec(b.create_time, b.end_time))

# requires the argument to have start_time and end_time.
add_build_run_duration = _duration_adder(  # pragma: no branch
    'run_durations', 'Duration between build start and completion',
    lambda b: _ts_delta_sec(b.start_time, b.end_time))

# requires the argument to have create_time and start_time.
add_build_scheduling_duration = _duration_adder(  # pragma: no branch
    'scheduling_durations', 'Duration between build creation and start',
    lambda b: _ts_delta_sec(b.create_time, b.start_time))
SEQUENCE_NUMBER_GEN_DURATION_MS = gae_ts_mon.CumulativeDistributionMetric(
    'buildbucket/sequence_number/gen_duration',
    'Duration of a sequence number generation in ms',
    [gae_ts_mon.StringField('sequence')],
    # Bucketer for 1ms..5s range
    bucketer=BUCKETER_5_SEC,
    units=gae_ts_mon.MetricsDataUnits.MILLISECONDS
)
TAG_INDEX_INCONSISTENT_ENTRIES = gae_ts_mon.NonCumulativeDistributionMetric(
    'buildbucket/tag_index/inconsistent_entries',
    'Number of inconsistent entries encountered during build search',
    [gae_ts_mon.StringField('tag')],
    # We can't have more than 1000 entries in a tag index.
    bucketer=BUCKETER_1K
)
TAG_INDEX_SEARCH_SKIPPED_BUILDS = gae_ts_mon.NonCumulativeDistributionMetric(
    'buildbucket/tag_index/skipped_builds',
    'Number of builds we fetched, but skipped',
    [gae_ts_mon.StringField('tag')],
    # We can't have more than 1000 entries in a tag index.
    bucketer=BUCKETER_1K
)
