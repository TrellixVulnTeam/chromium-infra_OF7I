In infra/go run the following to activate the go environment:
```
eval `./env.py`
```

To run backend tests while in src/infra/appengine/dashboard/backend:
```
go test
```

The following commands should be done while in the src/infra/appengine/dashboard/frontend directory:

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
go run ./
```