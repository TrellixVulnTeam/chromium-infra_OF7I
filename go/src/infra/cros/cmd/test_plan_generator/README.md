# Test plan generator

The test plan generator determines which tests need to run for a given set of
Chrome OS builds.

This is generally intended to be executed by a LUCI recipe. A service account
credentials file is needed in order to fetch CL data from Gerrit.

```shell
go run cmd/test_plan_generator/main.go gen-test-plan \
    --input_json=/path/to/input.json \
    --output_json=/path/for/output.json \
    --service-account-json=/path/to/service-account-json-file.json
```

See the sample directory for data that can be used for local runs.
