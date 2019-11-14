# Monorail Concepts

[TOC]

## User accounts

*   Users may visit the Monorail server without signing in, and may view public
    issues anonymously.

*   Signing in to Monorail requires a Google account. People can create a GMail
    account, or create a Google account using an existing email address.

*   User accounts are identified by email address.

*   When you post, your email address is shared with project members.

*   You can access your user profile via the account menu.

*   Your user pages also include a list of your recent posts, your saved
    queries, and your hotlists.

*   The settings page allows you to set user preferences, including a vacation
    message.

*   Monorail allows account linking between certain types of accounts.
    (Currently only allowed between @google.com and @chromium.org accounts). To
    link your accounts:

    1.  Log in to the account that you want to make a child account.
    1.  Navigate to your account 'Profile' page via the dropdown at the
        top-right of every page.
    1.  You should see "Link this account to:". Select the parent account and
        click "Link"
    1.  Switch to the account you've chosen to be the parent account.
    1.  Navigate to the 'Profile' page.
    1.  You should see a pending linked account invite from your first account.
        Click 'Accept'.

*   If you need to completely delete your account, there is a button for that on
    your profile page.

    *   Each issue or comment that you create becomes part of the project, so
        deleting your account simply removes your email address from those
        postings. If you need to delete individual posts, you should do that
        before deleting your account.

*   If Monorail sends an email to a user and that email bounces, the account is
    marked as bouncing. A user can sign in and clear the bouncing flag via their
    profile page.

*   The bouncing state, time of last visit, and any vacation message are all
    used to produce a user availability message that may be shown when that user
    is an issue owner or CC’d.

*   Project owners and site admins may ban user accounts that are used to post
    spam or other content that does not align with Monorail’s mission.

## Projects and roles

*   Each project contains issues, grants roles to project members, and
    configures how issues are tracked in that project.

*   Projects can be public or members-only. Only project members may access the
    contents of a members-only project, however the name of a members-only
    project may occur in comments and other places throughout the site.

*   The project owners are responsible for configuring the project to fit their
    development process (described below).

*   Project members have permissions to edit issues, and they are listed in the
    autocomplete menus for owner and CCs.

*   While most activity on a Monorail server occurs within a given project, it
    is also possible to work across projects. For example, a hotlist can include
    issues from multiple projects.

*   Some projects that we host have a branded domain name. Visiting one of these
    projects will redirect the user to that domain name.

*   When an old project is no longer needed, it can be archived or marked as
    moved to a different URL.

## Issues, comments, and attachments

Issues:

*   Each issue is given a unique numeric ID number when the issue is created.
    Issue IDs count up so that they serve as a project odometer and a quick way
    for members to judge the age of an issue.

*   Each issue has a set of fields including summary, reporter, owner, CCs,
    components, labels, and custom fields.

*   Issues may be blocked on other issues. The relationship is two-way, meaning
    that if issue A is blocked on issue B, then issue B is blocking issue A.

*   Each issue has several timestamps, including the time when it was opened,
    when it was most recently closed, when the status was most recently set,
    when the components were modified, and when the owner was most recently
    changed.

Comments:

*   Each comment consists of the comment text, the email address of the user
    which posted the comment, and the time at which the comment was posted.

*   Each comment has a unique anchor URL that can be bookmarked and shared.

*   Each comment has a sequence number within the issue. E.g., comment 1 is the
    first comment on the issue after it was reported. Older comments may be
    initially collapsed in the UI to help focus attention on the most recent
    comments.

*   Comment text is unicode and may include a wide range of characters including
    emoji.

*   Whitespace in comments is stored in our database but extra whitespace is
    usually not visible in the UI unless the user has clicked the “Code” button
    to use a fixed-width code-friendly font.

*   All comments on an issue have the same access controls as the issue.
    Monorail does not currently support private comments. If you need to make a
    private comment to another issue participant, you should do it via email or
    chat.

*   Each comment can list some amendments to the issue. E.g., if the issue owner
    was changed when the comment was posted, then the new owner is shown.

*   Each comment is limited to 50 KB. If you wish to share log files or other
    long documents, they should be uploaded as attachments or shared via Google
    Drive or another tool.

*   Comments can be marked as spam or marked deleted. Even these comments are
    still part of the project and may be viewed by project members, if needed.

Attachments:

*   Each comment can contain some attachments, such as logs, text files, images,
    or videos.

*   Monorail supports up to 10 MB worth of attachments on each issue. Larger
    attachments should be shared via Google Drive or some other way.

*   Monorail allows direct viewing of images, videos, and certain text files.
    Other attachment types can only be downloaded.

*   Attachment URLs are not all shareable. If you need to refer to an
    attachment, it is usually best to link to the comment that contains it.

## Issue fields and labels

*   Issues have several built-in fields including summary, reporter, owner, CCs,
    status, components, and various timestamps. However, many fields that are
    built into other issue tracking tools are configured as labels or custom
    fields in Monorail, for example, issue priority.

*   Issue labels are short strings that mean something to project members. They
    are described more below.

