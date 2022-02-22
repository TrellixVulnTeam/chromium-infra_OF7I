# This is an application designed to collect and analyze build/compile stats.

[go/cbs-doc](http://go/cbs-doc)

Deign Doc: [Chromium build time profiler](https://docs.google.com/a/chromium.org/document/d/16TdPTIIZbtAarXZIMJdiT9CePG5WYCrdxm5u9UuHXNY/edit#heading=h.xgjl2srtytjt)

How to:

See [infra/go/README.md](../../../../README.md) for preparation.

 to compile

```shell
  $ make build
```

 to deploy to production

```shell
  $ make deploy-prod
```

 to run test

```shell
  $ make test
```

## Operation for BigQuery Table

Setup

1. Make Dataset

```shell
$ bq --project_id=$PROJECT mk ninjalog
```

2. Update BigQuery table config/schema.

```shell
$ make update-staging # for staging
$ make update-prod # for prod
```


## ninja log upload from user

Ninja log is uploaded from user too.
Upload script is located in [depot_tools](https://chromium.googlesource.com/chromium/tools/depot_tools.git/+/master/ninjalog_uploader.py).

### example query

[link to query editor](https://console.cloud.google.com/bigquery?project=chromium-build-stats)

1. Find time consuming build tasks in a day per target_os, build os and outputs

```
SELECT
  (
  SELECT
    value
  FROM
    UNNEST(build_configs)
  WHERE
    key = "target_os") target_os,
  os,
  SUBSTR(ARRAY_TO_STRING(outputs, ", "), 0, 128) outputs,
  ROUND(AVG(end_duration_sec - start_duration_sec), 2) task_duration_avg,
  ROUND(SUM(end_duration_sec - start_duration_sec), 2) task_duration_sum,
  ROUND(SUM(weighted_duration_sec), 2) weighted_duration_sum,
  COUNT(1) cnt
FROM
  `chromium-build-stats.ninjalog.users`, UNNEST(log_entries)
WHERE
  created_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY)
GROUP BY
  target_os,
  os,
  outputs
ORDER BY
  weighted_duration_sum DESC
```
