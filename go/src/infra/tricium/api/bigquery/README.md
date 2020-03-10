# Tricium BigQuery Schema

This includes the schema for rows in the Tricium BigQuery tables, as well as
related scripts to update the table schemas and views.

## Creating tables and updating table schemas.

All tables belong to a dataset, which has a name and description. Datasets can
be created with the gcloud SDK `bq` command, which must be installed first.

Datasets can be created with command that look like:

```
bq --location=US mk --dataset --description "Analyzer runs" tricium-dev:analyzer
```

Schemas can be updated with `bqschemaupdater`. You should make sure you run an
up-to-date `bqschemaupdater`:

```
go install go.chromium.org/luci/tools/cmd/bqschemaupdater
```

This tool takes a proto message and table and updates the schema based on the
compiled proto message. Proto messages should be compiled by running `go
generate`. The commands to update the schemas look like this:

```
bqschemaupdater -message apibq.AnalysisRun -table tricium-dev.analyzer.results
bqschemaupdater -message apibq.FeedbackEvent -table tricium-dev.events.feedback
```
