# Regression Test Selection (RTS)

Regression Test Selection (RTS) is a technique to intellegently select tests to
run, without spending too much resources on testing, but still detecting bad
code changes. Conceptually, a selection strategy for CQ accepts changed files as
input and returns tests to run as output.

As of February 2021, most of the documention can be found at
https://bit.ly/chromium-rts. This document is focused on particular tasks
that a developer might need to perform in this code base.

note: TODO(nodir): move relevant parts from bit.ly/chromium-rts to this
README.md.

## Improving Chromium's selection strategy

Chromium's selection strategy can be found in ./cmd/rts-chromium/strategy.go.
It is primarily based on ./filegraph with some Chromium-specific rules.

Changes to the selection strategy should be accompanied with the
safety/efficiency scores before and after the change on the same dataset,
in order to ensure that the change is in fact an improvement.

1. Get yourself permissions to `chrome-rts` Cloud project by being a member of
  chrome-troopers@google.com group or by contacting nodir@google.com.
1. Ensure you have the Chromium checkout: just src.git is enough.
   This doc assumes you have it at `~/chromium/src`.
1. Fetch a sample of rejections, e.g. for the past month of linux-rel:
   ```bash
   go run ./cmd/rts-chromium fetch-rejections \
     -builder linux-rel \
     -from 2021-02-01 -to 2021-03-01 \
     -out linux-rel.rej
   ```
1. Fetch a sample of test durations, e.g. for for the past week of linux-rel.
   ```bash
   go run ./cmd/rts-chromium fetch-durations \
     -builder linux-rel \
     -from 2021-02-01 -to 2021-02-08 \
     -out linux-rel.dur
   ```
1. Create the model and note the safety/efficiency scores before your changes:
   ```bash
   go run ./cmd/rts-chromium create-model \
     -checkout ~/chromium/src
     -rejections linux-rel.rej \
     -durations linux-rel.dur \
     -model-dir ./model \
   ```
1. Change the selection strategy code.
1. Create the model again on the same dataset and see if the scores improved.
   Iterate until they improve.
1. Send the CL out and include the before/after scores in the CL description.

Model creation can take 30min on a powerful machine.
If you are developing on your laptop, it is recommended to scp the binary to
a more powerful machine and run the model creation there.
