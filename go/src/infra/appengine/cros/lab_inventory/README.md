# Chrome OS Lab Inventory AppEngine Service

Chrome OS Lab inventory is a service which manages lab configs of all devices
in ChromeOS.

Services maintained:
  - Inventory v2 (Go Service)
  - Manual Repair App (LitElement, Redux, TypeScript)

### Full Deployment
For service-based deployments, please refer to the Deployment sections of each service in this readme.

To deploy all services to staging, run the following:

```bash
# Upload and promote the Go service and Manual Repair App to staging
make up-dev-all
```

To deploy all services to production, run the following:

```bash
# Deploy the Go service and Manual Repair App to production without routing traffic to them
make up-prod-all

# Route all traffic to the uploaded Go service
make switch-prod-all
```

To route new traffic for the Prod version of Manual Repair App, please visit [GAE dashboard](https://pantheon.corp.google.com/appengine/versions?project=cros-lab-inventory&serviceId=manual-repair) and switch to the appropriate version.

## Inventory v2 Go Service
### Development
Run this for a devserver at http://localhost:8082:

```bash
make dev
```

### Deployment
To deploy staging, run the following:

```bash
# Upload and promote the Go service to staging
make up-dev
```

To deploy production, run the following:

```bash
# Deploy the Go service to production without routing traffic to it
make up-prod

# Route all traffic to the uploaded Go service
make switch-prod
```

## Manual Repair App
### Setup

The current working directory is `$SRC_ROOT/infra/appengine/cros/lab_inventory/app/ui/manual-repair`, i.e. the directory that contains this file. Please `cd` into it for the commands below to work.

To get started, run:

```bash
make mr-setup
```

### Development

The project uses `webpack` and `webpack-dev-server`. From command line in `$SRC_ROOT/infra/appengine/cros/lab_inventory/`, you can run:

```bash
make mr-dev
```

Then open http://localhost:8080 for the home page.

### Deployment

The app is built to be deployed with Google App Engine. The `app.yaml` is split into `app.stage.yaml` and `app.prod.yaml`. Currently, the only difference is the `NODE_ENV` environment variable. All Typescript is bundled into `dist/app.js` and loaded by the client browser in `index.html`.

Deploying this app will also deploy `$SRC_ROOT/infra/appengine/cros/lab_inventory/app/dispatch.yaml`, allowing routing to be set up between the Go service and Manual Repair.

Note that this deployment is only for the Manual Repair app and not the Go server.

To deploy staging, run the following. Stage will be automatically promoted:

```bash
# Upload and promote the Manual Repair App to staging
make mr-up-dev
```

To deploy production, run the following:

```bash
# Deploy the Manual Repair App to production without routing traffic to it
make mr-up-prod
```

To route new traffic for the Prod version, please visit [GAE dashboard](https://pantheon.corp.google.com/appengine/versions?project=cros-lab-inventory&serviceId=manual-repair) and switch to the appropriate version.

After following the on-screen prompts, the application will be deployed.
  - Stage: https://manual-repair-dot-cros-lab-inventory-dev.appspot.com/
  - Prod: https://manual-repair-dot-cros-lab-inventory.appspot.com/
