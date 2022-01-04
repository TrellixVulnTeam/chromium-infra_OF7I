# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
projdir := $(patsubst %/,%,$(dir $(mkfile_path)))

build:
	go vet   $(projdir)/...
	go test  $(projdir)/...
	go build -o $(projdir)/gaev2/crosadm/crosadm $(projdir)/gaev2/crosadm
	go build -o $(projdir)/gaev2/adminclient/adminclient $(projdir)/gaev2/adminclient

dev:
	gae.py devserver -A dev --app-dir $(projdir)/internal/app/appengine -- --host 0.0.0.0 --port 8082 --admin_port 7999 --log_level debug

up-prod:
	gae.py upload -A chromeos-skylab-bot-fleet --app-dir $(projdir)/internal/app/appengine

switch-prod:
	gae.py switch -A chromeos-skylab-bot-fleet --app-dir $(projdir)/internal/app/appengine

versions-cleanup-prod:
	gae.py cleanup -A chromeos-skylab-bot-fleet --app-dir $(projdir)/internal/app/appengine

up-staging:
	gae.py upload -A skylab-staging-bot-fleet --app-dir $(projdir)/internal/app/appengine

switch-staging:
	gae.py switch -A skylab-staging-bot-fleet --app-dir $(projdir)/internal/app/appengine

versions-cleanup-staging:
	gae.py cleanup -A skylab-staging-bot-fleet --app-dir $(projdir)/internal/app/appengine

test:
	go test ./...

gen:
	go generate ./...
