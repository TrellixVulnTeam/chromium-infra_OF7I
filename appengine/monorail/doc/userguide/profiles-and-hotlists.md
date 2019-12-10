# User Profiles and Hotlists

[TOC]

## The account menu

The user account menu is located at the far upper-right of each Monorail page.
When you are signed out, it offers a link to sign in. When you are signed in, it
offers a menu with choices to switch users, access your user pages, or sign out.
User pages include the user profile, updates, settings, saved queries, and
hotlists.

## The user profile page

Each Monorail user has a profile page that can be accessed at URL `/u/EMAIL` or
via their user ID number. You can access your own profile page through the
account menu. You can click to access the profile page of any user who you see
mentioned on an issue detail page as the issue reporter, owner, CC'd user, or a
comment author.

The user profile page lists projects where that user has a membership. The
profile page shows how long it has been since the user used that account to
visit the site. Please note that some users have multiple accounts, so they may
have visited more recently using a different account. Also, if the user has set
a vacation message, that message is shown here.

Any project owner may ban a user from the site by clicking a button on the user
profile page. This is one way that we fight spam and abuse.

## Linked accounts

Googlers who have an @chromium.org account may wish to link it to their
@google.com account. These two types of accounts can be linked with one becoming
the parent account and the other becoming the child account.

When accounts are linked:

*   Using the child account will display a reminder notice to switch to the
    parent account.

*   When signed in to the parent account, the user also has all permissions of
    the child account.

*   If both accounts would be listed in an autocomplete menu, only the parent
    account is listed.

*   If both accounts would be notified of an issue change, only the parent
    account is notified.

*   Searching using the `me` keyword will match issues that reference either
    account.

To link accounts:
1.  Sign in to the account that you want to become the child
    account (the one that you don't intend to use any longer).
1.  Visit the profile page for the child account.
1.  Invite the other account to be the parent account.
1.  Use the account menu to switch users to the parent account (the one
    that you intend to use from now on).
1.  Use the account menu to go to the profile page for the parent
    account. This is not the profile that you are already on.
1.  Accept the invitation to link the child account.

To unlink accounts: sign in as either account, use the account menu to navigate
to your profile page, and click the `Unlink` button.

## The user settings page

The settings page allows users to set user preferences for their account. You
can navigate to the settings page by signing in and selecting `Settings` from
the account menu.

On that page you can set preferences that affect:

*   Privacy: How your email address is displayed to non-members.

*   Notifications: What triggers notifications to you and how they are
    formatted.

*   Community interactions: Opt into settings that help avoid accidental
    oversharing.

*   Availability: You can let other users know that you are away.

Site administrators can also view and change the settings for any other user on
that user's profile page.

## The user activity page

The user updates page lists recent activity by that user. This page can be
reached by clicking `Updates` in the account menu for your own updates, or via
the `Updates` tab on any user's profile page.

The list of updates includes new issue reports and comments posted on existing
issues. Each row show how long ago the activity happened, which issue was
affected, and the content of the change. You can click to expand each row to
show more details.

The list of issue changes only includes rows for issues that the signed in user
is currently allowed to view.

## The user hotlists page

The user hotlists page lists hotlists that a user owns or can edit. This page
can be reached by clicking `Hotlists` in the account menu for your own hotlists,
or via the `Hotlists` tab at the top of any users profile page.

Clicking on a row in the hotlists table navigates to the list of issues in that
hotlist. When viewing a hotlist, the hotlist owner and editors may rerank issues
in the hotlist, and they may add or remove issues. Reranking issues in a hotlist
is only possible when the issues are sorted by rank. Reranking is done by
dragging a gripper icon up or down the list.

In the list of hotlists, only hotlists that the signed in user is allowed to
view are shown. And, within a specific hotlist, only issues that the signed in
user is allowed to view are shown.

## How to create a hotlist

1.  Sign in and select `Hotlists` from the account menu.
1.  Click the `Create hotlist` button.
1.  Choose a name and provide a description for the hotlist.
1.  You may list other users who should be able to edit the hotlist.
1.  Hotlists are only visible to hotlist members by default, but you can make
    your hotlist public.
1.  Submit the form.

It is also possible to create a hotlist directly from an issue list, an issue
detail page, or an existing hotlist page. See below for details.

## Who can view a hotlist?

A public hotlist can be viewed by anyone on the Internet, even anonymous users.

A members-only hotlist can only be viewed by the hotlist owner and members
listed on the hotlist people page. You can add or remove people from your
hotlist by clicking the `People` tab at the very top of any hotlist page.

Hotlists do not affect issue permissions. The individual issues within a hotlist
are subject to the normal permission checking for issues. If a user cannot view
an issue, they will not see it listed in the hotlist, even if they can view or
edit the hotlist itself.

## How to add or remove issues to a hotlist

There are several ways to do it.

Starting from an issue detail page, you can add the issue to one or more
hotlists, or remove it:

1.  Sign in and view an issue detail page.
1.  Click `Update your hotlists` in the issue data column.
1.  In the dialog box, check or uncheck the names of hotlists to add or remove.
1.  Alternatively, you can create a new hotlist directly from that dialog box.

Project members can add multiple issues to hotlists by starting from the issue
list page or an existing source hotlist:

<!-- TODO: The WC version of the issue list does not require the user
to be a member. -->

1.  Sign in as a project member and view an issue list page or existing hotlist.
1.  Select one or more issues by clicking checkboxes or using the `x` keystroke.
1.  Click `Add to hotlist...` in the action options above the issue list.
1.  In the dialog box, check the names of hotlists to add the issues to.
1.  Alternatively, you can create a new hotlist directly from that dialog box.

Starting from the list of issues in the target hotlist:

1.  Sign in as the hotlist owner or a member who can edit the hotlist.
1.  Click `Add issues...` from the actions above the issue list.
1.  Type in a comma-separated list of project names and issue IDs. For example
    `chromium:1234`.

## How to rerank issues in a hotlist

1.  Sign in as the hotlist owner or a member who can edit the hotlist.
1.  Visit the hotlist issue list page.
1.  Don't sort the issues by any column heading other than Rank.
1.  As you hover the mouse over each issue row, a gripper icon will appear in
    the far left column.
1.  Drag the gripper icon up or down the list to place the selected issue in the
    new position.

## How to delete a hotlist

1.  Sign in as the hotlist owner.
1.  Visit the hotlist issue list page.
1.  Click the `Settings` tab at the very top of the page.
1.  Click `Delete hotlist` and confirm the deletion.

## How can I completely delete my account?

You can delete your Google account at http://myaccount.google.com. If you do
that, a few days later, Monorail will be notified of the account deletion. At
that time, issues and comments posted by that account will be changed to
indicate that the author is `a deleted user`. The issues and comments themselves
are your contribution to the project and they remain a part of the project.

If you wish to completely delete your Monorail account without deleting your
Google account, please
[file an issue](http://bugs.chromium.org/p/monorail/issues/entry?labels=Restrict-View-Google).
