# Site Admin's Guide

[TOC]

## What are Monorail site administrator accounts and what are they used for?

Site administrators are like super-users in Monorail.  A site admin
account can perform any action that any other account can perform, and
some that are only available for site administrators.  While most
permissions can be granted to project members by project owners, some
of the administrative permissions are reserved for site admins only.

Site admins have the ability to create projects and user groups.  They
can also make changes to existing projects, user groups, users, or
issues on behalf of project owners that are having trouble making the
desired changes for some reason.  For example, a site admin might help
a project owner by setting up an initial project configuration.  Both
project owners and site admins can ban users from the site to help
fight spam and abuse.


## How to create a project

1.  Sign into your site admin account.
1.  Visit the site home page.
1.  Click the `Create a new project` link.
1.  Fill in the project name, summary, and description.
1.  Submit the form.

1.  In the new project, visit the People page to grant a role to a
    project owner and remove yourself.

## How to delete a project

1.  Sign into your site admin account.
1.  Open the gear menu and select `Administer`.
1.  Click the `Advanced` tab at the top of the page.
1.  Click a button to `Archive` the project.

Site admins also have a `Doom` option that schedules the project for
deletion in 90 days.  The `Archive` options will be a better choice
for most projects because storage space is typically not a problem for
our site.

## How to increase the project storage quota

1.  Sign into your site admin account and visit any page in the project.
1.  Open the gear menu and select `Administer`.
1.  Click the `Advanced` tab at the top of the page.
1.  Type in a new storage limit.  The limit is measured in megabytes.
1.  Click `Update Quota`.

## How to view the list of user groups

1.  Sign into your site admin account.
1.  Visit the `/g` URL to see the list of user groups.

There is currently no link to navigate to that page.  It is only
accessible to site admins.

## How to create a new user group

1.  Sign into your site admin account.
1.  Visit the `/g` URL to see the list of user groups
1.  Click `Create Group`.
1.  Fill in the form and submit it.

Monorail has three types of user groups: native groups that are
managed entirely within Monorail, synchronized user groups that are
periodically copied from Google Groups or other sources, and computed
user groups that are based entirely on email address domain name.  To
set up a synchronized user group, see the Monorail playbook.

## How to ban a user account

1.  Sign into your site admin account.
1.  Visit the user’s profile page.
1.  Fill in a reason to ban the user.  Or, click `Ban this user as a spammer`.

The reason field serves as a note to other site admins, it is not
shown to the user or other users.

If you use the `Ban this user as a spammer` button, all of the issues
and comments posted by that user will be marked as spam.

## How to completely delete a user account

1.  Sign into your site admin account.
1.  Visit the user's profile page.
1.  Click `Delete user account`.

The user record will be deleted from our database.  Any references to
that user in issue fields or comment author lines will be removed or
changed to `a deleted user`, but the content itself will be retained
as part of the project that it was contributed to.
