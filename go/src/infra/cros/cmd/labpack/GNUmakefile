.PHONY: labpack
labpack:
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
