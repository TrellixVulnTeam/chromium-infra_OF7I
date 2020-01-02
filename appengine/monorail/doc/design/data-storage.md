# Monorail Data Storage

## Objective

Monorail needs a data storage system that is expressive, scalable, and
performs well.

Monorail needs to store complex business objects for projects, issues,
users, and several other entities.  The tool is central to several
software development processes, and we need to be able to continue to
add more kinds of business objects while maintaining high confidence
in the integrity of the data that we host.

The database needs to scale well past one million issues, and support
several thousand highly engaged users and automated clients, while
also handling requests from many thousands more passive visitors.
While most issues have only a few kilobytes of text, a small number of
issues can have a huge number of comments, participants, attachments,
or starrers.

As a broad performance guideline, users expect 90% of operations to
complete in under one second and 99% of operations to be done in under
two seconds.  That timeframe must include work for all data
transmission, and business logic, which leaves under one second for
all needed database queries.  The system must perform well under load,
including during traffic spikes due to legitimate usage or attempts to
DoS the site.

Of course, we need our data storage system to be secure and reliable.
Even though the data storage system is not accessed directly by user
requests, there is always the possibility of an SQL injection attack
that somehow makes it through our request handlers.  We also need
access controls and policies to reduce insider risks.

## Background

Monorail evolved from code.google.com which used Bigtable for data
storage and a structured variant of Google's web search engine for
queries.  Using Bigtable gave code.google.com virtually unlimited
scalability in terms of table size, and made it easy to declare a
schema change at any time.  However, it was poor at returning complete
result sets and did not enforce referential integrity.  Over time, the
application code accumulated more and more layers of backward
compatibility with previous data schemas, or the team needed to spend
big chunks of time doing manual schema migrations.

Monorail is implemented in python and uses protocol buffers to
represent business objects internally.  This worked well with Bigtable
because it basically stored protocol buffers natively, but with a
relational database it requires some ORM code.  The construction of
large numbers of temporary objects can make python performance
inconsistent due to the work needed to construct those objects and the
memory management overhead.  In particular, the ProtoRPC library can
be slow.

## Approach

Monorail uses Google Cloud SQL with MySQL as its main data storage
backend.  The key advantages of this approach are that it is familiar
to many engineers beyond Google, scales well to several million rows,
allows for ad-hoc queries, enforces column types and referential
integrity, and has easy-to-administer access controls.

We mitigate several of the downsides of this approach as follows:

*  The complexity of forming SQL statements is handled by `sql.py`
   which exposes python functions that offer most options as keyword
   parameters.

*  The potential slowness of executing complex queries for issue search
   is managed by sharding queries across replicas and using an index
   that includes a `shard_id` column.

*  The slowness of constructing protocol buffers is avoided by using a
   combination of RAM caches and memcache.  However, maintaining
   distributed caches requires distributed invalidation.

*  The complexity of ORM code is managed by organizing it into classes
   in our services layer, with much of the serialization and
   deserialization code in associated cache management classes.

*  The security risk of SQL injection is limited by having all code
   paths go through `sql.py` which consistently makes use of the
   underlying MySQL library to do escaping, and checks each statement
   against an allow-list of regular expressions to ensure that no
   user-controlled elements are injected.

## Detailed design: Architecture

Monorail is a GAE application with multiple services that each have
automatic scaling of the number of instances.  The database is a MySQL
database with one master for most types of reads and all writes, plus
a set of read-only replicas that are used for sharded issue queries
and comments.  The main purpose of the sharded replicas is to
parallelize the work needed to scan table rows during issue search
queries, and to increase the total amount of RAM used for SQL sorting.
A few other queries are sent to random replicas to reduce load on the
master.

To increase the DB RAM cache hit ratio, each logical data shard is
sent to a specific DB replica.  E.g., queries for shard 1 are sent to
DB replica 1.  In cases where the desired DB replica is not available,
the query is sent to a random replica.  An earlier design iteration
also required `besearch` GAE instances to have 1-to-1 affinity with DB
replicas, but that constraint was removed in favor of automatic
scaling of the number of `besearch` instances.

