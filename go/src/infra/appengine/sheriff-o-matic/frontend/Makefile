.PHONY: test
VULCANIZE_PATH = ./node_modules/vulcanize/bin/vulcanize
WCT_DIR = ../../../tools/wct
WCT_GO = $(WCT_DIR)/wct.go $(WCT_DIR)/loader.go
default: help

help:
	@echo "Available commands:"
	@sed -n '/^[a-zA-Z0-9_]*:/s/:.*//p' <Makefile

node_deps:
	npm ci

bower_deps: node_deps
	npx bower install

lint:
	polylint -i elements/som-app/som-app.html

build_polymer: bower_deps
	$(VULCANIZE_PATH) elements/som-app/som-app.html -o elements/som-app/som-app.vulcanized.html --inline-scripts

typecheck: node_deps
	npm run types

build_lit: node_deps typecheck
	npm run build

build: build_polymer build_lit

clean:
	rm scripts/bundle.js*
	rm elements/som-app/som-app.vulcanized.html

wct: bower_deps
	npx run-wct

wct_debug: bower_deps
	npx run-wct --debug
