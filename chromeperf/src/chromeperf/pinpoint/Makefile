# Assumes you are running this in infra gclient checkout.

GCLIENT_ROOT=$(shell gclient root)
INFRA=$(GCLIENT_ROOT)/infra

all: regenerate

$(INFRA)/cipd/protoc:
	cd $(INFRA) && gclient runhooks

regenerate: $(INFRA)/cipd/protoc ./proto/*.proto
	$(INFRA)/cipd/protoc \
		--proto_path ./proto \
		./proto/*.proto \
		--python_out ./pb
