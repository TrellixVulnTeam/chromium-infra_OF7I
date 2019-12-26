# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Generates Python proto bindings using *.proto files located in luci-go repo.
# Assumes you are running this in infra gclient checkout.

GCLIENT_ROOT=$(shell gclient root)
INFRA=$(GCLIENT_ROOT)/infra
LUCI_GO=$(INFRA)/go/src/go.chromium.org/luci
TOOLS=$(INFRA)/luci/appengine/components/tools

all: regen

$(INFRA)/cipd/protoc:
	cd $(INFRA) && gclient runhooks

$(TOOLS)/protoc-gen-prpc-python:
	echo "Run gclient sync yourself"
	exit 1

regen: $(INFRA)/cipd/protoc $(TOOLS)/protoc-gen-prpc-python
	rm -rf ./go
	$(INFRA)/cipd/protoc \
		--proto_path $(INFRA)/go/src \
		--proto_path $(LUCI_GO)/grpc/proto \
		$(LUCI_GO)/buildbucket/proto/*.proto \
		$(LUCI_GO)/common/proto/options.proto \
		$(LUCI_GO)/cq/api/bigquery/*.proto \
		$(LUCI_GO)/cq/api/config/v2/*.proto \
		$(LUCI_GO)/server/auth/delegation/messages/*.proto  \
		$(LUCI_GO)/swarming/proto/api/*.proto \
		$(LUCI_GO)/tokenserver/api/*.proto  \
		$(LUCI_GO)/tokenserver/api/minter/v1/*.proto \
		--plugin=$(TOOLS)/protoc-gen-prpc-python \
		--python_out ./ \
		--prpc-python_out .
	python3 _fill_init_py.py go/
	git -C $(LUCI_GO) rev-parse HEAD > luci-go.revision
