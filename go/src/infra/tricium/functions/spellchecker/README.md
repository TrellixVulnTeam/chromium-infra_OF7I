# Spellchecker

Tricium analyzer to check spelling on comments and text files.

Consumes Tricium FILES and produces Tricium RESULTS comments.

## Development and Testing

Local testing:

```
$ go build
$ ./spellchecker --input=test --output=out
```

## Deployment

Deploy a new version of the analyzer using CIPD:

```
$ make
$ cipd create -pkg-def=cipd.yaml
<outputs the VERSION>
$ cipd set-ref infra/tricium/function/spellchecker -ref live -version VERSION
```

## Adding Terms to Dictionary

The `dictionary.txt` file comes from the [`codespell`] repo. If you would like
to add new terms to the dictionary you should submit a PR to the [`codespell`]
repo since the Tricium Spellchecker `dictionary.txt` is periodically synced with
that.

[`codespell`]: https://github.com/codespell-project/codespell
