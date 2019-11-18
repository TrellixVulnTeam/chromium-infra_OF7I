# QuotaScheduler Profiler

This package implements a simulation for testing the performance of the quotascheduler
algorithm.

Usage:

Run `./scheduler_benchmark.sh` to run simulation. Take note of its runtime. Consider
taking a few samples, for repeatability.

Run `./visualize.sh` to display a graphical view of which functions are taking
up most of the run time.

Use these tools before and after making changes to the scheduler algorithm, to
understand whether they have a significant performance implication. Sometimes,
a seemingly simple code change can make a huge difference to performance; see e.g.
https://chromium-review.googlesource.com/c/infra/infra/+/1696394
