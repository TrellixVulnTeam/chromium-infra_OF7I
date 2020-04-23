help:
	@echo "Available commands:"
	@sed -n '/^[a-zA-Z0-9_]*:/s/:.*//p' <Makefile

format:
	gofmt -s -w .

test:
	go test

deploy:
	gae.py upload -A chops-rotation-proxy-test

deploy-prod:
	gae.py upload -A chrome-ops-rotation-proxy