## Detailed design: Protections against SQL injection attacks

With very few exceptions, SQL statements are not formed in any other
place in our code than in `sql.py`, which has had careful review.
Values used in queries are consistently escaped by the underlying
MySQL library.  As a further level of protection, each SQL statement
is matched against an allow-list of regular expressions that ensure
that we only send SQL that fits our expectations.  For example,
`sql.py` should raise an exception if an SQL statement somehow
specified an `INTO outfile` clause.

Google Cloud SQL and the MySQL library also have some protections
built in.  For example, a statement sent to the SQL server must be a
single statement: it cannot have an unescaped semicolon followed by a
second SQL statement.

Also, to the extent that user-influenced queries are sent to DB
replicas, even a malicious SQL injection could not alter data or
permissions because the replicas are read-only.

## Detailed design: Two-level caches and distributed cache invalidation

Monorail includes an `AbstractTwoLevelCache` class that combines a
`RAMCache` object with logic for using memcache.  Each two-level cache
or RAM cache is treated like a dictionary keyed by an object ID
integer.  For example, the user cache is keyed by user ID number.
Each cache also has a `kind` enum value and a maximum size.  When a
cache is constructed, it registers with a singleton `CacheManager`
object that is used during distributed invalidation.

Each type of cache in Monorail implements a subclass of
`AbstractTwoLevelCache` to add logic for retrieving items from the
database on a cache miss and deserializing them.  These operations all
work as batch operations to retrieve a collection of keys into a
dictionary of results.

When retrieving values from a two-level cache, first the RAM cache in
that GAE instance is checked.  On a RAM miss, memcache is checked.  If
both levels miss, the `FetchItems()` method in the cache class is run
to query the database and deserialize DB rows into protocol buffer
business objects.

Values are never explicitly added to a two-level cache by calling
code.  Adding an item to the cache happens only as a side effect of a
retrieval that had a cache miss and required a fetch.

When services-level code updates a business object in the database, it
calls `InvalidateKeys()` on the relevant caches.  This removes the old
key and value from the local RAM cache in that GAE instance and
deletes any corresponding entry in memcache.  Of course, updating RAM
in one GAE instance does not affect any stale values that are cached
in the RAM of other GAE instances.  To invalidate those items, a row
is inserted into the `Invalidate` table in the DB.  In cases where it
is easier to invalidate all items of a given kind, the value zero is
used as a wildcard ID.

Each time that any GAE instance starts to service a new incoming
request, it first checks for any new entries in the `Invalidate`
table.  For up to 1000 rows, the `CacheManager` drops items from all
registered RAM caches that match that kind and ID.  Request processing
then proceeds, and any attempts to retrieve stale business objects
will cause cache misses that are then loaded from the DB.

Always adding rows to the `Invalidate` table would eventually make
that table huge.  So, Monorail uses a cron task to periodically drop
old entries in the `Invalidate` table.  Only the most recent 1000
entries are kept.  If a GAE instance checks the `Invalidate` table and
finds that there are 1000 or more new entries since the list time it
checked, the instance will flush all of its RAM caches.

Invalidating local caches at the start of each request does not handle
the situation where one GAE instance is handling a long-running
request and another GAE instance updates business objects at the same
time.  The long-running request may have retrieved and cached some
business objects early in the request processing, and then use a stale
cached copy of one of those same business objects later, after the
underlying DB data has changed.  To avoid this, services-level code
that updates business objects specifies the keyword use_cache=False to
retrieve a fresh copy of the object for each read-modify-write
operation.  As a further protection, the Issue protocol buffer has an
`assume_stale` boolean that helps check that issues from the cache are
not written back to the database.

## Detailed design: Read-only mode

Monorail has a couple of different levels of read-only modes.  The
entire site can be put into a read-only mode for maintenance by
specifying `read_only=True` in `settings.py`.  Also, individual
projects can be put into read-only mode by setting the
`read_only_reason` field on the project business object.

