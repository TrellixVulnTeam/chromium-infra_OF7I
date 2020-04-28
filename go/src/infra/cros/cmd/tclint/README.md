`tclint` is a tool to lint test configuration payloads for Chrome OS integration
tests.

See [go/cros-f20-api](https://go/cros-f20-api) for a description of the payload
schema.

`tclint` should be used to provide early feedback to configuration payload
authors. e.g., `tclint` may be included in the developer and build workflows of integration tests.

Payloads that are not `tclint` clean will be rejected by the Chrome OS Test
Platform implementation of the Test Lab Environment.