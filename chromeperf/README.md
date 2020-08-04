# Chromeperf Project Directory

This directory contains source code for all Python services for the
Chromeperf project. This will include:

- A port of the execution engine.
- Migrated code for anomaly detection.

Code for existing services for the Chromeperf project are currently hosted in
the
[catapult](https://source.chromium.org/chromium/chromium/src/+/master:third_party/catapult/)
repository.

## Development

We're following standard open source Python package development and best
practices. Specifically, we're using the following packages and dependencies:

- pytest: A test runner and testing framework which works well with the
standard Python unittest framework.

- tox: A testing automation tool which uses Python's standard venv package to
manage testing environments.

One recommendation we're following is the separation of tests from the
package code, which is why we have the `src` directory where all
implementation modules are defined.

## Setup

You should install `virtualenv` for Python3, and create a virtual environment
to install local dependencies.

```bash
pip3 install --user virtualenv
python3 -m venv $HOME/chromeperf-venv
```

This will create the directory `$HOME/chromeperf-venv` where all the
dependency packages will be installed. Once that's done you can activate the
virtual environment setup using the `activate` script in
`$HOME/chromeperf-venv/bin`:

```bash
source $HOME/chromeperf-venv/bin/activate
```

You can learn more about `venv` and how to use it at:

https://packaging.python.org/guides/installing-using-pip-and-virtual-environments/#creating-a-virtual-environment

Part of the process involves installing all the requirements for developing
the core libraries and services in this directory. We can do this by
installing all the requirements from `requirements.txt`:

```bash
pip install -r requirements.txt
```

## Testing

Following open source Python package best practices, we're developing the
`chromeperf` core libraries as if it can be installed using `pip`. The `tox`
package allows us to do that by automating the setup of temporary virtual
environments for hermetic and reproducible testing:

```bash
tox
```

You can learn more about `tox` at:

https://tox.readthedocs.io/en/latest/index.html
