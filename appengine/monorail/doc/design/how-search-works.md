# How Search Works in Monorail

[TOC]

## Objective

Our goal for Monorail's search design is to provide a fast issue
search implementation for Monorail that can scale well past one
million issues and give results in about two seconds.  Monorail
supports a wide range of query terms, so we cannot simply predefine
indexes for all possible queries.  Monorail also needs to be
scalable in the number of requests to withstand DoS attacks,
ill-behaved API clients, and normal traffic spikes.  A key requirement
of Monorail's search is to give exact result counts even for fairly
large result sets (up to 100,000 issues).

## Background

From 2005 to 2016, we tracked issues on code.google.com, which stored
issues in Bigtable and indexed them with Mustang ST (a structured
search enhancement to Google's web search).  This implementation
suffered from highly complex queries and occasional outages.  It
relied on caching to serve popular queries and could suffer a
"stampede" effect when the cache needed to be refilled after
invalidations.

When the Monorail SQL database was being designed in 2011, Google
Cloud SQL was much slower than it is today.  Some key factors made a
non-sharded design unacceptable:

*  The SQL database took too long to execute a query.  Basically, the
   time taken is proportional to the number of `Issue` table rows
   considered.  While indexes are used for many of Monorail's other
   queries, the issue search query is essentially a table scan in many
   cases.  The question is how much of the table is scanned.

*  Getting SQL result rows into python was slow.  The protocol between
   the database and app was inefficient, prompting some significant
   work-arounds that were eventually phased out.  And, constructing a
   large number of ProtoRPC internal business objects was slow.  Both
   steps were CPU intensive.  Being CPU-bound produced a poor user
   experience because the amount of CPU time given to a GAE app is
   unpredictable, leading to frustrating latency on queries that
   seemed fine previously.

## Overview of the approach

The design of our search implementation basically addresses these
challenges point-by-point:

Because there is no one index that can narrow down the number of table
rows considered for all possible user issue queries, we sharded the
database so that each table scan is limited to one shard.  For
example, with 10 shards, we can use 10 database instances in parallel,
each scanning only 1/10 of the table rows that would otherwise be
scanned.  This saves time in retrieving rows.  Using 10 DB instances
also increases the total amount of RAM available to those instances
which increases their internal cache hit ratio and allows them to do
more sorting in RAM rather than using slower disk-based methods.

Because constructing ProtoRPC objects was slow, we implemented RAM
caches and used memcache to reduce the number of issues that need to
be loaded from the DB and constructed in python for any individual
user request.  Using RAM caches means that we can serve traffic spikes
for popular issues and queries well, but it also required us to
implement a distributed cache invalidation feature.

Sharding queries at the DB level naturally led to sharding requests
across multiple besearch instances in python.  Using 10 besearch
instances gives 10x the amount of CPU time available for constructing
ProtoRPC objects.  Of course, sharding means that we needed to
implement a merge sort to produce an overall result list that is
sorted.

Another aspect of our approach is that we reduce the work needed in
the main SQL query as much as possible by mapping user-oriented terms
to internal ID integers in python code before sending the query to
SQL.  This mapping could be done via JOIN clauses in the query, but
the information needed for these mappings rarely changes and can be
cached effectively in python RAM.


## Detailed design: Architecture

The parts of Monorail's architecture relevant to search consists of:

*  The `default` GAE service that handles incoming requests from users,
   makes sharded queries to the `besearch` service, integrates the
   results, and responds to the user.

*  The `besearch` GAE service that handles sharded search requests from
   the `default` module and communicates with a DB instance.  The
   `besearch` service handles two kinds of requests: `search` requests
   which generate results that are not permission-checked so that they
   can be shared among all users, and `nonviewable` requests that do
   permission checks in bulk for a given user.

* A master DB instance and 10 replicas.  The database has an
   Invalidate table used for distributed invalidation.  And,
   issue-related tables include a `shard` column that allows us to
   define a DB index that includes the shard ID.  The worst (least
   specific) key used by our issue query is typically `(shard,
   status_id)` when searching for open issues and `(shard,
   project_id)` when searching all issues in a project.

*  There are RAM caches in the `default` and `besearch` service
   instances, and we use memcache for both search result sets and for
   business objects (e.g., projects, configs, issues, and users).

*  Monorail uses the GAE full-text search index library for full-text
   terms in the user query.  These terms are processed before the
   query is sent to the SQL database.  The slowness of GAE full-text
   search and the lack of integration between full-text terms and
   structured terms is a weakness of the current design.

## Detailed design: Key algorithms

### Query parsing

To convert the user's query string into an SQL statement, we first
parse it into query terms using regular expressions.  Then, we build
an abstract syntax tree (AST).  Then, we simplify that AST by doing
cacheable lookups in python.  Then, we convert the simplified AST into
a set of LEFT JOIN, WHERE, and ORDER BY clauses.

It is possible for a query to fail to parse and raise an exception
before the query is executed.

### Result set representations

We represent issue search results as lists of global issue ID numbers
(IIDs).  We represent nonviewable issues as sets of IIDs.

To apply permission checks to search results, we simply use set
membership: any issue IID that is in the nonviewable set for the
current user is excluded from the allowed results.

### Sharded query execution

To manage sharded requests to `besearch` backends, the
`FrontendSearchPipeline` class does the following steps:

1.  The constructor checks the user query and can determine an error
    message to display to the user.

1.  `SearchForIIDs()` calls `_StartBackendSearch()` which determines
    the set of shards that need to be queried, checks memcache for
    known results and calls backends to provide any missing results.
    `_StartBackendSearch()` returns a list of rpc_tuples, which
    `SearchForIIDs()` waits on.  Each rpc_tuple has a callback that
    contains some retry logic.  Sharded nonviewable IIDs are also
    determined. For each shard, the allowed IIDs for the current user
    are computed by removing nonviewable IIDs from the list of result
    IIDs.

1.  `MergeAndSortIssues()` merges the sharded results into an overall
    result list of allowed IIDs by calling `_NarrowFilteredIIDs()` and
    `_SortIssues()`.  An important aspect of this step is that only a
    subset of issues are retrieved.  `_NarrowFilteredIIDs()` fetches a
    small set of sample issues and uses the existing sorted order of
    IIDs in each shard to narrow down the set of issues that could be
    displayed on the current pagination page.  Once that subset is
    determined, `_SortIssues()` calls methods in
    `framework/sorting.py` to do the actual sorting.

### Issue position in list

Monorail's flipper feature also uses the `FrontendSearchPipeline`
class, but calls `DetermineIssuePosition()` rather than
`MergeAndSortIssues()`.  `DetermineIssuePosition()` also retrieves
only a subset of the issues in the allowed IIDs list.  For each shard,
it uses a sample of a few issues to determine the sub-range of issues
that must be retrieved, and then sorts those with the current issue to
determine the number of issues in that shard that would precede the
currently viewed issue.  The position of the current issue in the
overall result list is the sum of the counts of preceding issues in
each shard.  Candidates for the next and previous issues are also
identified on a per-shard basis, and then the overall next and
previous issues are determined.


### Memcache keys and invalidation

We cache search results keyed by query and shard, regardless of the
user or their permissions.  This allows the app to reuse cached
results for different users.  When issues are edited, we only need to
invalidate the shard that that issue belongs to.

The key format for search results in memcache is `memcache_key_prefix,
subquery, sd_str, sid`, where:

 * `memcache_key_prefix` is a list of project IDs or `all`
 * `subquery` is the user query (or one OR-clause of it)
 * `sd_str` is the sort directive
 * `sid` is the shard ID number

If it were not for cross-project search, we would simply cache when we
do a search and then invalidate when an issue is modified.  But, with
cross-project search we don't know all the memcache entries that would
need to be invalidated.  So, instead, we write the search result cache
entries and then an initial modified_ts value for each project if it
was not already there. And, when we update an issue we write a new
modified_ts entry for that issue's project shard. That entry
implicitly invalidates all search result cache entries that were
written earlier because they are now stale.  When reading from the
cache, we ignore any cache entry that corresponds to a project with
modified_ts after the cached search result timestamp, because it is
stale.

We cache nonviewable IID sets keyed by user ID, project ID, and shard
ID, regardless of query.  We only need to invalidate cached
nonviewable IDs when a user role is granted or revoked, when an issue
restriction label is changed, or a new restricted issue is created.


## Detailed design: Code walk-throughs

### Issue search

1. The user makes a request to an issue list page.  For the EZT issue
   list page, the usual request handling is done, including a call to
   `IssueList#GatherPageData()`.  For, the web components list page or
   an API client, the `ListIssues()` API handler is called.

1. One of those request handlers calls `work_env.ListIssues()` which
   constructs a `FrontendSearchPipeline` and works through the entire
   process to generate the list of issues to be returned for the
   current pagination page.  The pipeline object is returned.

The `FrontendSearchPipeline` process steps are:

1.  A `FrontendSearchPipeline` object is created to manage the search
    process.  It immediately parses some simple information from the
    request and initializes variables.

1.  `WorkEnv` calls `SearchForIIDs(`) on the
    `FrontendSearchPipeline`. It loops over the shards and:

  * It checks memcache to see if that (query, sort, shard_id) is
    cached and the cache is still valid.  If found, these can be used
    as unfiltered IIDs.

  * If not cached, it kicks off a request to one of the GAE `besearch`
    backend instances to get fresh unfiltered IIDs.  Each backend
    translates the user query into an SQL query and executes it on one
    of the SQL replicas.  Each backend stores a list of unfiltered
    IIDs in memcache.

  * In parallel, unviewable IIDs for the current user are looked up in
    the cache and possibly requested from the `besearch` backends.

  * Within each shard, unviewable IIDs are removed from the unfiltered
    IIDs to give sharded lists of filtered IIDs.

  * Sharded lists of filtered IIDs are combined into an overall result
    that has only the issues needed for the current pagination page.
    This step involves retrieving sample issues and a few distinct
    sorting steps.

  * Backend calls are made with the `X-AppEngine-FailFast: Yes` header
    set, which means that if the selected backend is already busy, the
    request immediately fails so that it can be retried on another
    backend that might not be busy.  If there is an error or timeout
    during any backend call, a second attempt is made without the
    `FailFast` header. If that fails, that failure results in an error
    message saying that some backends did not respond.

