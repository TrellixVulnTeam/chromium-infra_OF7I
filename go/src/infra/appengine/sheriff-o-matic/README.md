# sheriff-o-matic

aka SoM

**NOTE: All of the instructions below assume you are working in a single shell
window. All shell commands should be run from the sheriff-o-matic directory
(where this README lives).**

## Setting up local development environment

### Prerequisites

You will need a chrome infra checkout as
[described here](https://chromium.googlesource.com/infra/infra/). That will
create a local checkout of the entire infra repo, but that will include this
application and many of its dependencies.

Warning: If you are starting from scratch, there may be a lot more setup involved
than you expected. Please bear with us.

You'll also need some extras that aren't in the default infra checkout.

```sh
# sudo where appropriate for your setup.

npm install -g bower
```

If you don't have npm or node installed yet, make sure you do so using
`gclient runhooks` to pick up infra's CIPD packages for nodejs and
npm (avoid using other installation methods, as they won't match what
the builders and other infra devs have installed). *Then* make sure you've
run

```sh
eval `../../../../env.py`
```
in that shell window.

### Setting up credentials

You will need access to either staging or prod sheriff-o-matic before you can
do this, so contact chops-tfs-team@google.com to request access ("Please add me
to the relevant AMI roles...") if you don't already have it.

```
# in case you already have this pointed at a jwt file downloaded from gcp console:
unset GOOGLE_APPLICATION_CREDENTIALS

# Use your user identity instead of a service account, will require web flow auth:
gcloud auth application-default login
```

Note that some services (notably, Monorail) will not honor your credentials when
authenticated this way. You'll see `401 Unauthorized` responses in the console logs.
For these, you may need to get service account credentials.
We no longer recommend developers download service account credentials to their machines
because they are more sensitive (and GCP limits how many we can have out in the wild).

### Getting up and running locally

Note: if you would like, you can test on staging environment and skip the local
setup sections.

After initial checkout, make sure you have all of the bower dependencies
installed. Also run this whenever bower.json is updated:

```sh
make build
```

(Note that you should always be able to `rm -rf frontend/bower_components`
and re-run `bower install` at any time. Occasionally there are changes that,
when applied over an existing `frontend/bower_components`, will b0rk your
checkout.)

To run locally from an infra.git checkout:
```sh
make devserver
```

To run tests:
```sh
# Default (go and JS):
make test

# For go:
go test infra/appengine/sheriff-o-matic/som/...

# For interactive go, automatically re-runs tests on save:
cd som && goconvey

# For JS:
cd frontend
make wct

# For debugging JS, with a persistent browser instance you can reload:
cd frontend
make wct_debug
```

To view test coverage report after running tests:
```sh
google-chrome ./coverage/lcov-report/index.html
```

### Adding some trees to your local SoM

Once you have a server running locally, you'll want to add at least one
tree configuration to the datastore. Make sure you are logged in locally
as an admin user (admin checkbox on fake devserver login page).

Navigate to [localhost:8080/admin/portal](http://localhost:8080/admin/portal)
and fill out the tree(s) you wish to test with locally. For consistency, you
may just want to copy the [settings from prod](http://sheriff-o-matic.appspot.com/admin/portal).

If you don't have access to prod or staging, you can manually enter this for
"Trees in SOM" to get started with a reasonable default:

```
android:Android,chrome_browser_release:Chrome Browser Release,chromeos:Chrome OS,chromium:Chromium,chromium.clang:Chromium Clang,chromium.gpu:Chromium GPU,chromium.perf:Chromium Perf,fuchsia:Fuchsia,ios:iOS,lacros_skylab:Lacros Skylab,angle:Angle
```

After this step, you should see the trees appearing in SoM, but without any
alerts. To populate the alerts, continue to the next section.

### Populating alerts from local cron tasks

To populate alerts for ChromeOS or Fuchsia tree, run
[http://localhost:8081/_cron/analyze/chromeos](http://localhost:8081/_cron/analyze/chromeos) or
[http://localhost:8081/_cron/analyze/fuchsia](http://localhost:8081/_cron/analyze/fuchsia) accordingly.

To populate alerts for other trees, firstly you must run
[http://localhost:8081/_cron/bq_query/chrome](http://localhost:8081/_cron/bq_query/chrome).
This will populate the memcache. After that, you can run, for example
[http://localhost:8081/_cron/analyze/chromium](http://localhost:8081/_cron/analyze/chromium)
to populate chromium tree.

There is a difference because other trees (aside from ChromeOS and Fuchsia)
reads data from memcache (instead of querying BigQuery directly) in order to
save cost.

The purpose of the cronjobs is to process data from SoM's BigQuery view and
populate Datastore with alerts.

## Deployment

### Authenticating for deployment

In order to deploy to App Engine, you will need to be a member of the
project (either sheriff-o-matic or sheriff-o-matic-staging). Before your first
deployment, you will have to run `./gae.py login` to authenticate yourself.

### Deploying to staging

Sheriff-o-Matic has a staging server sheriff-o-matic-staging.appspot.com.
To deploy to staging:

- Make sure you have the right to deploy to staging, if not, please see the
"Contributors" section below.
- run `make deploy_staging`
- Optional: Go to the Versions section of the
[App Engine Console](https://appengine.google.com/) and update both the default
and backend versions of the app.
- Check https://viceroy.corp.google.com/chrome_infra/Appengine/sheriff_o_matic_staging?duration=1h
and make sure everything is ok.

Note: At the moment, the staging server uses build data from buildbucket prod,
but test data from resultdb staging. Also, as staging and prod has different
datastore, information like alert grouping, bug attached, ... may not match the
data in prod.

Currently, the cron jobs to populate data from BigQuery to datastore is
scheduled to run once a day, so if you want the latest data, you may want to
run the cron jobs manually.


### Deploying to production

If you want to release a new version to prod:

- Run `make deploy_prod`
- Double-check that the version is not named with a `-tainted` suffix,
as deploying such a version will cause alerts to fire (plus, you shouldn't
deploy uncommitted code :).
- Go to the Versions section of the
[App Engine Console](https://appengine.google.com/) and update the default
version of the app services. **Important**: *Remember to update both the
"default" and "analyzer" services* by clicking the "Migrate traffic" button.
Having the default and analyzer services running different versions may cause
errors and/or monitoring alerts to fire.
- Wait for a while, making sure that the graphs looks fine and there is no
abnormality in https://viceroy.corp.google.com/chrome_infra/Appengine/sheriff_o_matic_prod?duration=1h
- Verify the the cron jobs are still successful with the new version (currently
they are not sending alerts when they fail, so you need to check manually).
- Do some validity check by clicking through the trees in the UI.
- Send a PSA email to cit-sheriffing@ (cc chops-tfs-team@) about the new
release, together with the release notes.
- You can get the release notes by running (note that you may need to
authenticate for deployment first).

```sh
make relnotes
```

You can also use the optional flags `-since-date YYYY-MM-DD` or
`-since-hash=<git short hash>` if you need to manually specify the range
of commits to include, using the command

```sh
go run ../../tools/relnotes/relnotes.go -since-hash <commit_hash> -app sheriff-o-matic -extra-paths .,../../monitoring/analyzer,../../monitoring/client,../../monitoring/messages
```

Tips: You can find the commit hash of a version by looking at the version name
in appengine (Go to pantheon page for your app, and click at Versions section).
For example, if your version name is 12345-20d8b52, then the commit hash is
20d8b52.

### Deploying changes to BigQuery views

Changes to SoM's BigQuery view schema are deployed separately from AppEngine
deployment described above. This happens when  you modify the SQL files for
bigquery views (the sql files in ./bigquery folder).  The steps to deploy your
changes are as follows:

- `cd ./bigquery`
- Run `./create_views.sh` to deploy your change to staging
- Verify that everything works as expected
- Create a CL with your changes and get it reviewed
- Land your change
- Modify the file create_views.sh to point to prod by setting `APP_ID=sheriff-o-matic`
- Run `./create_views.sh` again to deploy your change to prod
- Verify that everything works as expected in prod
- Revert the change in create_views.sh by setting `APP_ID=sheriff-o-matic-staging`

If you want to revert your deployment, simply checkout the main git branch and
run `./create_view.sh` again for staging and prod.

Note: As there is no record of the deployment of BigQuery views, it is important
that you only deploy to prod once your CL is landed, so it will be easier to
debug later if something go wrong.

## Dataflow
A (simplified) dataflow in SoM is in this [diagram](https://docs.google.com/presentation/d/1onUTH9QSE5Y65Vlp7pm6UPM7LRIwWHvwc_yJoEXyfDg).

## Assigning builds to trees

### Chromium/Chrome

In the case of the **chromium** and **chrome** projects as well as their
corresponding branch projects (**chromium-m\*** and **chrome-m\***), builds are
assigned to trees based on the value of the `sheriff_rotations` property. The
`sheriff_rotations` property is a list containing the names of the trees the
build should be included in.

To modify the `sheriff_rotations` property for a builder's builds, update the
definition of the builder by setting the `sheriff_rotations` argument, which can
take a single value or a list of values:

```starlark
builder(
    name = "my-builder",
    ...
    sheriff_rotations = sheriff_rotations.ANDROID,
    ...
)
```

The builders are organized in files based on builder groups, which often are all
assigned the same tree. In that case, the `sheriff_rotations` value can be set
for the entire file by using module-level defaults:

```starlark
defaults.set(
    builder_group = "my-builder-group",
    ...
    sheriff_rotations = sheriff_rotations.CHROMIUM,
    ...
)
```

Any values specified at the builder will be merged with those set in the
module-level defaults. If the module-level defaults should be ignored,
`args.ignore_default` can be used to take only what is specified at the builder,
so the following example would cause the `sheriff_rotations` property to not be
set regardless of the module-level default value:

```starlark
builder(
    name = "my-unsheriffed-builder",
    ...
    sheriff_rotations = args.ignore_default(None),
    ...
)
```

## Troubleshooting common problems
### SoM is showing incorrect/missing/stale data
Firstly we should check if the [cron jobs](https://pantheon.corp.google.com/cloudscheduler?folder=&organizationId=&src=ac&project=sheriff-o-matic)
are running or not, and if the latest runs were successful. Maybe it also helps
to look at recent cron logs to verify that there are no unexpected error logs.

If the cronjobs are successful, we should check if the alerts are configured to
be shown in SoM or not. The conditions for each trees are in [bigquery_analyzer.go](https://source.chromium.org/chromium/infra/infra/+/HEAD:go/src/infra/appengine/sheriff-o-matic/som/analyzer/bigquery_analyzer.go;l=65).
The alerts will then go through a filter in [config.json](https://source.chromium.org/chromium/infra/infra/+/HEAD:go/src/infra/appengine/sheriff-o-matic/config/config.json).

If the alerts are supposed to show, but do not get shown in SoM, the next step
is to look at Datastore (AlertJSONNonGrouping table) to see if we can find the
alert there. If not, go to BigQuery, sheriffable_failures table to see if we
can find the failure there. If the failure is in BigQuery, but not in
DataStore, there may be a bug in the analyzer cron job, which will require
further investigation.

If the failure cannot be found in SoM's BQ table, check if the build can be
found in buildbucket table (e.g. cr-buildbucket.raw.completed_builds_prod). If
it can be found, then maybe there is a problem with the create_views.sh script,
and will require further investigation.

### Make SoM monitor additional builders
Most of the time, those requests can be fulfilled by modifying the bigquery_analyzer.go
file. We need to make sure those builders/steps are not filtered out in [config.json](https://source.chromium.org/chromium/infra/infra/+/HEAD:go/src/infra/appengine/sheriff-o-matic/config/config.json).


## Contributors

We don't currently run the `WCT` tests on CQ. So *please* be sure to run them
yourself before submitting. Also keep an eye on test coverage as you make
changes. It should not decrease with new commits.

If you would like to test your changes on our staging server (this is often
necessary in order to test and debug integrations, and some issues will
only reliably reproduce in the actual GAE runtime rather than local devserver),
please contact chops-tfs-team@google.com to request access. We're happy to
grant staging access to contributors!
