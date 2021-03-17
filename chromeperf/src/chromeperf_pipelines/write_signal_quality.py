# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import logging
import datetime
import apache_beam as beam

from apache_beam.io.gcp.datastore.v1new import datastoreio
from apache_beam.io.gcp.datastore.v1new.types import Query
from apache_beam.options.pipeline_options import GoogleCloudOptions
from apache_beam.options.pipeline_options import PipelineOptions


def make_signal_quality_entities(project, now, p):
    from apache_beam.io.gcp.datastore.v1new.types import Entity
    from apache_beam.io.gcp.datastore.v1new.types import Key

    def make_entity(content):
        score = content['culprit_found'] / content['bisection_count']
        version = 0

        key = Key(
            [
                'SignalQuality',
                content['test'],
                'SignalQualityScore',
                str(version),
            ],
            project=project,
        )
        entity = Entity(key)
        entity.set_properties({
            'score': score,
            'updated_time': now,
        })
        return entity

    return p | 'CreateEntity' >> beam.Map(make_entity)


def main():
    project = 'chromeperf'
    options = PipelineOptions()
    options.view_as(GoogleCloudOptions).project = project

    p = beam.Pipeline(options=options)
    bisections = p | 'QueryTable' >> beam.io.ReadFromBigQuery(
        use_standard_sql=True,
        query="""
        WITH
          anomaly_bisections AS (
            SELECT
              a.id id,
              a.test test,
              IF(p.difference_count is NULL, 0, p.difference_count) difference_count,
            FROM `chromeperf.chromeperf_dashboard_data.anomalies` a
            CROSS JOIN UNNEST(pinpoint_bisects) pinpoint_jobs
            INNER JOIN
              `chromeperf.chromeperf_dashboard_data.jobs` p
              ON
                CAST(CONCAT("0x", pinpoint_jobs) AS int64) = p.id
                AND p.start_time IS NOT NULL
                AND p.user_email
                  = "425761728072-pa1bs18esuhp2cp2qfa1u9vb6p1v6kfu@developer.gserviceaccount.com"
            WHERE DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 84 DAY)
          )
        SELECT
          test, SUM(IF(difference_count = 0, 0, 1)) as culprit_found, count(*) as bisection_count
        FROM anomaly_bisections
        GROUP BY test
        """,
    )
    entities = make_signal_quality_entities(
        project,
        datetime.datetime.now(),
        bisections,
    )
    entities | 'WriteToDatastore' >> datastoreio.WriteToDatastore(project)

    result = p.run()
    result.wait_until_finish()


if __name__ == '__main__':
    logging.getLogger().setLevel(logging.INFO)
    main()
