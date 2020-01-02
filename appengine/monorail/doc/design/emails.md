# Monorail email design

## Objective

Monorail needs a strong outbound and inbound email strategy that keeps
our users informed when issues require their attention.  We generate a
variety of outbound messages.  We accept inbound email replies from
human users who find it easier to post comments via email.  And, we
integrate with alerts monitoring via inbound email.

Our email features must scale up in several dimensions, including:

*  The number of messages generated, which is driven by the issue
   activity

*  The number of distinct types of notifications that the tool can
   generate

*  The amount of inbound replies, alerts, as well as spam and bounce
   notifications

*  The variety of access controls needed for sensitive issues and
   different types of users

## Background

Monorail is a [different-time, different-place CSCW
application](https://en.wikipedia.org/wiki/Computer-supported_cooperative_work)
in which users may need to work with each other on multiple occasions
to resolve an issue.  Furthermore, the exact set of collaborators
needed to resolve a given issue is discovered as part of the work for
that issue rather than being known from the start.  And, each issue
participant is likely to be highly multitasking across several issues
and other development tasks.  As is normal for issue tracking tools,
we send email notifications to issue participants for each issue
change.  However, because participants can get a fair number of emails
from us, they want to be able to filter those emails based on the
reason why the email was sent to them.

Email is not an inherently secure or private technology.  We trust
that email is delivered to the recipient without being read by any
servers along the way.  However, some email addresses may be
individual users and others might be mailing lists, so we "nerf"
messages in cases where Monorail has no indication that the recipient
is an individual.  Also, it is possible to forge an email reply, so we
rely on shared secrets to authenticate that an inbound message came
from a given user.  Because the email sender line is so vulnerable to
abuse, GAE does not allow GAE apps to set that header arbitrarily.
Instead, we rely on a combination of supported email senders, DNS SPF
entries, and friendly `From:` lines.

Users sometimes make mistakes when entering email addresses, and email
accounts can be shut down over time, both of these situations generate
bounce emails.  Continued generation of outbound emails that bounce is
a waste of resources and quota.

## Approach

To keep issue participants engaged, whenever an issue is created or
has a new comment, we send email notifications to the issue owner,
CC'd users, users who have starred the issue, and users who have
subscriptions that match the new state of the issue.  Monorail
generates notifications for individual issue changes, as well as bulk
edits, blocking notifications, and approval changes.  Monorail has a
special rule for "noisy" issues, which is to only generate emails when
project members post comments.  Also, when a date-type field has a
date value for a date that has arrived, we send a "follow-up" email to
remind the user to update the issue.

Monorail sends individual emails to each recipient rather than adding
them all to one long `Cc:` line.  The reason for this is so that we
can personalize the content of the message to each user based on their
preferences, the reason why the message was sent to them, and our
level of trust of that address.  Also, using individual messages
allows us to authenticate replies with a secret that is shared
individually with each user.  And, individual emails ensure that email
replies come directly back to Monorail rather than going to other
issue participants.  This reduces cases of duplicate emails and allows
for enforcement of permissions that might have changed after an
earlier notification was sent.

To keep outbound emails organized as threads in recipients' inboxes,
we set a `References:` header message ID that is used for all messages
that belong in that thread.  However, as a GAE app, Monorail has no
way to access the message ID of any actual outbound email.  Also, any
given thread participant might join the conversation late, after
missing the first email message that would have anchored the thread.
So, instead of using actual message IDs, we generate a synthetic
message ID that represents the thread as a whole and then reference
that same ID from each email.

When we send outbound emails, we include a shared secret in the
`References:` header that an email client will echo back in the reply,
much like a cookie.  When we receive an inbound email, we verify that
the `References:` header includes a value that is appropriate to the
specified user and issue.  One exception to this rule is that we allow
inbound emails from the alert system (Monarch).

Bounces are handled by flagging the user account with a bouncing
timestamp.  Monorail will not attempt to send any more emails to a
bouncing user account.  A user can clear the bouncing timestamp by
clicking a button on their user profile page.

An inbound email is first verified to check the identity of the sender
and the target issue.  After permissions are checked, the message body
is parsed.  The first few lines of the body can contain assignments to
the issue summary, status, labels, and owner.  The rest of the body is
posted to the issue as a comment.  Common email message footers and
.sig elements are detected and stripped from the comment.  The
original email message is also stored in the DB and can be viewed by
project owners.

## Detailed design: Architecture

Monorail is a GAE application with several services.  The `default`
service responds to user requests, while the `latency-insensitive`
service executes a variety of cron jobs and follow-up tasks.

Outbound email is generated in tasks that run in the
`latency-insensitive` service so as to remove that work from the time
needed to finish the user's request, and to avoid spikes in `default`
instance allocations when many outbound emails are generated at one
time.  We use automatic scaling, but turnover in `default` instances
would lower the RAM cache hit ratio.

Inbound email is currently handled in the `default` service.  However,
those requests could be routed to the `latency-insensitive` service in
the future.

## Detailed design: Domain configuration

Monorail serves HTTP requests sent to the `monorail-prod.appspot.com`
domain as well as `bugs.chromium.org` and other "branded" domains
configured in `settings.py` and the GAE console.  These custom domains
are also used in the email address part of the `From:` line of outbound
emails.  They must be listed as `monorail@DOMAIN` in the email senders
tab of the settings page for the GAE app.

The `Reply-To:` line is always set to
`PROJECTNAME@monorail-prod.appspotmail.com` and is not branded.

The DNS records for each custom domain must include a TXT record like
`v=spf1 include:_spf.google.com ?all` to tell other servers to trust
that the email sent from a certain list of SMTP servers is legitimate
for our app.

## Detailed design: Key algorithms and data structures

`User` protocol buffers include a few booleans for the user's
preferences for email notifications.

When generating a list of notification recipients, Monorail builds a
data structure called a `group_reason_list` which pairs
`addr_perm_lists` with reasons why those addresses are being notified.
Each `addr_perm_list` is a list of named tuples that have fields for a
project membership boolean, an email address, an optional `User`
protocol buffer, a constant indicating whether that user has permission
to reply, and a `UserPrefs` protocol buffer.

The `group_reason_list` is built up by code that is specific to each
reason.  A given email address might be notified for more than one
reason.  Then, that list is inverted to make a dictionary keyed by
recipient email address that lists the reasons why that address was
notified.  Entries for linked accounts are then merged.  And, the list
of reasons is used to add a footer to the email body that lists the
specific reasons for that user.

When generating an outbound email, we add a `References:` header to
make the messages for the same issue thread together and to
authenticate any reply.  That header value is computed using a hash of
the user's email address and the subject line of the emails (which
includes the project name and issue ID number).  Each part is combined
with a secret value stored in Cloud Datastore and accessed via
`secrets_svc.py`.

