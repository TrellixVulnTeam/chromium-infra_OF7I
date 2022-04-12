# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Generates Python proto bindings using *.proto files located in luci-go repo.
# Assumes you are running this in infra gclient checkout.

THIS_DIR=$(shell pwd)
INFRA=$(THIS_DIR)/..
LUCI_GO=$(INFRA)/go/src/go.chromium.org/luci
TOOLS=$(INFRA)/luci/appengine/components/tools
PROTOC=$(INFRA)/cipd/bin/protoc

all: regenerate

$(PROTOC):
	cd $(INFRA) && gclient runhooks

$(TOOLS)/protoc-gen-prpc-python:
	echo "Run gclient sync yourself"
	exit 1

regenerate: $(PROTOC) $(TOOLS)/protoc-gen-prpc-python
	rm -rf ./go
	$(INFRA)/cipd/bin/protoc \
		--proto_path $(INFRA)/go/src \
		--proto_path $(LUCI_GO)/common/proto/googleapis \
		$(LUCI_GO)/buildbucket/proto/*.proto \
		$(LUCI_GO)/common/proto/options.proto \
		$(LUCI_GO)/common/proto/structmask/structmask.proto \
		$(LUCI_GO)/common/proto/gerrit/gerrit.proto \
		$(LUCI_GO)/cv/api/bigquery/v1/*.proto \
		$(LUCI_GO)/cv/api/migration/*.proto \
		$(LUCI_GO)/cv/api/recipe/v1/*.proto \
		$(LUCI_GO)/cv/api/config/v2/*.proto \
		$(LUCI_GO)/resultdb/proto/v1/*.proto \
		$(LUCI_GO)/scheduler/appengine/messages/config.proto  \
		$(LUCI_GO)/server/auth/delegation/messages/*.proto  \
		$(LUCI_GO)/swarming/proto/api/*.proto \
		$(LUCI_GO)/tokenserver/api/*.proto  \
		$(LUCI_GO)/tokenserver/api/minter/v1/*.proto \
		--plugin=$(TOOLS)/protoc-gen-prpc-python \
		--python_out ./ \
		--prpc-python_out .
	python3 _fill_init_py.py go/
	@git -C $(LUCI_GO) rev-parse HEAD > luci-go.revision
	@git -C $(LUCI_GO) diff --quiet || echo 'DIRTY: please clean luci-go checkout' >> luci-go.revision
	@git -C $(LUCI_GO) merge-base --is-ancestor HEAD origin/main || echo "NOT-LANDED: please checkout revision from luci-go's origin/main" >> luci-go.revision
	git commit ./ -m "Update Python proto bindings."