### Issue flipper

For the issue detail page, we do not need to completely sort and
paginate the issues.  Instead, we only need the count of allowed
issues, the position of the current issue in the hypothetically sorted
list, and the IDs of the previous and next issues, if any, which we
call the "flipper" feature.

As of December 2019, the flipper does not use the pRPC API yet.
Instead, it uses an older JSON servlet implementation.  When it is
implemented in the pRPC API, only the first few steps listed below
will change.

The steps for the flipper are:

1.  The web components version of the issue detail page makes an XHR
    request to the flipper servlet with the search query and the
    current issue ref in query string parameters.

1.  The `FlipperIndex` servlet decides if a flipper should be shown,
    and whether the request is being made in the context of an issue
    search or a hotlist.

1.  It calls `work_env.FindIssuePositionInSearch()` to get the
    position of the current issue, previous and next issues, and the
    total count of allowed search results.

1.  Instead of calling the pipeline's `MergeAndSortIssues()`, the
    method `DetermineIssuePosition()` is called.  It retrieves only a
    small fraction of the issues in each shard and determines the
    overall position of the current issue and the IID of the preceding
    and following issues in the sorted list.

We also have special servlets that redirect the user to the previous
or next issues given a current issue and a query.  These allow for
faster navigation when the user clicks these links or uses the `j` or
`k` keystrokes before the flipper has fully loaded.