Most outbound emails include the summary line of the issue, the
details of any updates, and the text of the comment.  However, there
are some cases where Monorail sends a "nerfed" safe subset of
information that consists only of a link to the issue and a generic
message saying that the issue has been created or updated.  We send a
link-only notification when the issue is restricted and the recipient
may be a mailing list.  Monorail cannot know if an email address
represents a mailing list or an individual user, so it assumes that
any address corresponding to a `User` record which has never visited
the site is a mailing list.

Monorail considers an issue to be "noisy" if the issue already has
more than 100 comments and more than 100 starrers.  Such issues can
generate a huge amount of email that would consume our quota and have
low value for most recipients.  A large amount of chatter by
non-members can make it harder for project members to notice the
comments that are important to resolving the issue.  Monorail only
sends notifications for noisy issues if the comment author is a
project member.

## Detailed design: Code walk-throughs

### Individual issue change

1. User A posts a comment on an existing issue 123.

1.  The user's request is handled by the `default` GAE service.
    `work_env` coordinates the steps to update issue 123 in the
    database and invalidate it in caches.

1.  `work_env` calls `PrepareAndSendIssueChangeNotification()` to
    create a task entry that includes a reference to the new comment
    and a field to omit user A being notified.

1.  That task is processed by the `latency-insensitive` GAE service by
    the `NotifyIssueChangeTask` handler.  It determines if the issue is
    noisy, gathers potential recipents by reasons, omits the comment
    author, and checks whether each recipient would be allowed to view
    the issue.  Three different email bodies are prepared: one for
    non-members that has other users' email addresses obscured, one
    for members that reveals email addresses, and one for link-only
    notifications.