Read-only projects are a vestigial code.google.com feature that is not
currently exposed in any administrative UI.  It is implemented by
passing an EZT variable which causes `read-only-rejection.ezt` to be
shown to the user instead of the normal page content.  This UI-level
condition does not prevent API users from performing updates to the
project.  In fact, even users who have existing pages open can submit
forms to produce updates.

The site-wide read-only mode is implemented in `registerpages.py` to
not register POST handlers when the site is in read-only mode for
maintenance.  Also, in both the Endpoints and pRPC APIs there are
checks that reject requests that make updates during read-only mode.

## Detailed design: Connection pooling and the avoid list

It is faster to use an existing SQL connection object than to create a
new one.  So, `sql.py` implements a `ConnectionPool` class to keep SQL
connection objects until they are needed again.  MySQL uses implicit
transactions, so any connection keeps reading data as it existed at
the time of the last commit on that connection.  To get fresh data, we
do an empty commit on each connection at the time that we take it from
the pool.  To ensure that that commit is really empty, we roll back
any uncommitted updates for any request that had an exception.

A `MonorailConnection` is a collection of SQL connections with one for
the master DB and one for each replica that is used during the current
request.  Creating a connection to a given replica can fail if that
replica is offline.  Replicas can be restarted by the Google Cloud SQL
service at any time, e.g., to update DB server software.  When
Monorail fails to create a connection to a replica, it will simply use
a different replica instead.  However, the process of trying to
connect can be slow.  So, Monorail implements a dictionary with the
shard IDs of any unavailable replicas and the timestamp of the most
recent failure.  Monorail will avoid an unavailable replica for 45
seconds, giving it time to restart.

## Detailed design: Search result caches

This is described in [how-search-works.md](how-search-works.md).

## Detailed design: Attachments

Monorail's SQL database contains rows for issue attachments that
contain information about the attachment, but not the attachment
content.

Attachment content is stored in Google Cloud Storage.  Each attachment
is given a path of the form `/BUCKET/PROJECT_ID/attachments/UUID`
where UUID is a string with random hexadecimal digits generated by
python's [uuid
library](https://docs.python.org/2.7/library/uuid.html).  Each GCS
object has options specified which includes a `Content-Disposition`
header value with the desired filename.  We use the name of the
uploaded file in cases where it is known to be safe to keep it,
otherwise we use `attachment.dat`.

The MIME type of each attachment is determined from the file name and
file contents.  If we cannot determine a well-known file type,
`application/octet-stream` is used instead.

For image attachments, we generate a thumbnail-sized version of the
image using GAE's
[image](https://cloud.google.com/appengine/docs/standard/python/images/)
library at upload time.

Attachments are served to users directly from GCS via a signed link.
The risk of malicious attachment content is reduced by using a
different "locked domain" for each attachment link.  This prevents any
Javascript in an attachment from accessing cookies intended for our
application or any other website or even another attachment.

## Detailed design: Secrets

Monorail stores some small pieces of data in Datastore rather than
Google Cloud Storage.  This data includes the secret keys used to
generate XSRF tokens and authentication headers for reply emails.
These keys will never be a valid part of any SQL data export, so they
would need to be excluded from access granted to any account used for
SQL data export.  Rather than take on that complexity, we used
Datastore instead, and we do not grant access for anyone outside the
Monorail team to access the project's Datastore entities.

## Detailed design: Source code locations
*  `framework/sql.py`: Table managers, connection pools, and other utilities.
*  `framework/filecontent.py`: Functions to determine file types for
    attachments.
*  `framework/gcs_helpers.py`: Functions to write attachments into Google
    Cloud Storage.
*  `services/caches.py`: Base classes for caches.
*  `services/cachemanager.py`: Code for distributed invalidation and cron job
    for the `Invalidate` table.
*  `services/secrets_svc.py`: Code to get secret keys from Datastore.
*  `services/*.py`: Persistence code for business objects.
*  `settings.py`: Configuration of DB replicas and read_only mode.
