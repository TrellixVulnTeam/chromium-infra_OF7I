# cr\_rev
cr-rev is a service which provides short URLs that redirect to code reviews,
individual commits and other useful items for the greater Chromium project.

## Running tests
Run:

    goconvey

## Development deployment
To deploy the app onto the development environment, run:

    eval `../../../../env.py`
    gae.py upload --app-id cr-rev-dev --app-dir .

## Production deployment
To deploy the app onto the production environment, run:

    eval `../../../../env.py`
    gae.py upload --app-id cr-rev-prod --app-dir .