### Snapshots

To power the burndown charts feature, every issue create and update operation
writes a new row to the `IssueSnapshot` table. When a user visits a chart page,
the search pipeline runs a `SELECT COUNT` query on the `IssueSnapshot` table,
instead of what it would normally do, running a `SELECT` query on the `Issue`
table.

Any given issue will have many snapshots over time. The way we keep track of
the most recent snapshots are with the columns `IssueSnapshot.period_start`
and `IssueSnapshot.period_end`.

If you imagine a Monorail instance with only one issue, each time we add
a new snapshot to the table, we update the previous snapshot's `period_end`
and the new snapshot's `period_start` to be the current unix time. This means
that for a given range (period_start, period_end), there is only one snapshot
that applies. The most recent snapshot always has its period_end set to
MAX_INT.

    Snapshot ID:  1         2         3                 MAX_INT

    Unix time:
    1560000004                        +-----------------+
    1560000003              +---------+
    1560000002    +---------+


## Detailed design: Source code locations

*  `framework/sorting.py`: Sorting of issues in RAM.  See sorting
   design doc.

*  `search/frontendsearchpipeline.py`: Algorithm for determining issue
   position in flipper.  Sequences events for hitting sharded
   backends.  Does set logic to remove nonviewable IIDs from the
   current user's search results.  MergeAndSortIssues() combines
   search results from each shard into a unified result.  Also,
   DetermineIssuePosition() function calculates the position of the
   current issue in a search result without merging the entire search
   result..

*  `search/backendsearchpipeline.py`: Sequence of events to search for
   matching issues and at-risk issues, caching of unfiltered results,
   and calling code for permissions filtering. Also, calls ast2select
   and ast2sort to build the query, and combine SQL results with
   full-text results.

*  `search/backendsearch.py`: Small backend servlet that handles the
   request for one shard from the frontend, uses a
   backendsearchpipeline instance, returns the results to the frontend
   as JSON including an unfiltered_iids list of global issue IDs.  As
   a side-effect, each parallel execution of this servlet loads some
   of the issues that the frontend will very likely need and
   pre-caches them in memcache.

*  `search/backendnonviewable.py`: Small backend servlet that finds
   issues in one shard of a project that the given user cannot view.
   This is not specific to the user's current query.  It puts that
   result into memcache, and returns those IDs as JSON to the
   frontend.

*  `search/searchpipeline.py`: Utility functions used by both frontend
   and backend parts of the search process.

*  `tracker/tracker_helpers.py`: Has a dictionary of key functions used
   when sorting issues in RAM.

*  `services/issue_svc.py`: RunIssueQuery() runs an SQL query on a
   specific DB shard.

*  `search/query2ast.py`: parses the user’s query into an AST (abstract
   syntax tree).

*  `search/ast2ast.py`: Simplifies the AST by doing some lookups in
   python for info that could be cached in RAM.

*  `search/ast2select.py`: Converts the AST into SQL clauses.

*  `search/ast2sort.py`: Converts sort directives into SQL ORDER BY
   clauses.
