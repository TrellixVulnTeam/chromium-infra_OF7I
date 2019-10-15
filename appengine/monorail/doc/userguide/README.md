# Monorail Issue Tracker User Guide


## What is Monorail?

Monorail is the Issue Tracker used by the Chromium project and other related
projects. It is hosted at [bugs.chromium.org](https://bugs.chromium.org).


## Why we use Monorail

The projects that use Monorail have carefully considered other issue
tracking tools and selected Monorail because of several key features:

* Monorail is extremely flexible, allowing for a range of development
  processes to be used in different projects or within the same project,
  and for process details to be gracefully phased in and phased out
  over time.  For example, labels and custom fields are treated very
  much like built-in fields.
* Monorail is inclusive in that it allows average users to view details
  of how a project's development process is configured so that contributors
  can understand how their contributions fit in.  And, Monorail's UI
  emphasizes usability and accessibility.
* Monorail has a long track record of hosting a mix of public-facing and
  private issues.  This has required per-issue access controls and user
  privacy features.
* Monorail helps users focus on individual issues and also work with sets
  of issues through powerful issue list, grid, and graph views.
* Monorail is built and maintained by the Chrome team, allowing for
  customization to our processes.  For example, Feature Launch Tracking.


## Links to chapter pages

This user guide is organized into the following chapters:

* [Quick start](/quick-start.md)
* [Concepts](/concepts.md)
* [Working with individual issues](/working-with-issues.md)
* [Issue lists, grids, and charts](/list-views.md)
* [Power user features](/power-users.md)
* [Email notifications and replies](/email.md)
<!-- Feature launch tracking and approvals -->
<!-- Other project pages for users -->
<!-- User profiles and hotlists -->
<!-- Project owner guide -->
<!-- Site admin guide -->


## How to ask for help and report problems with Monorail itself

If you wish to file a bug against Monorail itself, please do so in our
[self-hosting tracker](https://bugs.chromium.org/p/monorail/issues/entry).
We also discuss development of Monorail at `infra-dev@chromium.org`.


## Brief note on Monorail history

The design of Monorail was insipred by our experience with Bugzilla and
other issue trackers that tended toward hard-coding a development
process into the tool.  This work was previously part of Google's
Project Hosting service on code.google.com from 2006 until 2016.
Several Chromium-related projects were heavy users of the issue
tracker part of code.google.com, and they opted to continue
development work on it.  Monorail launched as an open source project
in 2016.  Bugs.chromium.org currently hosts over 25 related projects,
with over one million issues in the /p/chromium project alone.
