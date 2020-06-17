In infra/go run the following to activate the go environment:
eval `./env.py`

The following commands should be done while in the /dashboard/frontend directory:

To install JS dependencies:
```
make deps
```

To build the JavaScript bundle:
```
make build_js
```

Use gae.py to deploy:
```
gae.py upload -A chopsdash
gae.py switch -A chopsdash
```

To run a local instance:
```
make serve
```

To run backend tests:
```
go test
```