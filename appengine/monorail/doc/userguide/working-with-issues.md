# Working with Issues

[TOC]

## Why do we  track issues?

The goal of tracking an issue is to resolve it.  Issues are a way for
the development team to organize information about work that could be
done to improve their software product.  There are many types of
issues including defects, enhancement requests, operational alert
tickets, and several others.  There are typically far more things that
the team could be working on than they actually have time to work on,
so issues are triaged, prioritized, labeled, assigned, and discussed.
They can be open for a long time before being resolved.  After they
are resolved, many issues contain important rationale that can be used
when investigating regressions.


## What’s in an issue?  And, why is that information needed?

Issues in Monorail contain a summary line, description, comments, a
reporter, an owner, CC’d users, labels, components, custom fields, and
references to related issues.  Users can express interest in an issue
by starring it, and each issue shows a star count.  Each piece of that
information serves one or more of the following purposes:


* Description of the situation.  Issues should capture details about
  error messages, steps to reproduce, and how the software product
  falls short of user expectations.  Attachments may help capture this
  information in the form of screenshots or logs.

* Routing and prioritization.  Before an issue can be resolved, it
  needs to be seen by the right team and assigned to an owner who can
  make the needed changes.  CC’s, components, and labels are often
  used to make issues show up in the triage queries used by different
  teams.

* Investigation.  Not every issue can be solved immediately.  Some
  issues need further investigation to find the root cause of the
  problem or to weigh different solution approaches.  This information
  is usually captured in comments, attachments, and links.

* Traceability.  Comments with references to CLs show how much work
  has been done so far.  After code changes are committed, they must
  often be verified by a QA team or the issue reporter.  Later, the
  ability to trace between code commits and issues provides important
  rationale for the state of the code, which can help prevent future
  regressions or fix them.

* Advocacy and community.  Users should have a voice in the project.
  Issue stars help prioritize issues and keep users in the loop as the
  discussion continues.  Users can post comments to explain the impact
  that an issue is having on them, to weigh in on solution
  alternatives, and to thank developers.


## Who can view an issue?

Basically, public issues may be viewed by anyone, whereas restricted
issues can only be viewed by people involved in the issue and project
members who were granted access that type of issue.

<!-- TODO(jrobbins): Maybe move this to a separate permissons.md
     reference page. -->

Here are the details:

1. Before a user may access an issue, they must first be able to
   access the project.  Most projects that we host are public, but
   some are members-only.

1. The issue participants (including reporter, owner, and any CC’d
   users) can always view that issue.  Users named in certain custom
   fields also gain access.  Project owners can view all issues in
   that project, and site administrators can view any issue in any
   project.

1. For other project members, an issue that has a label
   `Restrict-View-X` may only be viewed if the user has been granted
   permission `X` in that project.  If there are multiple
   `Restrict-View-*` labels, the user needs all permissions specified
   in those labels.

1. An issue that is in a public project and that has no restriction
   labels may be viewed by anyone.

For example, in a public project there could be an issue labeled
`Restrict-View-SecurityTeam`.  The only users who may view that issue
would be the reporter, owner, CC’d users, users named in custom
fields, project owners, site admins, and project members who were
granted the `SecurityTeam` permission.  It would not be viewable by
anonymous visitors, non-members who are not participants, or even
project members who were not granted the `SecurityTeam` permission.


## Who can edit an issue?

Basically, only project committers and project owners may edit issues.
That set of users is narrowed down if the issue has restriction
labels.

<!-- TODO(jrobbins): Maybe move this to a separate permissons.md
     reference page. -->

Here are the details:

1. A user cannot edit an issue if they are not permitted to view it.

1. The issue owner can always edit that issue.  Also, users named in
   certain custom fields may be able to edit the issue.  Project
   owners can edit any issue in that project, and site administrators
   can edit any issue in any project.

1. For other project members, editing an issue requires the
   `EditIssue` permission.  That permission is part of the project
   committer role, and it can also be granted to a project
   contributor.

1. If the issue has a restriction label `Restrict-EditIssue-X`, then
   only project members who were granted permission `X` in that
   project may edit that issue.  If there are multiple
   `Restrict-EditIssue-*` labels, the user needs all permissions
   specified in those labels.


## Who owns issue content?

Because the purpose of tracking issues is to help project members
improve the software that they are developing, all issue content
belongs to the project.  When a user reports an issue or posts a
comment, they are contributing that information to the project for the
benefit of the project.


## How to enter an issue

<!-- Note that this is also in quick-start.md. -->

1. Sign in to Monorail.  If this is your first time, you will see a
   privacy notice dialog box.

1. Click on the `New issue` button at the top of the page.

1. In the /p/chromium project, non-members will be directed to the
   "new issue wizard".

1. Choose an issue template based on the kind of issue that you want to create.

1. Fill in the issue summary and details.

1. Project members can also set initial values for labels and fields.

1. Optionally, attach files that will help project members understand
   and resolve the issue.

1. When you report an issue, you star the issue by default.  Starring
   causes you to get email notifications of comments on that issue.


## How to link to an issue or a specific comment in an issue

