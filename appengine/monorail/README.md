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
    1.  `fetch infra` (make sure you are not "fetch internal_infra" )
    1.  `cd infra/appengine/monorail`
1.  Make sure you have the AppEngine SDK:
    1.  It should be fetched for you by step 1 above (during runhooks)
    1.  Otherwise, you can download it from https://developers.google.com/appengine/downloads#Google_App_Engine_SDK_for_Python
    1.  Also follow https://cloud.google.com/appengine/docs/standard/python3/setting-up-environment to setup `gcloud`
1.  Install CIPD dependencies:
    1. `gclient runhooks`
1.  Install MySQL v5.6.
    1. On Mac, use [homebrew](https://brew.sh/) to install MySQL v5.6:
            1.  `brew install mysql@5.6`
    1. Otherwise, download from the [official page](http://dev.mysql.com/downloads/mysql/5.6.html#downloads).
        1.  **Do not download v5.7 (as of April 2016)**
1.  Set up SQL database. (You can keep the same sharding options in settings.py that you have configured for production.).
    1. Copy setup schema into your local MySQL service.
        1.  `mysql --user=root -e 'CREATE DATABASE monorail;'`
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
        1. [Install homebrew](https://brew.sh)
        1.  Install node and npm
            1.  Install node version manager `brew install nvm`
            1.  See the brew instructions on updating your shell's configuration
            1.  Install node and npm `nvm install 12.13.0`
            1.  Add the following to the end of your `~/.zshrc` file:

                    export NVM_DIR="$HOME/.nvm"
                    [ -s "/usr/local/opt/nvm/nvm.sh" ] && . "/usr/local/opt/nvm/nvm.sh"  # This loads nvm
                    [ -s "/usr/local/opt/nvm/etc/bash_completion.d/nvm" ] && . "/usr/local/opt/nvm/etc/bash_completion.d/nvm"  # This loads nvm bash_completion

1.  Install Python and JS dependencies:
    1.  Optional: You may need to install `pip`. You can verify whether you have it installed with `which pip`.
        1. make sure to install `pip` using `python2` instead of `python3` (use `python --version` to check the version for 2.7, `which python2` to check the path)
            1. If you need python 2.7 for now: `sudo apt install python2.7 python2.7-dev python-is-python2`
        1. `curl -O /tmp/get-pip.py https://bootstrap.pypa.io/pip/2.7/get-pip.py`
        1. `sudo python /tmp/get-pip.py`
    1.  Use `virtualenv` to keep from modifying system dependencies.
        1. `pip install virtualenv`
        1. `python -m virtualenv venv` to set up virtualenv within your monorail directory.
        1. `source venv/bin/activate` to activate it, needed in each terminal instance of the directory.
    1.  Mac only: install [libssl](https://github.com/PyMySQL/mysqlclient-python/issues/74), needed for mysqlclient. (do this in local env not virtual env)
        1. `brew install openssl; export LIBRARY_PATH=$LIBRARY_PATH:/usr/local/opt/openssl/lib/`
    1.  `make dev_deps` (run in virtual env)
    1.  `make deps` (run in virtual env)
1.  Run the app:
    1.  `make serve` (run in virtual env)
    1.  Start MySQL:
        1. Mac: `brew services restart mysql@5.6`
        1. Linux: `mysqld`
1. Browse the app at localhost:8080 your browser.
1. Set up your test user account (these steps are a little odd, but just roll with it):
       1.  Sign in using `test@example.com`
       1.  Log back out and log in again as `example@example.com`
       1.  Log out and finally log in again as `test@example.com`.
       1.  Everything should work fine now.
1.  Modify your Monorail User row in the database and make that user a site admin. You will need to be a site admin to gain access to create projects through the UI.
    1.  `mysql --user=root monorail -e "UPDATE User SET is_site_admin = TRUE WHERE email = 'test@example.com';"`
    1.  If the admin change isn't immediately apparent, you may need to restart your local dev appserver. If you kill the dev server before running the docker command, the restart may not be necessary.

Instructions for deploying Monorail to an existing instance or setting up a new instance are [here](doc/deployment.md).


## Feature Launch Tracking

To set up FLT/Approvals in Monorail:
1. Visit the gear > Development Process > Labels and fields
1. Add at least one custom field with type "Approval" (this will be your approval
1. Visit gear > Development Process > Templates
1. Check "Include Gates and Approval Tasks in issue"
1. Fill out the chart - The top row is the gates/phases on your FLT issue and you can select radio buttons for which gate each approval goes

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

*   `RuntimeError: maximum recursion depth exceeded while calling a Python object`

If running `make serve` gives an output similar to [this](https://paste.googleplex.com/4693398234595328),
1.  make sure you're using a virtual environment (see above for how to configure one). Then, make the changes outlined in [this CL](https://chromium-review.googlesource.com/c/infra/infra/+/3152656).
1.  Also try `pip install protobuf`

*   `gcloud: command not found`

Add the following to your `~/.zshrc` file: `alias gcloud='/Users/username/google-cloud-sdk/bin/gcloud'`. Replace `username` with your Google username.

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

*  `enum hash not match` when run `make dev_peds`

You may run the app using python3 instead of python2.

1. Determine the python version used in virtual environment `python --version` if it's 3.X

   `deactivate`

   `rm -r venv/`

    `pip uninstall virtualenv`

    `pip uninstall pip`

   in local environment `python --version` make sure to change it to python2

   follow previous to instructions to reinstall `pip` and `virtualenv`

* `mysql_config not found` when run `make dev_deps`

  this may be caused installing the wrong version of six. run `pip list` in virtual env make sure it is 1.15.0
  if not

   `deactivate`

   `rm -r venv/`

   `pip uninstall six`

   `pip install six==1.15.0`

   `virtualenv venv`

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
