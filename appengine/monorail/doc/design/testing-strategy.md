# Monorail Testing Strategy

## Problem

Monorail (bugs.chromium.org) is a complex issue tracking tool with a
large number of features that needs to work very reliably because it
is used by everyone working on Chrome, which is a critical Google
product.  At the same time, testing needs to be done at low cost
because the Monorail team is also tasked with developing new
functionality.

## Strategy

Basically, the end goal is a test automation pyramid with the base
being unit tests with the highest test coverage, a middle layer where
we test the API that the the UI uses, and the top layer is automated
system tests for a few things to avoid surprises.

![Testing pyramid](https://2.bp.blogspot.com/-YTzv_O4TnkA/VTgexlumP1I/AAAAAAAAAJ8/57-rnwyvP6g/s1600/image02.png)

The reason for that is that unit tests give the best return on
investment and the best support for ensuring that individual changes
work correctly as those changes are being made.  End-to-end testing is more
like a necessary evil: it is not very cost-effective because these
tests are hard to write, easily broken by unrelated changes, and often
have race conditions that make them flakey if we are not very careful.
The API tests at the middle layer are a kind of integration testing
that proves that larger parts of the code work together in a
production environment, but they should still be easy to maintain.

Past experience on code.google.com supports that strategy. Automated
system tests were done through the UI and were notoriously slow and
flakey. IIRC, we ran them 4 times and if any one run passed, it was
considered a pass, and it was still flakey.  They frequently failed
mysteriously.  They were so flakey that it was hard to know when a new
real error had been introduced rather than flakiness getting worse. We
repeatedly rewrote tests to eliminate flakiness and got them to pass,
but that is a slow process because you need to run the tests many
times to be sure that it is really passing, and there was always the
doubt that a test could be falsely passing.  Many manual tests were
never automated because of a backlog of problems with the existing
automated tests.


See also:
[Google Test Blog posting about end-to-end tests](https://testing.googleblog.com/2015/04/just-say-no-to-more-end-to-end-tests.html).
And, [Google internal unit test how-to](go/unit-tests).


## Test coverage goals

| Type        | Lang/Tech                | Coverage | Flakiness |
|-------------|--------------------------|----------|-----------|
| Unit        | Python pRPC API code     | 100%     | None      |
| Unit        | Other python code        | >85%     | None      |
| Unit        | Javascript functions     | >90%     | None      |
| Unit        | LitElement JS            | >90%     | None      |
| Integration | Probers                  | 10%?     | None      |
| Integration | pRPC client in python?   | 25%?     | None      |
| UI          | Web testing tool?        | 10%?     | As little as possible |
| UI          | English in a spreadsheet | 100%     | N/A       |


## Plan of action

Building all the needed tests are a lot of work, and we have limited
resources and a lot of other demands.  So, we need to choose wisely.
Also, we need to keep delivering quality releases at each step, so we
need to work incrementally.  Since we won't "finish", we need a
mindset of constant test improvement, as tracked in coverage and
flakiness metrics.

Steps:

1.  Strictly maintain unit test coverage for pRPC, work_env, and other
    key python files at 100%.

1.  Improve python unit test code style and update library usage,
    e.g., mox to mock.

1.  Improve unit test coverage for other python code to be > 85%.

1.  Maintain JS unit tests as part of UI refresh.  Run with `karma`.

1.  Design and implement probers for a few key features.

1.  Design and implement API unit tests and system tests.

1.  Research options for web UI testing. Select one.

1.  Implement automated UI tests for a few key features.

1.  Maintain go/monorail-system-test spreadsheet as we add or modify
    functionality.


## Related topics

Accessibility testing: So far we have just used the audit tools in
Chrome. We could find additional audit tools and/or request
consultation with some of our users who are accessibility experts.  We
have, and will continue to, give high priority to accessibility
problems that are reported to us.

API performance testing: Some API calls are part of our monitoring.
However, we currently do not actively manage small changes in latency.
We should look at performance changes for each release and over longer
time-spans.

UI performance testing: Again, we have monitoring, but we have not
been looking critically at small changes in UI performance across
releases.

Security testing:  We are enrolled in a fuzzing service.
