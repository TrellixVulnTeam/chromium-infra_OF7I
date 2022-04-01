.PHONY: help build clean test devserver deploy_staging deploy_prod
DEV_CLOUD_PROJECT = eswar-celab-prototype

help:
	@echo "Available commands:"
	@sed -n '/^[a-zA-Z0-9_]*:/s/:.*//p' <Makefile

build:
	cd frontend && npm run build

clean:
	cd frontend && rm -f src/build/*

test:
	go test -race -count 10 -failfast ./...
	cd frontend && npm run test

emulator:
	set DATASTORE_DATASET=$(DEV_CLOUD_PROJECT)
	set DATASTORE_EMULATOR_HOST=localhost:8081
	set DATASTORE_EMULATOR_HOST_PATH=localhost:8081/datastore
	set DATASTORE_HOST=http://localhost:8081
	set DATASTORE_PROJECT_ID=$(DEV_CLOUD_PROJECT)
	gcloud beta emulators datastore start --consistency=1.0

backend-dev:
	go run . -cloud-project $(DEV_CLOUD_PROJECT)

frontend-dev:
	cd frontend && npm start

runlocal:
	$(MAKE) -j3 emulator backend-dev frontend-dev