1.  The group reason list is then passed to
    `MakeBulletedEmailWorkItems()` which inverts the list, adds
    personalized footers, and converts the plain text email body into
    a simple HTML formatted email body.  It then returns a list with a
    new task record for each email message to be sent.

1.  If the generation task fails for any reason, it will be retried,
    but no email messages are actually sent on the failed run.  If the
    entire process up to this point succeeds, then
    `AddAllEmailTasks()` is called to enqueue each of the single
    message tasks.

1.  Those tasks are handled by the `OutboundEmailTask` handler in the
    `latency-insensitve` service.  Individual tasks are used for each
    outbound email because sending emails can sometimes fail.  Each
    task is automatically retried without rerunning other tasks.

### Bulk issue change

The process is similar to the individual issue change process, except
that a list of allowed recipients is made for each issue.  Then, that
list is inverted to make a list of issues that a given recipient
should be notified about.  This is done in the `NotifyBulkChangeTask`
handler.

When a given recipient is to be notified of multiple issues that were
changed, the email message body lists the updates made and then lists
each of the affected issues.  In contrast, when a given recipient is
to be notified of exactly one issue, the body is formatted to look
like the individual issue change notification.

### Blocking change

When issue 123 is edited to make it blocked on issue 456, participants
in the upstream issue (456) are notified.  Likewise, when a blocking
relationship is removed, another notification is sent.  This is done
by the `NotifyBlockingChangeTask` handler.

### Approval issue change

This is handled by `NotifyApprovalChangeTask`.

TODO: needs more details.

### Date-action notifications

Some date-type custom fields can be configured to send follow-up
reminders when the specified date arrives.

1.  The date-action cron task runs once each day as configured in
    `cron.yaml`.

1.  The `DateActionCron` handler is run in the `latency-insensitive`
    GAE service.  It does an SQL query to find issues that have a
    date-type field set to today's date and that is configured to send
    a notification.  For each such issue, it enqueues a new task to
    handle that date action.

1.  Each of those tasks is handled by `IssueDateActionTask` which
    works like an individual email notification.  The main difference
    is that issue subscribers are not notified and issue starrers are
    only notified if they have opted into that type of notification.
    The handler posts a comment to the issue, calls
    `ComputeGroupReasonList()` to compute a group reason list, calls
    `MakeBulletedEmailWorkItems()` to make individual message tasks,
    and calls `AddAllEmailTasks()` to enqueue those email tasks.

### Inbound email processing

TODO: needs more details

### Alerts

TODO: needs more details

## Detailed design: Source code locations

*  `settings.py`: Configuration of branded domains.  Also, email
   From-line string templates that are used to re-route email
   generated on the dev or staging servers.

*  `businesslogic/work_env.py`: Internal handlers for many changes such
   as updating issues.  It checks permissions, coordinates updates to
   various backends, and enqueues tasks for follow-up work such as
   sending notifications.

*  `features/notify.py`: This file has most email notification task
   handlers.

*  `features/notify_reasons.py`: Functions to compute `AddrPermLists`
   from a list of potential recipients by checking permissions and
   looking up user preferences.  It combines these lists into an
   overall group reason list.  Also, computes list of issue
   subscribers by evaluating saved queries.

*  `features/notify_helpers.py`: Functions to generate personalized
   email bodies and enqueue lists of individual email tasks based on a
   group reason list.

*  `features/dateaction.py`: Cron task for the date-action feature and
   task handler to generate the follow-up comments and email
   notifications.

*  `features/inboundemail.py`: Handlers for inbound email messages and
   bounce messages.

*  `features/commands.py` and `features/commitlogcomands.py`: Parsing
   and processing of issue field assignment lines that can be at the
   top of the body of an inbound email message.

*  `features/alert2issue.py`: Functions to parse emails from our alert
   monitoring service and then create or update issues.

*  `framework/emailfmt.py`: Utility functions for parsing and
   generating email headers.
