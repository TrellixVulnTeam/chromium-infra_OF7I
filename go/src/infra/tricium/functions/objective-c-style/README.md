# Objective-C Style Checker

Performs the following checks:

1. Tricium analyzer checking inappropriate use of get prefix in Objective-C
   methods.

   Flags Objective-C methods which have "get" prefix and return type other than
   void or don't have any arguments, which could be used for out arguments.

   See this guide for more details:
   https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/CodingGuidelines/Articles/NamingMethods.html#:~:text=The%20use%20of%20%22get%22%20is%20unnecessary,%20unless%20one%20or%20more%20values%20are%20returned%20indirectly.

Consumes Tricium FILES and produces Tricium RESULTS comments.

## Development and Testing

Local testing:

```
$ go build
$ ./objective-c-style --input=test --output=output
```

## Deployment

Deploy a new version of the analyzer using CIPD:

```
$ go build
$ cipd create -pkg-def=cipd.yaml
<outputs the VERSION>
$ cipd set-ref infra/tricium/function/objective-c-style -ref live -version VERSION
```
