help:
	@echo "Available commands:"
	@sed -n '/^[a-zA-Z0-9_]*:/s/:.*//p' <Makefile

format:
	gofmt -s -w .

test:
	go test

# TODO (nqmtuan): Change project to chops-rotation-proxy after resolving permission problem.
deploy:
	gae.py upload -A chops-rotation-proxy-test
