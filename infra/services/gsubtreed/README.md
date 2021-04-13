Note: For googler only: Gsubtreed is deprecated.
Please use [copybara](https://go/copybara-chrome#copybara-gsubtreed).

# Gsubtreed

#### (git subtree daemon)

Gsubtreed was a daemon which mirrors subtrees from a large git repo to their
own smaller repos. It is typically run for medium-term intervals (e.g. 10 min),
restarting after every interval. While it's running it has a short (e.g. 5s)
poll+process cycle for the repo that it's mirroring.

It's no longer used by Chrome infrastructure, but it's latest working code
can be seen at
https://source.chromium.org/chromium/infra/infra/+/main:infra/services/gsubtreed/;drc=85d658410841ba5764031715fe94c9572f45c58b
