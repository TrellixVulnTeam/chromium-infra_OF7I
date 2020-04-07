# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.
# Or at https://developers.google.com/open-source/licenses/bsd

from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import io
import tensorflow as tf

from googleapiclient import discovery
from googleapiclient import errors
from oauth2client.client import GoogleCredentials

from trainer2 import train_ml_helpers


def fetch_training_data(bucket, prefix, trainer_type):

  credentials = GoogleCredentials.get_application_default()
  storage = discovery.build('storage', 'v1', credentials=credentials)
  objects = storage.objects()

  request = objects.list(bucket=bucket, prefix=prefix)
  response = make_api_request(request)
  items = response.get('items')
  csv_filepaths = [blob.get('name') for blob in items]

  if trainer_type == 'spam':
    return fetch_spam(csv_filepaths, bucket, objects)
  else:
    return fetch_component(csv_filepaths, bucket, objects)


def fetch_spam(csv_filepaths, bucket, objects):

  all_contents = []
  all_labels = []
  # Add code
  csv_filepaths = [
      'spam-training-data/full-android.csv',
      'spam-training-data/full-support.csv',
  ] + csv_filepaths

  for filepath in csv_filepaths:
    media = fetch_training_csv(filepath, objects, bucket)
    contents, labels, skipped_rows = train_ml_helpers.spam_from_file(
        io.StringIO(media))

    # Sanity check: the contents and labels should be matched pairs.
    if len(contents) == len(labels) != 0:
      all_contents.extend(contents)
      all_labels.extend(labels)

    tf.get_logger().info(
        '{:<40}{:<20}{:<20}'.format(
            filepath, 'added %d rows' % len(contents),
            'skipped %d rows' % skipped_rows))

  return all_contents, all_labels


def fetch_component(csv_filepaths, bucket, objects):

  all_contents = []
  all_labels = []
  for filepath in csv_filepaths:
    media = fetch_training_csv(filepath, objects, bucket)
    contents, labels = train_ml_helpers.component_from_file(io.StringIO(media))

    # Sanity check: the contents and labels should be matched pairs.
    if len(contents) == len(labels) != 0:
      all_contents.extend(contents)
      all_labels.extend(labels)

    tf.get_logger().info(
        '{:<40}{:<20}'.format(filepath, 'added %d rows' % len(contents)))

  return all_contents, all_labels


def fetch_training_csv(filepath, objects, bucket):
  request = objects.get_media(bucket=bucket, object=filepath)
  return str(make_api_request(request), 'utf-8')


def make_api_request(request):
  try:
    return request.execute()
  except errors.HttpError as err:
    tf.get_logger().error('There was an error with the API. Details:')
    tf.get_logger().error(err._get_reason())
    raise
