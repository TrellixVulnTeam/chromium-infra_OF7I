.PHONY: stable_version2
stable_version2:
	go build -o stable_version2 -- .

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
	$(RM) ./stable_version2
	$(RM) -r ./.cipd
	$(RM) -r ./.versions

# Run the update with omaha task locally.
# Doing this will consult omaha the same way running the task in production will,
# but it will write the output.json to stdout rather than committing it.
# We clear the PATH pre-emptively do we don't pick up user's real version of any
# tools that stable_version2 calls or might call in the future.
# TODO(gregorynisbet): clear the environment completely except for the variables
# needed to authenticate.
.PHONY: check-update-with-omaha
check-update-with-omaha: stable_version2
	dir="$$(mktemp -d)" && \
	(PATH= && ./stable_version2 update-with-omaha -dryrun -output_json="$${dir}/output.json") && \
	status="$$?"; \
	cat < "$${dir}/output.json"; \
	exit "$$status"
