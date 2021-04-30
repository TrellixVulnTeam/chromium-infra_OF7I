# Monorail Issue Tracker

Monorail is the Issue Tracker used by the Chromium project and other related
projects. It is hosted at [bugs.chromium.org](https://bugs.chromium.org).

If you wish to file a bug against Monorail itself, please do so in our
[self-hosting tracker](https://bugs.chromium.org/p/monorail/issues/entry).
We also discuss development of Monorail at `infra-dev@chromium.org`.

# Getting started with Monorail development

*For Googlers:* Monorail's codebase is open source and can be installed locally on your workstation of choice.

Here's how to run Monorail locally for development on MacOS and Debian stretch/buster or its derivatives.

1.  You need to [get the Chrome Infra depot_tools commands](https://commondatastorage.googleapis.com/chrome-infra-docs/flat/depot_tools/docs/html/depot_tools_tutorial.html#_setting_up) to check out the source code and all its related dependencies and to be able to send changes for review.
1.  Check out the Monorail source code
    1.  `cd /path/to/empty/workdir`
    1.  `fetch infra`
    1.  `cd infra/appengine/monorail`
1.  Make sure you have the AppEngine SDK:
    1.  It should be fetched for you by step 1 above (during runhooks)
    1.  Otherwise, you can download it from https://developers.google.com/appengine/downloads#Google_App_Engine_SDK_for_Python
1.  Spin up dependent services.
    1. We use docker and docker-compose to orchestrate. So install [docker](https://docs.docker.com/get-docker/) and [docker-compose](https://docs.docker.com/compose/install/) first. For glinux users see [go/docker](http://go/docker)
    1.  Make sure to authenticate with the App Engine SDK and configure Docker. This is needed to install Cloud Tasks Emulator.
        1.  `gcloud auth login`
        1.  `gcloud auth configure-docker`
    1. Run `docker-compose -f dev-services.yml up -d`. This should spin up:
        1. MySQL v5.6
        1. Redis
        1. Cloud Tasks Emulator
            1. [TODO](https://github.com/aertje/cloud-tasks-emulator/issues/4) host this publicly and remove section.
            1. This will require you to authenticate to Google Container Registry to pull the docker image: `gcloud auth login` `gcloud auth configure-docker`. If you're an open source developer and do not have access to the monorail project and thereby its container registry you will need to start the Cloud Tasks Emulator from [source](https://github.com/aertje/cloud-tasks-emulator)
1.  Set up SQL database. (You can keep the same sharding options in settings.py that you have configured for production.).
    1. Copy setup schema into the docker container
        1.  `docker cp schema/. mysql:/schema`
        1.  `docker exec -it mysql bash`
        1.  `mysql --user=root monorail < schema/framework.sql`
        1.  `mysql --user=root monorail < schema/project.sql`
        1.  `mysql --user=root monorail < schema/tracker.sql`
        1.  `exit`
1.  Configure the site defaults in settings.py.  You can leave it as-is for now.
1.  Set up the front-end development environment:
    1. On Debian
        1.  ``eval `../../go/env.py` `` -- you'll need to run this in any shell you
            wish to use for developing Monorail. It will add some key directories to
            your `$PATH`.
        1.  Install build requirements:
            1.  `sudo apt-get install build-essential automake`
    1. On MacOS
        1.  Install node and npm
            1.  Install node version manager `brew install nvm`
            1.  See the brew instructions on updating your shell's configuration
            1.  Install node and npm `nvm install 12.13.0`
1.  Install Python and JS dependencies:
    1.  Install MySQL, needed for mysqlclient
        1. For mac: `brew install mysql@5.6`
        1. For Debian derivatives, download and unpack [this bundle](https://dev.mysql.com/get/Downloads/MySQL-5.6/mysql-server_5.6.40-1ubuntu14.04_amd64.deb-bundle.tar): `tar -xf mysql-server_5.6.40-1ubuntu14.04_amd64.deb-bundle.tar`. Install the packages in the order of `mysql-common`,`mysql-community-client`, `mysql-client`, then `mysql-community-server`.
    1.  Optional: You may need to install `pip`. You can verify whether you have it installed with `which pip`.
        1. `curl -O https://bootstrap.pypa.io/2.7/get-pip.py`
        1. `sudo python get-pip.py`
    1.  Optional: Use `virtualenv` to keep from modifying system dependencies.
        1. `sudo pip install virtualenv`
        1. `virtualenv venv` to set up virtualenv within your monorail directory.
        1. `source venv/bin/activate` to activate it, needed in each terminal instance of the directory.
    1.  Mac only: install [libssl](https://github.com/PyMySQL/mysqlclient-python/issues/74), needed for mysqlclient.
        1. `brew install openssl; export LIBRARY_PATH=$LIBRARY_PATH:/usr/local/opt/openssl/lib/`
    1.  `make dev_deps`
    1.  `make deps`
1.  Run the app:
    1.  `make serve`
1.  Browse the app at localhost:8080 your browser.
1.  Optional: Create/modify your Monorail User row in the database and make that user a site admin. You will need to be a site admin to gain access to create projects through the UI.
    1.  `docker exec mysql mysql --user=root monorail -e "UPDATE User SET is_site_admin = TRUE WHERE email = 'test@example.com';"`
    1.  If the admin change isn't immediately apparent, you may need to restart your local dev appserver.

Instructions for deploying Monorail to an existing instance or setting up a new instance are [here](doc/deployment.md).

Here's how to run unit tests from the command-line:

## Testing

To run all Python unit tests, in the `appengine/monorail` directory run:

```
make test
```

For quick debugging, if you need to run just one test you can do the following. For instance for the test
`IssueServiceTest.testUpdateIssues_Normal` in `services/test/issue_svc_test.py`:

```
../../test.py test appengine/monorail:services.test.issue_svc_test.IssueServiceTest.testUpdateIssues_Normal --no-coverage
```

### Frontend testing

To run the frontend tests for Monorail, you first need to set up your Go environment. From the Monorail directory, run:

```
eval `../../go/env.py`
```

Then, to run the frontend tests, run:

```
make karma
```

If you want to skip the coverage for karma, run:
```
make karma_debug
```

To run only one test or a subset of tests, you can add `.only` to the test
function you want to isolate:

```javascript
// Run one test.
it.only(() => {
  ...
});

// Run a subset of tests.
describe.only(() => {
  ...
});
```

Just remember to remove them before you upload your CL.

## Troubleshooting

*   `BindError: Unable to bind localhost:8080`

This error occurs when port 8080 is already being used by an existing process. Oftentimes,
this is a leftover Monorail devserver process from a past run. To quit whatever process is
on port 8080, you can run `kill $(lsof -ti:8080)`.

*   `TypeError: connect() got an unexpected keyword argument 'charset'`

This error occurs when `dev_appserver` cannot find the MySQLdb library.  Try installing it via <code>sudo apt-get install python-mysqldb</code>.

*   `TypeError: connect() argument 6 must be string, not None`

This occurs when your mysql server is not running.  Check if it is running with `ps aux | grep mysqld`.  Start it up with <code>/etc/init.d/mysqld start </code>on linux, or just <code>mysqld</code>.

*   dev_appserver says `OSError: [Errno 24] Too many open files` and then lists out all source files

dev_appserver wants to reload source files that you have changed in the editor, however that feature does not seem to work well with multiple GAE modules and instances running in different processes.  The workaround is to control-C or `kill` the dev_appserver processes and restart them.

*   `IntegrityError: (1364, "Field 'comment_id' doesn't have a default value")` happens when trying to file or update an issue

In some versions of SQL, the `STRICT_TRANS_TABLES` option is set by default. You'll have to disable this option to stop this error.

*   `ImportError: No module named six.moves`

You may not have six.moves in your virtual environment and you may need to install it.

1.  Determine that python and pip versions are possibly in vpython-root
    1.  `which python`
    1.  `which pip`
1.  If your python and pip are in vpython-root
    1.  ```sudo `which python` `which pip` install six```

# Development resources

## Supported browsers

Monorail supports all browsers defined in the [Chrome Ops guidelines](https://chromium.googlesource.com/infra/infra/+/main/doc/front_end.md).

File a browser compatability bug
[here](https://bugs.chromium.org/p/monorail/issues/entry?labels=Type-Defect,Priority-Medium,BrowserCompat).

## Frontend code practices

See: [Monorail Frontend Code Practices](doc/code-practices/frontend.md)

## Monorail's design

* [Monorail Data Storage](doc/design/data-storage.md)
* [Monorail Email Design](doc/design/emails.md)
* [How Search Works in Monorail](doc/design/how-search-works.md)
* [Monorail Source Code Organization](doc/design/source-code-organization.md)
* [Monorail Testing Strategy](doc/design/testing-strategy.md)

## Triage process

See: [Monorail Triage Guide](doc/triage.md).

## Release process

See: [Monorail Deployment](doc/deployment.md)

# User guide

For information on how to use Monorail, see the [Monorail User Guide](doc/userguide/README.md).

## Setting up a new instance of Monorail

See: [Creating a new Monorail instance](doc/instance.md)
