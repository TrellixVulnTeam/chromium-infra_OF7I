#!/bin/sh

cd frontend &&
  npm run build &&
  cd .. &&
  gae.py upload --switch --app-id=google.com:chrome-infra-stats
