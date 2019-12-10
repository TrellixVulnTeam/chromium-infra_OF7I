# Issue lists, grids, and charts

[TOC]

## How do teams use Monorail list, grid, and chart views?

There are many uses for issue aggregate views, including:

*   Finding a particular issue or a set of related issues
*   Seeing issues assigned to you, or checking the status of issues you reported
*   Seeing incoming issues that need to be triaged
*   Understanding the set of open issues that your team is responsible for
    resolving
*   Understanding trends, hot spots, or rates of progress

Monorail has flexible list views so that users can accomplish a wide range of
common tasks without needing external reporting tools. But, for more challenging
understanding tasks, we also offer limited integration with some reporting tools
to Googlers.

## How to search for issues

1.  Sign in, if you want to work with restricted issues or members-only
    projects.
1.  Navigate into the appropriate project by using the project list on the site
    home page or the project menu in the upper-left of each page.
1.  Type search terms into the search box at the top of each project page. An
    autocomplete menu will help you enter search terms. The triangle menu at the
    end of the search box has an option for a search tips page that gives
    example queries.
1.  Normally, Monorail searches open issues only. You can change the search
    scope by using the menu just to the left of the search box.
1.  Press enter or click the search icon to do the search

You can also jump directly to any issue by searching for the issue’s ID number.

## How to find issues that you need to work on

1.  Sign in and navigate to the appropriate project.
1.  Search for `owner:me` to see issues assigned to you. You might also try
    `cc:me` or `reporter:me`. If you are working on a specific component, try
    `component:` followed by the component’s path.

If you are a project member, the project owner may have already configured a
default query that will show your issues or a list of high-priority issues that
all team members should focus on.

Issue search results pages can be bookmarked, so it is common for teams to
define a query that team members should use to triage incoming issues, and then
share that as a link or make a short-link for it.

## How to refine a search query

You can refine a query by editing the query terms in the search box at the top
of the page. You can add more search terms to the end of the query or adjust
existing terms. The autocomplete menu will appear if you position the text
cursor in the existing query term. One common way to adjust search terms is to
add more values to an existing term by using commas, e.g., `Pri:0,1`.

A quick way to narrow down a displayed list of issues is to filter on one of the
shown columns. Click on the column heading, open the `Show only` submenu, and
select a value that you would like to see. A search term that matches only that
value will be added to the query and the list will be updated.

## How to sort an issue list

1.  Do a query to get the right set of issues, and show the column that you want
    to sort on.
1.  Click the column heading to open the column menu.
1.  Choose `Sort up` or `Sort down`.

Labels defined by the project owners will sort according to the order in which
they are listed on the label admin page, e.g., Priority-High, Priority-Medium,
Priority-Low. Teams are also free to use labels that make sense to them without
waiting for the project owner to define them, and those labels will sort
alphabetically.

To sort on multiple columns, A, B, and C: First sort on column, C, then B, and
then A. Whenever two issues have the same value for A, the tie will be broken
using column B, and C if needed.

For multi-valued fields, issues sort based on the best value for that
field.  E.g., if an issue has CC’s for `a-user@example.com` and
`h-user@example.com`, it will sort with the A’s when sorting up and
with the H’s when sorting down.

Project owners can specify a default sort order for issues. This default order
is used when the user does not specify a sort order and when there are ties.

## How to change columns on an issue list

1.  Click on the `...` menu located after the last column heading in the issue
    list.
1.  Select one of the column names to toggle it.

Alternatively, you can hide an existing column by clicking on the column heading
and choosing `Hide column`.

## How to put an issue list into a spreadsheet

You can copy and paste from an HTML table to a Google Spreadsheet:

1.  Do a search and configure the issue list to show the desired columns.
1.  Drag over the desired issues to select all cells.
1.  Copy with control-c or by using the browse `Edit` menu.
1.  Visit the Google Spreadsheet that you wish to put the information into.
1.  Paste with control-v or by using the browse `Edit` menu.

For longer sets of results, project members can export CSV files:

1.  Sign in as a project member.
1.  Do a search and configure the issue list to show the desired columns.
1.  Click the `CSV` link at the bottom of the issue list.  A file will download.
1.  Visit Google Drive.
1.  Choose the menu item to upload a file.
1.  Upload the CSV file that was download.

## How to group rows

The issue list can group rows and show a group heading. For example, issues
could be grouped by priority. To group rows:

1.  Do the issue search and, if needed, add a column for the field that you want
    to use for grouping.
1.  Click on the column header.
1.  Choose `Group rows`.

Each section header shows the count of issues in that section, and sections
expanded or collapsed as you work through the list.

## How to do a bulk edit

1.  Search for issues that are relevant to your task.
1.  Click the checkboxes for the issues that you want to edit, or use the
    controls above the issue table to select all issues on the current page.
1.  Select `Bulk edit` from the controls above the issue list.
1.  On the bulk edit page, describe the reason for your change, and enter values
    to add, remove, or set on each issue.
1.  Submit the form.

## How to view an issue grid

Monorail’s issue grid is similar to a scatter chart in that it can give a
high-level view of the distribution of issues across rows and columns that you
select.

1.  On the issue list page, click `Grid` in the upper-left above the issue list
    table.
1.  The issue grid shows the same issues that were shown in the issue list, but
    in a grid format.
1.  Select the fields to be used for the grid rows and columns.

You can also set the level of detail to show in grid cells: issue tiles, issue
IDs, or counts. Tiles give the most details about issues, but only a limited
number of tiles can fit most screens. If issue IDs are shown, hovering the mouse
over an ID shows the issue summary. Counts can be the best option for a large
set of issues, and clicking on a count link navigates you to a list of specific
issues.

Note: Because some issue fields can be multi-valued, it is possible for a given
issue to appear in multiple places on the issue grid at the same time. The total
count shown above the grid is the number of issues in the search results, even
if some of them are displayed multiple times.

## How to view an issue chart

Monorail’s chart view is a simple way to see the number of issues that satisfy a
query over a period of time, which gives you insights into issue trends.
Monorail uses historical data to chart any query made up of query terms that we
support in charts. Unlike many other reporting tools, you do not need to define
a query ahead of time. To see a chart:

1.  Do a query for the issues that you are interested in. Charts currently
    support only query terms for cc, component, hotlist, label, owner, reporter,
    and status.
1.  Click `Chart` in the upper-left above the issue table.
1.  Hover your mouse over data points to see exact dates and counts.
1.  Use the controls below the chart to adjust date range, grouping, and
    predictions.

## How to see a burndown chart

One of the most important issue trends is the trend toward zero as a team works
to resolve a set of issues related to some upcoming milestone or launch.
Monorail can add a prediction line to the chart, which is commonly called a
burndown chart. Here’s how:

1.  Do a query for the issues that you are interested in. Charts currently
    support only query terms for cc, component, hotlist, label, owner, reporter,
    and status.
1.  Click `Chart` in the upper-left above the issue table.
1.  Click one of the prediction options in the controls below the chart. A
    dotted line will be added to the chart showing the current trend and a
    future prediction.
1.  Optionally, click the chart legend items at the top of the chart to add
    prediction error ranges.
