Feature: IsolateFinder
    An evaluator which supports finding isolates and starting BuildBucket
    builds.

    Background:
        Given a commit chromium@f00dcafe and change 123456
        And a Pinpoint job
        And a isolate-finding task graph for commit chromium@f00dcafe

    Scenario: No cached isolates scheduling builds
        Given attempts to schedule builds succeed
        When we evaluate the task graph
        Then we must have scheduled 1 build
        And the task payload has a buildbucket build
        And the task is ongoing

    Scenario: There is a cached isolate for the commit
        Given a cached isolate for commit chromium@f00dcafe
        When we evaluate the task graph
        Then we must have scheduled 0 builds
        And the task payload has no buildbucket build
        And the task has isolate details
        And the task is completed