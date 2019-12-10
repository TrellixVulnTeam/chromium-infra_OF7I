# Monorail Quick Start

[TOC]

## Mission: Track issues to help improve software products

Monorail’s mission is to track issues to help improve software
products.  Every piece of information stored in Monorail’s database is
contributed from a user to one of the projects that we host for the
purpose of helping developers work on that project.  Issue tracking
covers many different development processes, including fixing defects,
organizing enhancement requests, breaking down new development work
into tasks, tracking operational and support tasks, coordinating and
prioritizing work, estimating schedules, and overseeing new feature
launches.


## Guiding principles

* Flexibility: Monorail is unusually flexible, especially in its use
  of labels and custom fields.  This allows large projects to include
  several small teams, some of which care about labels and fields that
  are specific to their own processes.  Flexibility also enables
  process changes to be gracefully phased in or phased out over the
  long term.

* Security: Even open source projects need access controls so that
  developers can work to resolve security flaws before disclosing
  them.  Per-issue access controls allow developers to work closely
  with users and partners on a mix of public and restricted issues.

* Inclusiveness: Computing is an empowering and equalizing force in
  society.  Monorail’s inclusive functionality and user interface can
  help many different stakeholders influence the future of a project,
  causing ripple effects of inclusion through our software ecosystem.


## High-level organization

A Monorail server is divided into projects, such as `/p/chromium` and
`/p/monorail`, that each have a list of project members, and that
contain a set of issues.  Each project also has a page that lists the
history of issue changes, and a set of pages that describe the software
development process configured for that project.

Each issue has metadata such as the issue summary, reporter, owner,
CC'd users, labels, and custom fields.  Each issue also has a list of
comments, which may each have some attachments and amendments to the
metadata.

Each user also has a profile page and related pages that show that user's
activity on the site, their saved queries, and their hotlists.


## How to enter an issue

Please search for relevant existing issues before entering a new issue.

1. Sign in to Monorail.  If this is your first time, you will see a
   privacy notice dialog box.
1. Click on the `New issue` button at the top of the page.
   Note: In the `/p/chromium` project, non-members will be redirected
   to a new issue wizard.
1. Choose an issue template based on the kind of issue that you want
   to create.
1. Fill in the issue summary and details.
1. Project members can also set initial values for labels and fields.
1. Attach files that will help project members understand and
   resolve the issue.

Most issue types are public or could become public, so don't include
personal or confidential information.  Be mindful of the contents of
attachments, and crop and redact screenshots to avoid sharing
unintended details.  Never include passwords.

When you report an issue, you star the issue by default.  Starring
causes you to get email notifications of comments on that issue.

It is also possible to enter issues by clicking on a "deep link" to
our issue entry page.  Such links are sometimes used in documentation
pages that tell users when to file an issue.


## How to search for issues

1. Click on the search box at the top of the page.
1. Type in some search terms.  These can be structured or full text terms.
   The autocomplete menu lists possible structured terms.
1. The search will cover open issues by default.  You can search all
   issues or another scope by selecting a value from the search scope
   menu.
1. Press Enter, or click the search icon.

A menu at the end of the search box input field offers links to the
advanced search page and the search tips page.

You can jump directly to any issue by searching for the issue’s ID
number.


## How to comment on an issue

1. Sign in to Monorail.  If this is your first time, you will see a
   privacy notice dialog box.
1. The comment box is located at the bottom of each issue detail page,
   below existing comments.
1. Please keep comments respectful and constructive with the goal of
   resolving the issue in mind.  A message with a code of conduct link
   is shown to new users.
1. If you want to be notified of future updates on this issue, click
   the star icon.

If you need to delete a comment that you posted, use the "..." menu
for that comment.  Attachments can also be marked as deleted.


## How to edit an issue

1. Sign in to Monorail as a project member.  Only project members may
   edit issues.
1. The editing fields are located with the comment form at the bottom
   of the page.
1. Please consider posting a comment that briefly explains the reason
   for the edit.  For example, "Lowering the priority of this defect
   because there is a clear work-around."

It is also possible for project members to bulk edit multiple issues
at one time, and to update an issue by replying to some issue update
notifications.


## How to link to an issue

If you need a URL that you can bookmark or paste into another document:

* You can copy and share the URL of the issue as it is shown in the
  browser location bar.

* For a cleaner link, open the browser context menu on the link icon
  located next to the issue summary, then choose "Copy Link Address".

If you are writing text in an issue comment, you can make a textual
reference to another issue by typing the word "issue" or "issues".  For
example:

* issue 1234
* issues 1234, 2345, and 3456
* issue monorail:1234


## How to be notified of changes to an issue

There are several ways to get notifications:

* Click the star icon at the top of the issue to express your interest
  in seeing the issue resolved and to be notified of future updates to
  the issue.  Or, click the star icon for that issue in the issue list.

* The issue owner and any CC’d addresses are notified of changes.

* You can subscribe to a saved query.  Start by clicking the "Saved
  queries" item in the account menu.


## How to associate a CL with an issue

1. When you create a code change list, include a "BUG:" or "Fixed:"
   line at the bottom of the change description.
1. When you upload the CL for review, consider posting a comment to
   the issue with the URL of the pending CL.
1. Most projects have set up the Bugdroid tool to post a comment to
   the issue when the CL lands.


## How to ask for help and report problems with Monorail itself

<!-- This is purposely written in a couple different places to make it
     easier for users to find. -->

If you wish to file a bug against Monorail itself, please do so in our
[self-hosting tracker](https://bugs.chromium.org/p/monorail/issues/entry).
We also discuss development of Monorail at `infra-dev@chromium.org`.

You can report spam issues via the "..." menu near the issue summary.
You can report spam comments via the "..." menu on that comment.  Any
project owner can ban a spammer from the site.
