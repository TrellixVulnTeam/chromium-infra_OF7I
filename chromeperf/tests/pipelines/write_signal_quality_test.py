# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest
import datetime

import apache_beam as beam
from apache_beam.testing import test_pipeline
from apache_beam.testing.util import assert_that
from apache_beam.testing.util import equal_to
from apache_beam.io.gcp.datastore.v1new.types import Entity
from apache_beam.io.gcp.datastore.v1new.types import Key
from apache_beam.testing.util import assert_that, equal_to

from chromeperf_pipelines import write_signal_quality


def make_entity(now, id, score):
    entity = Entity(key=Key(
        [
            'SignalQuality',
            f'ChromiumPerf/android-pixel2-perf/blink_perf/{id}',
            'SignalQualityScore',
            '0',
        ],
        project='test',
    ))
    entity.set_properties({
        'score': score,
        'updated_time': now,
    })
    return entity


def test_write_signal_quality():
    now = datetime.datetime.now(),
    with test_pipeline.TestPipeline() as p:
        input = p | beam.Create([{
            'test': 'ChromiumPerf/android-pixel2-perf/blink_perf/1',
            'culprit_found': 1,
            'bisection_count': 2,
        }, {
            'test': 'ChromiumPerf/android-pixel2-perf/blink_perf/2',
            'culprit_found': 1,
            'bisection_count': 1,
        }, {
            'test': 'ChromiumPerf/android-pixel2-perf/blink_perf/3',
            'culprit_found': 0,
            'bisection_count': 1,
        }])
        pcoll = write_signal_quality.make_signal_quality_entities(
            'test', now, input)
        assert_that(
            pcoll,
            equal_to([
                make_entity(now, 1, 0.5),
                make_entity(now, 2, 1.0),
                make_entity(now, 3, 0.0),
            ]),
        )
