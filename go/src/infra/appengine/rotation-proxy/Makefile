GAE_PY = ../../../../../luci/appengine/components/tools/gae.py

default: help

help:
	@echo "Available commands:"
	@sed -n '/^[a-zA-Z0-9_]*:/s/:.*//p' <Makefile

format:
	gofmt -s -w .

test:
	go test

deploy:
	$(GAE_PY) upload -A chops-rotation-proxy-test

deploy-prod:
	$(GAE_PY) upload -A chrome-ops-rotation-proxy

gen:
	cproto proto
