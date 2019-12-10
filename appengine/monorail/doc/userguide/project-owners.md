# Project Owner's Guide

[TOC]

## Why does Monorail have projects?

Each project contains issues, grants roles to project members, and
configures how issues are tracked in that project.

Projects are coarse-grained containers that provide the most basic
issue organization and access control capabilities.  For example,
issues related to the Chromium browser are in `/p/chromium`, while
issues for the Monorail issue tracker are in `/p/monorail`.  Monorail
has many ways to organize issues, such as labels and components, but
at the highest level, issues are organized by project.  Likewise,
Monorail has many ways to control access to issues through restriction
labels, but at the highest level a user either has permission to visit
an entire project or they do not.  Projects also provide a
coarse-grained life-cycle for issues: when the entire project is
archived, all issues belonging to that project become inaccessible.

The rest of this chapter deals with how project owners can configure
the issue tracking process within a project.  Each project is intended
to have a single, unified, and coordinated process for tracking
issues.  If two issues are in the same project, they should be
expected to have roughly the same life-cycle and meaningful fields,
whereas two issues in two separate projects might be tracked in fairly
different ways.  Also, the set of possible issue owners is determined
by the members of the project, so two issues in two distinct projects
could have two distinct sets of possible issue owners.

Unlike some other issue tracking tools, components in Monorail are not
a unit of process definition: an issue can be in zero, any one, or any
number of components within a project. The components should just
provide context as to which part of the project's source code has the
defect and which teams should be CC'd on the issue.  Every issue in a
project should have the same life-cycle regardless of component.  Any
member of a project could be the owner of any issue in that project,
regardless of components.

Issues can be moved between projects, but that is uncommon, and they
are contained within exactly one project at any time.  When an issue
is moved between projects, it is likely that several fields of the
issue will need to be updated, such as the status and owner.  In
contrast, components within a project can usually be added to or
removed from an issue while keeping other fields unchanged.

## How to quickly remove spam and spammers

The purpose of Monorail is to help developers resolve software defects
and other issues.  Any comments that seem to be spam, abuse, or wildly
off topic should be removed from the site.  That can be done by using
the `...` menu on comments or issues.

Any project owner can ban a user from the site by clicking on the user
email link to get to that user’s profile page, and then clicking `Ban
Spammer`.  All comments and issues entered by that user are
automatically marked as spam.

## How to grant roles to project members

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `People`.
1.  Click the `Add members` button.
1.  Enter the email addresses of the users that you want to add to the project.
1.  Choose the role that they should have: Owner, Committer, or Contributor.
1.  Click `Save changes`.

Once a user has been granted a role in the project, the people list
page will have a row for that user.  Anyone who can visit the project
can click a project member row to see details of that user’s
permissions in the project on a people detail page.  Project owners
can use the people details page to change the role of a user or grant
them individual permissions.

User roles in a project can be removed by clicking buttons on either
the people list page or people detail page.

## How to configure statuses

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Development process`.
1.  Click the `Statuses` tab at the top of the page.
1.  Type open and closed status definition lines in two text
    input areas on that page.
1.  Click `Save changes`.

The syntax of a status definition line is `[#]StatusName[=
docstring]`.  Where `#` indicates that the status is deprecated.
`StatusName` is the name of the status, which may contain dots,
dashes, and underscores, but no spaces.  And, the optional `docstring`
is the documentation string that will be displayed to users to explain
the meaning of that status.

Deprecated status values are not offered in autocomplete menus or the
status field menu.  However, they are kept in the system so that
existing issues that have that status can be sorted according to the
logical rank.  In contrast, a status value that is no longer desired
could be simply deleted, which would remove it from menu choices and
also lose the logical ranking of that status value.

The status definition page also has a field to list statuses that
indicate that an issue is being merged into another issue.  Usually
that is set to simply `Duplicate`.  However, it is possible to use a
different name for that status that fits your process better, or to
list multiple such statuses.

