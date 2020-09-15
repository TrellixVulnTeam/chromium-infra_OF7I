# cr-rev
cr-rev is a service which provides short URLs that redirect to code reviews,
individual commits and other useful items for the greater Chromium project.

Before running any targets, run the following command:

    eval `../../../../env.py`

## Running tests
Run:

    make test

If you want to run tests interactively, you can use:

    goconvey

## Development deployment
To deploy the app onto the development environment, run:

    make dev

## Production deployment
To deploy the app onto the production environment, run:

    make prod