*   Project owners can define custom fields of several different types,
    including enums, strings, dates, and users.

*   There are three main types of labels:

    *   OneWord labels contain no dashes and are treated similarly to hashtags
        and tags found in other tools.
    *   Key-Value labels have one or more dashes and are treated almost like
        enumerated-type fields.
    *   Restriction labels start with “Restrict-” and have the effect of
        limiting access to the issue. These are described more below.

*   A list of well-known labels can be configured by project owners. Each
    well-known label can have a documentation string and it will sort by its
    rank in the list.

*   Well-known labels are offered in the autocomplete menus. However, users are
    still free to type out other label strings that make sense to their team.

## Labels for flexibility and process evolution

Monorail normally treats key-value labels and custom fields and labels in the
same way that built-in fields are treated. For example:

*   When displayed in the UI, they are shown individually as equals of built-in
    fields, not nested under a subheading.

*   They can be used as columns in the list view, and also in grid and chart
    views.

*   Users can search for them using the same syntax as built-in fields.

Monorail tracks issues over multi-year periods, so we need to gracefully handle
process changes that happen from time to time. In particular, Monorail allows
for incremental formalization of enum-like values. For example:

*   A team may start labeling issues simply as a way to identify a set of
    related issues. E.g., `LCDDisplay` and `OLEDDisplay`.

*   The team might decide to switch to Key-Value labels to make it easier to
    query and read in the list views. E.g., `Display-LCD` and `Display-OLED`.

*   If more people start using those labels, the project owners might make them
    well-known labels and add documentation strings to clarify the meaning of
    each. However, oddball labels like `Display-Paper` could still be used. This
    configuration might last for years.

*   If these labels used so much that it seems worth adding a row to the editing
    form, then the project owners can define an enum-type custom field named
    `Display` with the well-known label suffixes as possible values. This would
    discourage oddball values, but they could still exist on existing issues.

*   At a later date, the project owners might review the set of fields, and
    decide to demote some of them back to well-known labels.

*   If the process changes to the point that it is no longer useful to organize
    issues by those labels, they can be removed from the well-known list, but
    still exist on old issues.

## Permissions

*   In Monorail, a permission is represented by a short string, such as `View`
    or `EditIssue`.

*   Project owners grant roles to users in a project. Each role includes a list
    of permission strings.

*   The possible roles are: Anonymous visitor, signed-in non-member,
    contributor, committer, and owner.

*   Project owners may also grant additional permissions to a project member.
    For example, a user might be a contributor, plus `EditIssue`.

*   When a user makes a request to the Monorail server, the server checks that
    they can access the project and that they have the needed permission for
    that action. E.g., viewing an issue requires the `View` permission.

*   If an issue has a restriction label of the form
    `Restrict-Action-OtherPermission` then the user may only perform `Action` if
    they could normally do it, and if they also have permission
    `OtherPermission`. For example, `Restrict-View-EditIssue` means that the
    only users who can view the issue are the ones who could also edit it.

*   Since both permissions and restriction labels are just strings, they can be
    customized with words that make sense to the project owners. For example, if
    only a subset of project members are supposed to deal with security issues,
    they could be granted a `SecurityTeam` permission and those issues labeled
    with `Restrict-View-SecurityTeam`. The most common example is
    `Restrict-View-Google`.

*   Restriction labels can be added in any of the ways that other labels can be
    added, including adding them directly to an individual issue, bulk edits,
    filter rules, or including them in an issue template.

*   Regardless of restriction labels, the issue reporter, owner, CC’d users, and
    users named in certain user-type custom fields always have permission to
    view the issue. And, issue owners always have permission to edit.

*   Project owners and site administrators are not subject to restriction
    labels.

## Project configuration

Projects are configured to define the development process used to track issues,
including:

*   The project description, optional link to a project home page, and optional
    logo

*   A list of open and closed issue statuses. Each status can have a
    documentation string.

*   A list of well-known labels and their documentation strings.

*   A list of custom fields, each with a documentation string and validation
    options.

*   A list of issue templates to use when creating new issues.

*   A list of components, each with a documentation string, auto-CCs, and labels
    to add.

*   A list of filter rules to automatically add some issue fields based on other
    values.

*   Default list and grid view configurations for project members.

<!-- TODO: These areas of project configuration are covered more in
     the Project Owner Guide. -->

## Personal hotlists

*   Each user has a list of personal hotlists that can be accessed via the
    account menu.

*   A hotlist is a ranked list of issues, which can belong to multiple projects.

*   Users can rerank issues in a hotlist by dragging them up or down.

*   Issues can be added to a hotlist via the hotlist page, the issue detail
    page, or issue list.

*   Each hotlist belongs to one user, but that user can add other users to be
    editors.

*   Each issue in a hotlist can also have a short note attached.

*   Hotlists themselves can be public or members-only. A user who is allowed to
    view a hotlist will only see the subset of issues that they are allowed to
    view as determined by issue permissions. Hotlists do not affect issue
    permissions.