## How to configure labels

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Development process`.
1.  Click the `Labels and fields` tab at the top of the page.
1.  Type label definition lines in the text input area.
1.  Click `Save changes`.

The syntax of a label definition line is `[#]LabelName[= docstring]`.
Where `#` indicates that the label is deprecated.  `LabelName` is the
name of the status, which may contain dots, dashes, and underscores,
but no spaces.  And, the optional `docstring` is the documentation
string that will be displayed to users to explain the meaning of that
label.

It is common to define a set of related Key-Value labels that all have
the same Key.  The Monorail user interface treats them somewhat like
enum fields.  The Key part of the label can be used in queries, as
search result column headings, or as grid axes.  Some Key strings can
be listed as exclusive prefixes, which means that the Monorail UI will
not offer autocomplete options for another value once an issue has one
of those Key-Value labels.

Deprecated labels values are not offered in autocomplete menus, just
as with deprecated status values.  See the section above for details.

## How to configure custom fields

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Development process`.
1.  Click the `Labels and fields` tab at the top of the page.
1.  To edit an existing custom field, click on the row for that
    custom field in the field definition table.
1.  Or, to create a new custom field, click `Add field`.

The form used to create or edit a field definition consists of the
field name, field type, and various validation options that are
appropriate to that field type.  For example, an integer custom field
could specify a minimum or maximum value.  Most details of a field
definition can be changed later, but the name cannot.  Also, a deleted
field name cannot be reused.

Enum-type custom fields are stored as labels in Monorail's database.
If you start to create an enum-type custom field with name "Key", you
will immediately see enum values offered for each existing Key-Value
label that has the same Key part.  The syntax for defining new enum
options is `EnumValue[= docstring]`.

Custom fields may be configured to be applicable to any issue or only
to issues that have a specific `Type-*` label.  And, the field can be
optional or required on issues where it is applicable.  For example, a
DesignDoc custom field with a link to a design document might be a
required field for any issue that has the Type-Design-Review label.

Some fields are more commonly used than others.  In large projects,
there may be variations of the software development process that are
only used with a few issues.  Over time, more and more such process
variations will be defined, and the total set of custom fields to
support all those different variations could make issue editing forms
long and complex.  Monorail helps manage that situation by allowing
fields to be defined as important enough to always be offered as a
visible field when the field is applicable to the issue, or only
important enough to be kept behind a `Show all fields` link.

Project owners may edit any field.  Each field may also specify a list
of field administrators who are also allowed to edit that field.  This
helps project owners delegate responsibility for configuring fields
used in certain development processes to the developers who perform
those processes.

## How to configure approvals

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Development process`.
1.  Click the `Labels and fields` tab at the top of the page.

TODO: Write more detail here.

## How to configure filter rules

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Development process`.
1.  Click the `Rules` tab at the top of the page.
1.  Fill in some rule predicates and consequences.
1.  Submit the form.

Filter rules are if-then rules that automatically add information to
issues when the issues are created, when the issues are updated, and
when the rules change.  Rules add derived values to issues, which are
stored separately from the explicitly set values on the issue itself.
When a value is explicitly set, that value overrides the derived
value, however the derived value itself cannot be edited except by
changing the rule.

The purpose of filter rules is to allow the explicit values on an
issue to focus on capturing the details of the problem situation,
while pragmatic concerns such as access controls and prioritization
can have meaningful defaults set automatically.

For example, there could be a rule that says `if [type=Defect
component=Authentication] then add label Restrict-View-SecurityTeam`.
That would mean that any defect in the system’s authentication
component should be restricted to only be viewable by the security
team (and the reporter, owner, and any CC’d users).

## How to configure issue templates

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Development process`.
1.  Click the `Templates` tab at the top of the page.

Issue templates are used to create new issues with some of the details
filled in.  Each template has a name and the initial values for the
issue summary, description, and other fields.  Most templates will be
available to all users who can report issues in the project, but it is
also possible to restrict a template to only project members.

On that page you can set the default templates that are used for new
issue reports by project members or non-members.  And, there is a list
of existing templates and a button to create a new template.

Each template must have a unique name.  Templates cannot be renamed
after creation, but you can create a new template with the desired
name then delete the original template.

Template summary lines that begin with text inside brackets, such as
`[Deployment] Name of system that needs to be deployed` will cause the
issue entry page to keep the bracketed text when the user initially
types the summary line.  Also, any labels that end with a question
mark, like `Key-?`, will cause the issue entry page to require the
user to edit that label before submitting the form.

Each template can have a comma-separated list of template
administrators who are allowed to edit that template.  This allows
project owners to delegate authority to maintain certain templates to
the teams that work on issues that use that template.  However, the
overall set of templates is controlled by the project owners.

## How to configure components

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Development process`.
1.  Click the `Components` tab at the top of the page.

Components form a high-level outline of the software being developed
in the project so that an issue can be related to the part of that
software that needs to change.  For example, if a software system has
architectural tiers for database access, business logic, and user
interface, that would suggest using three components in Monorail.  If
a piece of software is developed by a different team or using
different processes, then using a separate monorail project may be
more appropriate.

The components list page shows a list of all currently active
components in the project.  It can be filtered to show a smaller set
of components, for example just the components that the signed-in user
is involved in.  Showing all components includes both active
components and components that have been marked as deprecated.

Each component is identified by a path string that consists of a list
of names separated by greater-than signs, e.g., `Database>Metrics`.
When searching for issues by component, subcomponents are normally
included in the results.

The main purpose of components is to indicate the part of the software
that needs to change to resolve an issue.  That could be determined as
part of the investigative work needed to fully document the issue.
Monorail components help with some of the pragmatic aspects of issue
tracking by automatically adding labels or CCing people who might help
resolve the issue.

Each component can have a comma-separated list of component
administrators who are allowed to edit that component.  This allows
project owners to delegate authority to maintain certain components to
the teams that work on issues that use that component.  However, the
overall set of components and their organization is controlled by the
project owners.

## How to configure default views

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Development process`.
1.  Click the `Views` tab at the top of the page.
1.  Fill in a default query for project members to help them stay
    focused on the issues that are most important for the team as a
    whole, or set it to `owner:me` to focus each team member on
    resolving the issues assigned to them.
1.  Fill in default list columns and grid options.
1.  Submit the form.

## How to administer project settings

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Administer`.

This page allows project owners to edit the project summary line,
description, access level and some other settings.  The description
can be written in Markdown.

## How to view the project storage quota

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Administer`.
1.  Click the `Advanced` tab at the top of the page.

The second section of the page shows how much storage space has been
used for attachments in this project and the current limit.  If the
usage reaches the limit, users will no longer be offered the option to
add attachments to issues.  Site administrators can increase the
storage limit for each project.

## How to move, archive, or delete a project

1.  Sign in as a project owner and visit any page in your project.
1.  Open the gear menu and select `Administer`.
1.  Click the `Advanced` tab at the top of the page.
1.  Click a button to `Archive` the project.
1.  Or, fill in a new project URL and click the `Move` button to
    indicate that the project has moved.

When a project is archived, only project owners may access the content
of the project.  Also, ‘Unarchive’ and `Delete` options will be
offered on that page.  If the project owner clicks the `Delete`
button, the contents of the project will immediately become
inaccessible to any users, and all data for that project will be
deleted from Monorail's database within a few days.
