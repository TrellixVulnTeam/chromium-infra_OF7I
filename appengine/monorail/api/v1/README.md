# Monorail v1.0 pRPC API

This directory holds all the source for the Monorail pRPC API. This API is
implemented using `.proto` files to describe a `gRPC` interface (services,
methods, and request/response messages). It then uses a shim which
converts the
[`gRPC` server](http://www.grpc.io/docs/tutorials/basic/python.html)
(which doesn't work on AppEngine, due to lack of support for HTTP/2) into a
[`pRPC` server](https://godoc.org/github.com/luci/luci-go/grpc/prpc) which
supports communication over HTTP/1.1, as well as text and JSON IO.

- Resource name formats for each message are found in the message's resource annotation `pattern` field.
- This v1.0 pRPC API is a resource-oriented API and aims to closely follow the principles at aip.dev.


## API Documentation

All resources, methods, request parameters, and responses are documented in
[./api_proto](./api_proto).

Resource name formats for each message are found in the message's resource annotation `pattern` field.

## Development

### Regenerating Python from Protocol Buffers

In order to regenerate the python server and client stubs from the `.proto`
files, run this command:

```bash
$ make prpc_proto_v1
```
