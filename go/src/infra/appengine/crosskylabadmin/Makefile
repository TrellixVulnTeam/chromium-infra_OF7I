# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
projdir := $(patsubst %/,%,$(dir $(mkfile_path)))

dev: env-check
	gae.py devserver -A dev --app-dir $(projdir)/app/appengine -- --host 0.0.0.0 --port 8082 --admin_port 7999 --log_level debug

up-prod: env-check
	gae.py upload -A chromeos-skylab-bot-fleet --app-dir $(projdir)/app/appengine

switch-prod: env-check
	gae.py switch -A chromeos-skylab-bot-fleet --app-dir $(projdir)/app/appengine

versions-cleanup-prod: env-check
	gae.py cleanup -A chromeos-skylab-bot-fleet --app-dir $(projdir)/app/appengine

up-staging: env-check
	gae.py upload -A skylab-staging-bot-fleet --app-dir $(projdir)/app/appengine

switch-staging: env-check
	gae.py switch -A skylab-staging-bot-fleet --app-dir $(projdir)/app/appengine

versions-cleanup-staging: env-check
	gae.py cleanup -A skylab-staging-bot-fleet --app-dir $(projdir)/app/appengine

test: env-check
	go test ./...

gen: env-check
	go generate ./...

# the presence or absence of the 'mockgen' command on the $PATH seems to be
# a reliable way to distinguish an environment inside the chrome_infra GOPATH
# workspace from one that's outside of it.
env-check:
	@which mockgen 1>/dev/null 2>/dev/null || ( echo 'bad environment. must eval output of chrome_infra/infra/go/env.py' && exit 20 )
