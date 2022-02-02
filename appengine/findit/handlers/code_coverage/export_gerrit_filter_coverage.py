# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.from datetime import datetime

import datetime
import logging
import time

from gae_libs.handlers.base_handler import BaseHandler, Permission

from google.appengine.api import taskqueue
from google.appengine.ext import ndb

from common import constants
from handlers.code_coverage import utils
from model.code_coverage import CoverageReportModifier
from model.code_coverage import PostsubmitReport
from services.code_coverage import gerrit_filter_coverage


class ExportAllCoverageMetricsCron(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandleGet(self):
    # Cron jobs run independently of each other. Therefore, there is no
    # guarantee that they will run either sequentially or simultaneously.
    #
    # Executing this job concurrently doesn't bring much
    # benefits, so use task queue to enforce that at most one task
    # can be executed at any time.
    taskqueue.add(
        method='GET',
        queue_name=constants.ALL_GERRIT_FILTER_COVERAGE_QUEUE,
        target=constants.CODE_COVERAGE_BACKEND,
        url='/coverage/task/all-gerrit-filter-coverage')
    return {'return_code': 200}


class ExportAllCoverageMetrics(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def _GetActiveGerritFilters(self):
    """Returns filters for which coverage is to be generated.

    Yields id of the corresponding CoverageReportModifier.
    """
    query = CoverageReportModifier.query(
        CoverageReportModifier.server_host == 'chromium.googlesource.com',
        CoverageReportModifier.project == 'chromium/src',
        CoverageReportModifier.is_active == True)
    more = True
    cursor = None
    page_size = 100
    make_inactive = []
    while more:
      results, cursor, more = query.fetch_page(
          page_size,
          start_cursor=cursor,
          config=ndb.ContextOptions(use_cache=False))
      for x in results:
        if x.gerrit_hashtag or x.author:
          # To prevent bloating up of feature coverage dashboard, we do not
          # generate coverage metrics for features which are older than 90 days.
          if x.gerrit_hashtag and x.update_timestamp + datetime.timedelta(
              days=90) < datetime.datetime.now():
            x.is_active = False
            make_inactive.append(x)
          else:
            yield x.key.id()
    ndb.put_multi(make_inactive)

  def HandleGet(self):
    # Spawn a sub task for each active filter
    for modifier_id in self._GetActiveGerritFilters():
      modifier = CoverageReportModifier.Get(modifier_id)
      logging.info('modifier_id: %d, gerrit_hashtag: %s, author: %s' %
                   (modifier_id, modifier.gerrit_hashtag, modifier.author))
      url = '/coverage/task/gerrit-filter-coverage?modifier_id=%d' % (
          modifier_id)
      author = modifier.author.replace('@', '_') if modifier.author else None
      taskqueue.add(
          method='GET',
          url=url,
          name='%s-%s-%s' % (modifier.gerrit_hashtag, author,
                             datetime.datetime.now().strftime('%d%m%Y-%H%M%S')),
          queue_name=constants.GERRIT_FILTER_COVERAGE_QUEUE,
          target=constants.CODE_COVERAGE_GERRIT_FILTER_COVERAGE_WORKER)
    return {'return_code': 200}


class ExportCoverageMetrics(BaseHandler):
  PERMISSION_LEVEL = Permission.APP_SELF

  def HandleGet(self):
    start_time = time.time()
    modifier_id = int(self.request.get('modifier_id'))
    gerrit_filter_coverage.ExportCoverage(modifier_id, int(start_time))
    minutes = (time.time() - start_time) / 60
    modifier = CoverageReportModifier.Get(modifier_id)
    logging.info(
        'Generating coverage for hashtag:%s author:%s took %.0f minutes',
        modifier.gerrit_hashtag, modifier.author, minutes)
    return {'return_code': 200}
