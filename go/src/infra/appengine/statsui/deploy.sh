#!/bin/sh

cd frontend &&
  npm run build &&
  cd .. &&
  gcloud app deploy --project google.com:chrome-infra-stats
