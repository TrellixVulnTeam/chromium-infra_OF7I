# Assumes you are running this in infra gclient checkout.

GCLIENT_ROOT=$(shell gclient root)
INFRA=$(GCLIENT_ROOT)/infra
#ENGINE=$(INFRA)/chromeperf/src/chromeperf/engine

all: regenerate

$(INFRA)/cipd/protoc:
	cd $(INFRA) && gclient runhooks

regenerate: $(INFRA)/cipd/protoc ./proto/*.proto
	$(INFRA)/cipd/protoc \
		--proto_path ./proto \
		./proto/*.proto \
		--python_out ./pb
