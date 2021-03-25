.PHONY: labpack
labpack: check-environment
	go build -o labpack -- .

.PHONY: test
test: check-environment
	go test ./...

.PHONY: check
check: test

.PHONY: clean
clean:
	$(RM) ./labpack

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: check-environment
check-environment:
	@if ! (env | grep -q '^GOPATH.*infra/go/[.]vendor'); then \
		echo 'not in infra gopath workspace' 1>&2;      \
		exit 1;                                         \
	fi
