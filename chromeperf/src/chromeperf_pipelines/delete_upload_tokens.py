# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import apache_beam as beam
import datetime
import logging
import re

from apache_beam import metrics
from apache_beam.io.gcp.datastore.v1new import datastoreio
from apache_beam.io.gcp.datastore.v1new.types import Query
from apache_beam.options.pipeline_options import GoogleCloudOptions
from apache_beam.options.pipeline_options import PipelineOptions

DATETIME_FORMAT = '%G-%m-%d:%H:%M:%S%z'


class TokenSelectionOptions(PipelineOptions):
    @classmethod
    def _add_argparse_args(cls, parser):
        parser.add_argument(
            '--max_lifetime',
            help=('The duration of time an UploadToken should be kept in '
                  'the Datstore, expressed as a string in hours or '
                  'minutes or combinations (e.g. 1h30m)'),
            default='6h',
        )
        parser.add_argument(
            '--reference_time',
            help=(f'A datetime-parseable reference time, in this '
                  f'format: {DATETIME_FORMAT} -- if empty means '
                  f'"now".'),
            default='',
        )


def select_expired_tokens(
        raw_tokens: beam.Pipeline,
        raw_measurements: beam.Pipeline,
        max_lifetime: datetime.timedelta,
        reference_time: datetime.datetime,
        expired_tokens,
):
    missing_token_measurements_count = metrics.Metrics.counter(
        'select', 'missing_token_measurements_count')
    deleted_tokens = metrics.Metrics.counter('select', 'deleted_tokens')
    deleted_measurements = metrics.Metrics.counter('select',
                                                   'deleted_measurements')

    def extract_update_timestamp(token):
        return (token.key.to_client_key().id, token)

    tokens = (raw_tokens
              | 'ExtractTokenKey' >> beam.Map(extract_update_timestamp))

    def extract_associated_token(measurement_entity, missing_counter):
        measurement = measurement_entity.to_client_entity()
        token_key = measurement.get('token')
        if not token_key:
            missing_counter.inc()
            token_key = '(unspecified)'
        else:
            token_key = token_key.id
        return (token_key, measurement_entity.key)

    measurements = (
        raw_measurements
        | 'ExtractAssociatedToken' >> beam.Map(
            extract_associated_token, missing_token_measurements_count))

    # We'll collect all `Measurement` keys by the 'Token' key.
    measurements_by_token = (({
        'token': tokens,
        'measurements': measurements,
    })
                             | 'Merge' >> beam.CoGroupByKey())

    def select_expired(keyed_token_and_measurements,
                       max_lifetime: datetime.timedelta,
                       reference_time: datetime.datetime, expired_tokens):
        _, token_and_measurements = keyed_token_and_measurements
        tokens = token_and_measurements['token']

        # This means we have already deleted the token for these
        # measurements, so we'll always delete these measurements.
        if not tokens:
            expired_tokens.inc()
            return True
        token = token_and_measurements['token'][0].to_client_entity()
        lifetime = reference_time - token['update_time']
        if lifetime >= max_lifetime:
            expired_tokens.inc()
            return True
        return False

    # We return two PCollection instances, one with just the expired tokens and
    # the other the expired measurements.
    expired_tokens = (measurements_by_token
                      | 'SelectExpiredTokens' >> beam.Filter(
                          select_expired,
                          max_lifetime,
                          reference_time,
                          expired_tokens,
                      ))

    def extract_token_key(keyed_token_and_measurements, counter):
        _, token_and_measurements = keyed_token_and_measurements
        tokens = token_and_measurements['token']
        res = [t.key for t in tokens]
        counter.inc(len(res))
        return res

    def pick_nonempty_tokens(keyed_token_and_measurements):
        token_key, _ = keyed_token_and_measurements
        return token_key != '(unspecified)' or len(token_key) > 0

    tokens_to_delete = (
        expired_tokens
        | 'PickNonEmptyTokens' >> beam.Filter(pick_nonempty_tokens)
        |
        'ExtractTokenKeys' >> beam.FlatMap(extract_token_key, deleted_tokens))

    def extract_measurement(keyed_token_and_measurements, counter):
        _, token_and_measurements = keyed_token_and_measurements
        res = token_and_measurements['measurements']
        counter.inc(len(res))
        return res

    measurements_to_delete = (expired_tokens
                              | 'ExtractMeasurementKeys' >> beam.FlatMap(
                                  extract_measurement, deleted_measurements))

    return tokens_to_delete, measurements_to_delete


class CountInput(beam.DoFn):
    def __init__(self, counter):
        self._counter = counter

    def process(self, input):
        self._counter.inc()
        yield input


def parse_timedelta(value):
    # Support hours and minutes and combinations of each.
    result = re.match(r'((?P<hours>\d+)h)?((?P<minutes>\d+)m)?', value)
    hours = result.group('hours') or '0'
    minutes = result.group('minutes') or '0'
    return datetime.timedelta(minutes=int(minutes), hours=int(hours))


def parse_datetime(value):
    if not value:
        return datetime.datetime.now(tz=datetime.timezone.utc)
    return datetime.datetime.strptime(value, DATETIME_FORMAT)


def main():
    project = 'chromeperf'
    options = PipelineOptions()
    options.view_as(GoogleCloudOptions).project = project
    deletion_options = options.view_as(TokenSelectionOptions)
    max_lifetime = parse_timedelta(deletion_options.max_lifetime)
    reference_time = parse_datetime(deletion_options.reference_time)

    p = beam.Pipeline(options=options)
    token_count = metrics.Metrics.counter('main', 'tokens_read')
    measurement_count = metrics.Metrics.counter('main', 'measurements_read')
    expired_token_count = metrics.Metrics.counter('main', 'expired_tokens')

    raw_tokens = (p | 'ReadUploadTokens' >> datastoreio.ReadFromDatastore(
        query=Query(kind='Token', project=project)))

    # Count the tokens.
    _ = (raw_tokens | 'CountTokens' >> beam.ParDo(CountInput(token_count)))

    raw_measurements = (p
                        | 'ReadMeasurements' >> datastoreio.ReadFromDatastore(
                            query=Query(kind='Measurement', project=project)))

    # Count the measurements.
    _ = (raw_measurements
         | 'CountMeasurements' >> beam.ParDo(CountInput(measurement_count)))

    tokens_to_delete, measurements_to_delete = select_expired_tokens(
        raw_tokens,
        raw_measurements,
        max_lifetime,
        reference_time,
        expired_token_count,
    )

    _ = (tokens_to_delete
         | 'DeleteTokens' >> datastoreio.DeleteFromDatastore(project))

    _ = (measurements_to_delete
         | 'ReshuffleMeasurements' >> beam.Reshuffle()
         | 'DeleteMeasurements' >> datastoreio.DeleteFromDatastore(project))

    # Run the pipeline!
    result = p.run()
    result.wait_until_finish()


if __name__ == '__main__':
    logging.getLogger().setLevel(logging.INFO)
    main()
