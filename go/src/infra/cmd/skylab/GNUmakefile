.PHONY: skylab
skylab:
	go build -o skylab -- .

.PHONY: gen
gen:
	go generate ./...

.PHONY: test
test:
	go test ./...

.PHONY: check
check: test

.PHONY: clean
clean:
	$(RM) ./skylab

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...
