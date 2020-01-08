.PHONY: skylab
skylab: check-environment
	go build -o skylab -- .

.PHONY: gen
gen: check-environment
	go generate ./...

.PHONY: test
test: check-environment
	go test ./...

.PHONY: check
check: test

.PHONY: clean
clean: check-environment
	$(RM) ./skylab

.PHONY: check-environment
check-environment:
	@if ! (env | grep -q '^GOPATH.*infra/go/[.]vendor'); then \
		echo 'not in infra gopath workspace' 1>&2;      \
		exit 1;                                         \
	fi
