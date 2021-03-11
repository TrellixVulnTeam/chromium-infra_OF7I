# Stats UI SQL Scripts

SQL statements to materialize the data shown in the stats UI.

These scripts are run as BigQuery scheduled queries under the
`chrome-trooper-analytics` project.

## Deploying

```sh
./deploy.sh
```

This will run the script and compare the SQL locally to the SQL in GCP. If they
do not match, the local version will be uploaded.

## Alternatives considered

This tool was created because of a lack of other options to run these
materialization scripts. Here are the alternative approaches considered and what
the issues were

*   BQ command-line tool: While this tool can create BigQuery scheduled queries,
    it always creates new ones and cannot replace existing queries.
*   Internal workflows: Currently does not support the `merge` statement.
*   Terraform:
    [Terraform](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_data_transfer_config)
    is probably a viable option. It does require the team to adopt new tools and
    workflows however. We could migrate to terraform in the future as Terraform
    is much better supported.

## TODO

Explore some form of templating with `deploy.go` ([CL Discussion](http://cl/356352576))
