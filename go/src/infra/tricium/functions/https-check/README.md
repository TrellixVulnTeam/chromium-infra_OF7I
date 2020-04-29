# HTTPS-check for URLs

Tricium analyzer checking that all links (except `go/` and `g/` links)
in README files use `https`.

Consumes Tricium FILES and produces Tricium RESULTS comments.

## Development and Testing

Local testing:

```
$ go build
$ ./https-check --input=test --output=output
```

## Deployment

Deploy a new version of the analyzer using CIPD:

```
$ go build
$ cipd create -pkg-def=cipd.yaml
<outputs the VERSION>
$ cipd set-ref infra/tricium/function/https-check -ref live -version VERSION
```
