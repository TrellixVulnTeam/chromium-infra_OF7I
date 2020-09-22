# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file or at
# https://developers.google.com/open-source/licenses/bsd
""" Tasks and handlers for maintaining the spam classifier model. These
    should be run via cron and task queue rather than manually.
"""
from __future__ import print_function
from __future__ import division
from __future__ import absolute_import

import cloudstorage
import datetime
import logging
import webapp2

from google.appengine.api import app_identity

from features.generate_dataset import build_component_dataset
from framework import cloud_tasks_helpers
from framework import servlet
from framework import urls


class ComponentTrainingDataExport(webapp2.RequestHandler):
  """Trigger a training data export task"""
  def get(self):
    logging.info('Training data export requested.')
    task = {
        'app_engine_http_request':
            {
                'http_method': 'GET',
                'relative_uri': urls.COMPONENT_DATA_EXPORT_TASK,
            }
    }
    cloud_tasks_helpers.create_task(task, queue='componentexport')


class ComponentTrainingDataExportTask(servlet.Servlet):
  """Export training data for issues and their assigned components, to be used
     to train  a model later.
  """
  def get(self):
    logging.info('Training data export initiated.')
    bucket_name = app_identity.get_default_gcs_bucket_name()
    logging.info('Bucket name: %s', bucket_name)
    date_str = datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')

    logging.info('Opening cloud storage')
    gcs_file = cloudstorage.open('/' + bucket_name
                                 + '/component_training_data/'
                                 + date_str + '.csv',
        content_type='text/csv', mode='w')

    logging.info('GCS file opened')

    gcs_file = build_component_dataset(self.services.issue, gcs_file)

    gcs_file.close()
