# Assumes you are running this in infra gclient checkout.

GCLIENT_ROOT=$(shell gclient root)
INFRA=$(GCLIENT_ROOT)/infra

all: regenerate

$(INFRA)/cipd/protoc:
	cd $(INFRA) && gclient runhooks

regenerate: $(INFRA)/cipd/protoc chromeperf/engine/*.proto chromeperf/pinpoint/*.proto
	$(INFRA)/cipd/protoc \
		chromeperf/engine/*.proto chromeperf/pinpoint/*.proto \
		--python_out .
