.PHONY: stable_version2
stable_version2: check-environment
	go build -o stable_version2 -- .

.PHONY: gen
gen: check-environment
	go generate ./...

.PHONY: test
test: check-environment
	go test ./...

.PHONY: check
check: test

.PHONY: clean
clean:
	$(RM) ./stable_version2
	$(RM) -r ./.cipd
	$(RM) -r ./.versions

.PHONY: check-environment
check-environment:
	@if ! (env | grep -q '^GOPATH.*infra/go/[.]vendor'); then \
		echo 'not in infra gopath workspace' 1>&2;      \
		exit 1;                                         \
	fi
