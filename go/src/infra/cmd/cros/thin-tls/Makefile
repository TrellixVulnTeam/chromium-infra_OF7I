.PHONY: all
all: generate test thin-tls client

.PHONY: generate
generate:
	go generate ./...

.PHONY: test
test:
	go test ./...

.PHONY: thin-tls
thin-tls:
	go build .

.PHONY: client
client:
	go build -o $@ ./thin-tls-client
