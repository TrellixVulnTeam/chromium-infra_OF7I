# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import pytest
import datetime

import apache_beam as beam
from apache_beam.testing import test_pipeline
from apache_beam.io.gcp.datastore.v1new.types import Entity, Key
from apache_beam.testing.util import assert_that, equal_to

from chromeperf_pipelines import delete_upload_tokens


def new_token(key):
    e = Entity(Key(['Token', key], project='test'))
    e.set_properties({
        'update_time':
        datetime.datetime(2020, 1, 1, 0, 0, 0, 0, datetime.timezone.utc)
    })
    return e


def new_measurement(key, token_key):
    e = Entity(Key(['Measurement', key], project='test'))
    if token_key is not None:
        e.set_properties({'token': Key(['Token', token_key], project='test')})
    return e


TOKENS = [new_token(k) for k in range(10)]
MEASUREMENTS = [
    new_measurement(k + (t * 10), t) for t in range(10) for k in range(10)
]


def test_select_all_expired():
    with test_pipeline.TestPipeline(additional_pipeline_args=[
            '--max_lifetime',
            '1H',
            '--reference_time',
            '2020-01-02:00:00:01+0000',
    ]) as p:
        selection_options = p.options.view_as(
            delete_upload_tokens.TokenSelectionOptions)
        tokens = p | "CreateTokens" >> beam.Create(TOKENS)
        measurements = p | "CreateMeasurements" >> beam.Create(MEASUREMENTS)

        tokens_to_delete, measurements_to_delete = delete_upload_tokens.select_expired_tokens(
            tokens, measurements, selection_options.get_selection_provider())

        assert_that(
            tokens_to_delete,
            equal_to([Key(['Token', t], project='test') for t in range(10)]),
            label='AssertTokensToDelete')

        assert_that(measurements_to_delete,
                    equal_to([
                        Key(['Measurement', m], project='test')
                        for m in range(100)
                    ]),
                    label='AssertMeasurementsToDelete')


def test_select_none_expired():
    with test_pipeline.TestPipeline(additional_pipeline_args=[
            '--max_lifetime',
            '240H',
            '--reference_time',
            '2020-01-01:00:00:01+0000',
    ]) as p:
        selection_options = p.options.view_as(
            delete_upload_tokens.TokenSelectionOptions)
        tokens = p | "CreateTokens" >> beam.Create(TOKENS)
        measurements = p | "CreateMeasurements" >> beam.Create(MEASUREMENTS)

        tokens_to_delete, measurements_to_delete = delete_upload_tokens.select_expired_tokens(
            tokens, measurements, selection_options.get_selection_provider())

        assert_that(tokens_to_delete,
                    equal_to([]),
                    label='AssertTokensToDelete')

        assert_that(measurements_to_delete,
                    equal_to([]),
                    label='AssertMeasurementsToDelete')


def test_select_missing_tokens():
    with test_pipeline.TestPipeline(additional_pipeline_args=[
            '--max_lifetime',
            '1H',
            '--reference_time',
            '2020-01-01:00:00:01+0000',
    ]) as p:
        selection_options = p.options.view_as(
            delete_upload_tokens.TokenSelectionOptions)
        tokens = p | "CreateTokens" >> beam.Create([])
        measurements = p | "CreateMeasurements" >> beam.Create([
            new_measurement(k + (t * 10), t) for t in range(10)
            for k in range(10)
        ])

        tokens_to_delete, measurements_to_delete = delete_upload_tokens.select_expired_tokens(
            tokens, measurements, selection_options.get_selection_provider())

        assert_that(tokens_to_delete,
                    equal_to([]),
                    label='AssertTokensToDelete')

        assert_that(measurements_to_delete,
                    equal_to([
                        Key(['Measurement', m], project='test')
                        for m in range(100)
                    ]),
                    label='AssertMeasurementsToDelete')


def test_select_measurements_no_token():
    with test_pipeline.TestPipeline(additional_pipeline_args=[
            '--max_lifetime',
            '1H',
            '--reference_time',
            '2020-01-01:00:00:01+0000',
    ]) as p:
        selection_options = p.options.view_as(
            delete_upload_tokens.TokenSelectionOptions)
        tokens = p | "CreateTokens" >> beam.Create([])
        measurements = p | "CreateMeasurements" >> beam.Create([
            new_measurement(k + (t * 10), None) for t in range(10)
            for k in range(10)
        ])

        tokens_to_delete, measurements_to_delete = delete_upload_tokens.select_expired_tokens(
            tokens, measurements, selection_options.get_selection_provider())

        assert_that(tokens_to_delete,
                    equal_to([]),
                    label='AssertTokensToDelete')

        assert_that(measurements_to_delete,
                    equal_to([
                        Key(['Measurement', m], project='test')
                        for m in range(100)
                    ]),
                    label='AssertMeasurementsToDelete')