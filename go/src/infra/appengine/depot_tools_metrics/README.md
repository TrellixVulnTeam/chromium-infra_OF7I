# depot\_tools monitoring app
This is a simple GAE app to collect the metrics uploaded by depot\_tools and
store them in BigQuery.
This app is only reachable when the *X-AppEngine-Trusted-IP-Request* header,
which is set only on requests received from corp machines, is present on the
request. This is a way to ensure we don't collect data from non-Googlers.

## API
This app exposes two endpoints:
- **/should-upload**

  Returns 200 if the request comes from a corp machine, and 403 otherwise.
- **/upload**

  Accepts a JSON file in the format described by `monitoring_logs_schema.json`
  and writes the data to the `depot_tools` table in the `metrics` dataset of the
  `cit-cli-metrics` project.

  Returns:
  - 403 if the request comes from a non-corp machine.
  - 400 if the reported metrics are invalid.
  - 500 if there was an internal error.
  - 200 if the request succeeded.

## Deployment
### Updating the Schema
To update the metrics table and schema, run:
- `go generate schema/gen.go`
- `bqschemaupdater -table cit-cli-metrics.metrics.depot_tools -message-dir
  schema -message schema.Metrics`

If these steps modified any files in the repo, add them to a change for review.
By the end of this process, running these steps on a clean checkout of the main
branch should not result in further updates to the schema.

### Uploading a New Version
Make sure you're on the main branch and that your working tree is clean. Run
`gae.py upload`. If everything worked, you should see a new version with name
`{ID}-{commit hash}` in the cit-cli-metrics dashboard.

If `gae.py upload` doesn't work, try `gae.py upload --app-id cit-cli-metrics
--app-dir .`

### Migrating Traffic
After uploading, navigate to the cit-cli-metrics dashboard and migrate traffic
to your new version.
