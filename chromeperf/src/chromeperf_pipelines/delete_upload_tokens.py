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


def parse_timedelta(value):
    # Support hours and minutes and combinations of each.
    result = re.match(r'((?P<hours>\d+)h)?((?P<minutes>\d+)m)?', value)
    hours = result.group('hours') or '0'
    minutes = result.group('minutes') or '0'
    return datetime.timedelta(minutes=int(minutes), hours=int(hours))


_DATETIME_FORMAT = '%G-%m-%d:%H:%M:%S%z'


def parse_datetime(value):
    if not value:
        return datetime.datetime.now(tz=datetime.timezone.utc)
    return datetime.datetime.strptime(value, _DATETIME_FORMAT)


class TokenDeletionOptions(PipelineOptions):
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
                  f'format: {_DATETIME_FORMAT} -- if empty means '
                  f'"now".'),
            default='',
        )


def main():
    project = 'chromeperf'
    options = PipelineOptions()
    options.view_as(GoogleCloudOptions).project = project
    deletion_options = options.view_as(TokenDeletionOptions)
    max_lifetime = parse_timedelta(deletion_options.max_lifetime)
    reference_time = parse_datetime(deletion_options.reference_time)

    p = beam.Pipeline(options=options)
    token_count = metrics.Metrics.counter('main', 'tokens_read')
    measurement_count = metrics.Metrics.counter('main', 'measurements_read')
    missing_token_measurements_count = metrics.Metrics.counter(
        'main', 'missing_token_measurements_count')
    deleted_tokens = metrics.Metrics.counter('main', 'deleted_tokens')
    deleted_measurements = metrics.Metrics.counter('main',
                                                   'deleted_measurements')

    # Read 'UploadToken' entities, and only get the required fields.
    def extract_update_timestamp(token):
        return (token.key.to_client_key().id, token)

    tokens = (p
              | 'ReadUploadTokens' >> datastoreio.ReadFromDatastore(
                  query=Query(kind='Token', project=project))
              | 'ExtractTokenKey' >> beam.Map(extract_update_timestamp))

    class CountInput(beam.DoFn):
        def __init__(self, counter):
            self._counter = counter

        def process(self, input):
            self._counter.inc()
            yield input

    # Count the tokens.
    _ = (tokens | 'CountTokens' >> beam.ParDo(CountInput(token_count)))

    def extract_associated_token(measurement_entity, missing_counter):
        measurement = measurement_entity.to_client_entity()
        token_key = measurement.get('token')
        if not token_key:
            missing_counter.inc()
            token_key = '(unspecified)'
        else:
            token_key = token_key.id
        return (token_key, measurement_entity.key)

    # Read 'Measurement' entities.
    measurements = (p
                    | 'ReadMeasurements' >> datastoreio.ReadFromDatastore(
                        query=Query(kind='Measurement', project=project))
                    | 'ExtractAssociatedToken' >> beam.Map(
                        extract_associated_token,
                        missing_counter=missing_token_measurements_count))

    # Count the measurements.
    _ = (measurements
         | 'CountMeasurements' >> beam.ParDo(CountInput(measurement_count)))

    # We'll collect all `Measurement` keys by the 'Token' key.
    measurements_by_token = (({
        'token': tokens,
        'measurements': measurements,
    })
                             | 'Merge' >> beam.CoGroupByKey())

    def select_expired(keyed_token_and_measurements, max_lifetime,
                       reference_time):
        _, token_and_measurements = keyed_token_and_measurements
        tokens = token_and_measurements['token']

        # This means we have already deleted the token for these
        # measurements, so we'll always delete these measurements.
        if not tokens:
            return True
        token = token_and_measurements['token'][0].to_client_entity()
        lifetime = reference_time - token['update_time']
        return lifetime >= max_lifetime

    # Now we delete all the measurements and tokens that are expired.
    expired_tokens = (measurements_by_token
                      | 'SelectExpiredTokens' >> beam.Filter(
                          select_expired,
                          max_lifetime,
                          reference_time,
                      ))

    class TokenKeyExtractor(beam.DoFn):
        def __init__(self, counter):
            self._counter = counter

        def process(self, keyed_token_and_measurements):
            # We extract the key from just the token, but only if we have it.
            _, token_and_measurements = keyed_token_and_measurements
            token = token_and_measurements['token'][0]
            self._counter.inc()
            yield token.key

    # Delete the Token.
    _ = (expired_tokens
         | 'PickNonEmptyTokens' >>
         beam.Filter(lambda ktm: len(ktm[1].get('token', [])) > 0)
         | 'ExtractTokenKeys' >> beam.ParDo(TokenKeyExtractor(deleted_tokens))
         | 'DeleteTokens' >> datastoreio.DeleteFromDatastore(project))

    class MeasurementKeyExtractor(beam.DoFn):
        def __init__(self, counter):
            self._counter = counter

        def process(self, keyed_token_and_measurements):
            # We extract the key from just the token, but only if we have it.
            _, token_and_measurements = keyed_token_and_measurements
            for measurement_key in token_and_measurements['measurements']:
                self._counter.inc()
                yield measurement_key

    # Delete the Measurement entities.
    _ = (expired_tokens
         | 'ExtractMeasurementKeys' >> beam.ParDo(
             MeasurementKeyExtractor(deleted_measurements))
         | 'DeleteMeasurements' >> datastoreio.DeleteFromDatastore(project))

    # Run the pipeline!
    result = p.run()
    result.wait_until_finish()
    for counter in result.metrics().query()['counters']:
        print(f'Counter: {counter}')
        print(f'  = {counter.result}')


if __name__ == '__main__':
    logging.getLogger().setLevel(logging.INFO)
    main()
