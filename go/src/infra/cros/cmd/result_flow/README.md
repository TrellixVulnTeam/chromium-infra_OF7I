# Result Flow implementations.

This binary's subcommands transform the ChromeOS Test Platform implementation output to Skylab
test results in a user friendly format and insert them into Bigquery.

Result Flow has below subcommands for different Recipes:

## `publish`
Executed twice in cros_test_platform Recipe, `publish` pushes a CTP build ID to Pub/Sub:
- before test execution: notify the subscriber that a new CTP build was kicked off, and its metadata is ready.
- after test execution: notify the subscriber to update CTP build's status.

## `ctp`
Run in Result Flow Recipe, `ctp` pulls a set of CTP Build IDs from the Pub/Sub topic.
By calling Buildbucket API, `ctp` catches the CTP's metadata and status(whether the build is completed), transforms to `test_platform/analyticsTestPlanRun` and uploads to Bigquery.

## `skylab`
Run in Result Flow Recipe, `skylab` catches the test runner build ID from the Test Runner
Pub/Sub topic. Via Buildbucket's GetBuild call, it collects the test runner output, transforms it into `test_platform/analytics/[TestRun|TestCaseResult]`, and inserts to Bigquery.
