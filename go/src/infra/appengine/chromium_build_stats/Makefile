# Copyright 2018 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

build:
	go build ./...

generate: ninjaproto/ninjalog.proto
	go generate ./...

update-staging:
	go run ./cmd/bqsetup/bqsetup.go -project chromium-build-stats-staging -table bots
	go run ./cmd/bqsetup/bqsetup.go -project chromium-build-stats-staging -table users

update-prod:
	go run ./cmd/bqsetup/bqsetup.go -project chromium-build-stats -table bots
	go run ./cmd/bqsetup/bqsetup.go -project chromium-build-stats -table users

deploy-staging:
	gae.py upload -x -f -A chromium-build-stats-staging --app-dir app

deploy-prod:
	gae.py upload -x -f -A chromium-build-stats --app-dir app

test:
	go test ./...
