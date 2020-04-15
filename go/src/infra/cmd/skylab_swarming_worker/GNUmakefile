skylab_swarming_worker: ssw

.PHONY: ssw
ssw: check-environment
	go build -o skylab_swarming_worker -- .

.PHONY: clean
clean: check-environment
	$(RM) ./skylab_swarming_worker

.PHONY: test
test: check-environment
	go test ./...

.PHONY: generate
generate:
	go generate ./...

.PHONY: check-environment
check-environment:
	@if ! (env | grep -q '^GOPATH.*infra/go/[.]vendor'); then \
		echo 'not in infra gopath workspace' 1>&2;      \
		exit 1;                                         \
	fi
