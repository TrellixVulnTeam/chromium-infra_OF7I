help:
	@echo "Available commands:"
	@sed -n '/^[a-zA-Z0-9_]*:/s/:.*//p' <Makefile

format:
	gofmt -s -w .

# TODO (nqmtuan): Change project to chops-rotation-proxy after resolving permission problem.
deploy:
	gcloud app deploy --no-promote --no-stop-previous-version app.yaml --project chops-rotation-proxy-test