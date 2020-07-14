# cr\_rev
cr-rev is a service which provides short URLs that redirect to code reviews,
individual commits and other useful items for the greater Chromium project.

## Deployment on development
To deploy the app onto development environment, run:
- `eval \`../../../../env.py\``
- `gae.py upload -A cr-rev-dev`
