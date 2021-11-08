# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import webapp2

import gae_ts_mon

from gae_libs import appengine_util

from handlers import code_coverage_monolith
from handlers.code_coverage import export_absolute_coverage
from handlers.code_coverage import export_feature_coverage
from handlers.code_coverage import update_postsubmit_report

# Feaure coverage worker module.
feature_coverage_worker_handler_mappings = [
    ('.*/coverage/task/feature-coverage.*',
     export_feature_coverage.ExportFeatureCoverageMetrics),
]
feature_coverage_worker_application = webapp2.WSGIApplication(
    feature_coverage_worker_handler_mappings, debug=False)
if appengine_util.IsInProductionApp():
  gae_ts_mon.initialize(feature_coverage_worker_application)

# Referenced coverage worker module.
referenced_coverage_worker_handler_mappings = [
    ('.*/coverage/task/referenced-coverage.*',
     code_coverage_monolith.CreateReferencedCoverageMetrics),
]
referenced_coverage_worker_application = webapp2.WSGIApplication(
    referenced_coverage_worker_handler_mappings, debug=False)
if appengine_util.IsInProductionApp():
  gae_ts_mon.initialize(referenced_coverage_worker_application)


# "code-coverage-backend" module.
code_coverage_backend_handler_mappings = [
    ('.*/coverage/task/fetch-source-file',
     code_coverage_monolith.FetchSourceFile),
    ('.*/coverage/task/process-data/.*',
     code_coverage_monolith.ProcessCodeCoverageData),
    ('.*/coverage/cron/files-absolute-coverage',
     export_absolute_coverage.ExportFilesAbsoluteCoverageMetricsCron),
    ('.*/coverage/task/files-absolute-coverage',
     export_absolute_coverage.ExportFilesAbsoluteCoverageMetrics),
    ('.*/coverage/cron/all-feature-coverage',
     export_feature_coverage.ExportAllFeatureCoverageMetricsCron),
    ('.*/coverage/task/all-feature-coverage',
     export_feature_coverage.ExportAllFeatureCoverageMetrics),
    ('.*/coverage/cron/referenced-coverage',
     code_coverage_monolith.CreateReferencedCoverageMetricsCron),
    ('.*/coverage/task/postsubmit-report/update',
     update_postsubmit_report.UpdatePostsubmitReport),
]
code_coverage_backend_web_application = webapp2.WSGIApplication(
    code_coverage_backend_handler_mappings, debug=False)
if appengine_util.IsInProductionApp():
  gae_ts_mon.initialize(code_coverage_backend_web_application)

# "code-coverage-frontend" module.
code_coverage_frontend_handler_mappings = [
    # TODO(crbug.com/924573): Migrate to '.*/coverage/api/coverage-data'.
    ('/coverage/api/coverage-data', code_coverage_monolith.ServeCodeCoverageData
    ),
    # These mappings are separated so that ts_mon data (e.g. latency) is
    # groupable by view. (instead of a single entry like /coverage/p/.*)
    ('/coverage/p/.*/referenced', code_coverage_monolith.ServeCodeCoverageData),
    ('/coverage/p/.*/component', code_coverage_monolith.ServeCodeCoverageData),
    ('/coverage/p/.*/dir', code_coverage_monolith.ServeCodeCoverageData),
    ('/coverage/p/.*/file', code_coverage_monolith.ServeCodeCoverageData),
    ('/coverage/p/.*', code_coverage_monolith.ServeCodeCoverageData)
]
code_coverage_frontend_web_application = webapp2.WSGIApplication(
    code_coverage_frontend_handler_mappings, debug=False)
if appengine_util.IsInProductionApp():
  gae_ts_mon.initialize(code_coverage_frontend_web_application)
