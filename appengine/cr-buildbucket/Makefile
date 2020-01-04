# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by the Apache v2.0 license that can be
# found in the LICENSE file.

update-protos:
	cd ../../python_pb2 && $(MAKE)

test:
	tools/run_tests.sh

deploy:
	tools/gae upload -x

upload:
	tools/gae upload

rollback:
	appcfg.py -V `tools/gae version` rollback .

format:
	git cl format --python --full

format-all:
	find . -iname \*.py -type f | xargs yapf -i
