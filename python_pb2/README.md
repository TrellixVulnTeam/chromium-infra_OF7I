# Compiled protobufs for Python

This dir contains generated protos for Python for infra & infra_internal repo.

## Regenerate

```
make
```

**Assumption**: you are doing this in *infra* gclient checkout (get it via
`fetch infra`).


## Use

Ensure this dir is in your PYTHONPATH or `sys.path` or via
`infra.init_python_pb2` module. For example,

```python
from infra import init_python_pb2  # pylint: disable=unused-import
from go.chromium.org.luci.buildbucket.proto import build_pb2
build = build_pb2.Build(...)
```