<!-- Note that this is also in quick-start.md. -->

If you need a URL that you can bookmark or paste into another document:

* You can copy and share the URL of the issue as it is shown in the
  browser location bar.

* For a cleaner link, click on the link icon located to the right of
  the issue summary.

If you are writing text in an issue comment, you can make a textual
reference to another issue by typing word "issue" or "issues".  For
example:

* issue 1234
* issues 1234, 2345, and 3456
* issue monorail:1234
* Googlers can reference an internal issue by using: b/1234


## How to view and download issue attachments

* Attached images and videos show a thumbnail or preview image.  You
  can click on that to see the full-sized image or play the video.

* Click the `View` link to view the attachment in a new browser tab.
  Not all types of attachments can be viewed this way.

* For many types of text files, viewing the attachment opens a new
  page that shows the file with syntax highlighting.

* Click the `Download` link to download the attachment to your
  computer.  Usually the file name is the same as the one used when
  the file was uploaded, however in some cases we use a filename that
  we know is safe.


## How to link to a specific line of attached text file

1. Open the attachment by clicking the `View` link.

1. In the text file attachment viewer page, the line numbers are
   hyperlinks.  Click one to add an anchor to your current browser
   location, or use a pop-up menu to copy the link address.


## How to comment on an issue

<!-- Note that this is also in quick-start.md. -->

1. Sign in to Monorail.  If this is your first time, you will see a
   privacy notice dialog box.

1. The comment box is located at the bottom of each issue detail page,
   below existing comments.

1. Please keep comments respectful and constructive with the goal of
   resolving the issue in mind.  A message with a code of conduct link
   is shown to new users.

1. If you want to be notified of future updates on this issue, click
   the star icon.


## How to delete a comment or attachment

Users may delete comments that they posted.  Also, project owners and
site administrators may delete any comment.

1. Open the `...` menu that is near the comment.
1. Select `Delete comment` to mark the comment as deleted.


## How to report spam

Users may report spam issues and comments.  These spam reports are
used as inputs to our spam detection model.  When a project member
reports spam, the issue or comment is immediately classified as spam.

1. Sign in to Monorail.
1. Open the `...` menu near the issue summary or comment.
1. Select `Flag comment` item or the `Flag issue as spam` item.


## How to edit an issue

<!-- Note that this is also in quick-start.md. -->

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


## How to close an issue as a duplicate of another issue

1. Sign in as project committer and view the issue.

1. Set the issue status to `Duplicate`.

1. A `Merged into:` input field will appear, type the other issue ID
   into that field.
  * If the other issue is in a different project, use `projectname:issue_id`.
  * For Googlers, if the issue is in our internal issue tracking tool,
    use `b/issue_id`.

1. Press `Save changes`.


## How to be notified of changes to an issue

Anyone can star an issue by clicking the star icon near the issue ID
on the issue detail page, issue list page, or hotlist page.  Starring
an issue will subscribe you to email notifications for changes to that
issue.  The issue reporter stars the issue by default, but they can
unstar it if they are no longer interested.

Another way to get notifications is to create a saved query.  Sign in
and click your email address in the far upper right of the page to
access your account menu, then choose `Saved queries`.  You can name
and define an issue query for issues that you are interested in.  When
any issue that matches that query is updated, you will get an email
notification.

Email notifications of changes are also sent to the issue owner and
any CC’d users.  Users named in certain custom fields will also be
notified.  When the issue owner is changed, the old issue owner gets
one final notification of that change, even though they are no longer
an issue participant.

Project owners and component admins can also set up filter rules and
auto-cc rules to automatically add users to the CC field of an issue.
Filter rules can also cause notifications to be sent to email
addresses that do not represent users, for example, mailing lists.


## How to associate a CL with an issue

When writing your CL description, add a `BUG=` line or a `Fixes:` line
to the end of the commit log message.  These lines can reference a
`/p/chromium` issue by issue ID number, or an issue in any project by
using the "project_name:issue_id" syntax.

After you upload a CL for review, you can copy the URL of the code
review and then paste it into an issue comment.

When the CL passes review and is committed, the bugdroid utility will
automatically post a comment to the issue referenced in the `BUG=` or
`Fixes` line.


## How to move or copy an issue between projects

<!-- I am not mentioning DeleteIssue because I think it should not be
required.  See monorail:6634 -->

You must have permission to edit issues in the destination project.
Only non-restricted issues can be moved: if an issue is restricted,
consider creating a new issue in the target project that blocks the
original issue.

1. Open the `...` menu near the issue summary and choose `Move issue`
   or “Copy issue”.

1. Select the name of the target project in the dialog box that opens.

1. Press the `Move issue` or `Copy issue` button.


## How to delete an issue

In most cases, you should close the issue with status `Invalid` or
`WontFix` rather than deleting it.  Spam issues should be marked as
spam so that they help build our spam model.  Only project owners and
site administrators can mark issues as deleted.

1. Sign in as a project owner and view the issue.
1. Open the `...` menu near the issue summary and choose `Delete issue`.
1. Confirm that you really want to mark the issue as deleted.
