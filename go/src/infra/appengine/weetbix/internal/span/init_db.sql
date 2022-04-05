-- Copyright 2021 The Chromium Authors. All rights reserved.
--
-- Use of this source code is governed by a BSD-style license that can be
-- found in the LICENSE file.

--------------------------------------------------------------------------------
-- This script initializes a Weetbix Spanner database.

-- Stores a test variant.
-- The test variant should be:
-- * currently flaky
-- * suspected of flakiness that needs to be verified
-- * flaky before but has been fixed, broken, disabled or removed
CREATE TABLE AnalyzedTestVariants (
  -- Security realm this test variant belongs to.
  Realm STRING(64) NOT NULL,

  -- Builder that the test variant runs on.
  -- It must have the same value as the builder variant.
  Builder STRING(MAX),

  -- Unique identifier of the test,
  -- see also luci.resultdb.v1.TestResult.test_id.
  TestId STRING(MAX) NOT NULL,

  -- key:value pairs to specify the way of running the test.
  -- See also luci.resultdb.v1.TestResult.variant.
  Variant ARRAY<STRING(MAX)>,

  -- A hex-encoded sha256 of concatenated "<key>:<value>\n" variant pairs.
  -- Combination of Realm, TestId and VariantHash can identify a test variant.
  VariantHash STRING(64) NOT NULL,

  -- Timestamp when the row of a test variant was created.
  CreateTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),

  -- Status of the analyzed test variant, see AnalyzedTestVariantStatus.
  Status INT64 NOT NULL,
  -- Timestamp when the status field was last updated.
  StatusUpdateTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  -- Timestamp the next UpdateTestVariant task is enqueued.
  -- This timestamp is used as a token to validate an UpdateTestVariant is
  -- expected. A task with unmatched token will be silently ignored.
  NextUpdateTaskEnqueueTime TIMESTAMP,
  -- Previous statuses of the analyzed test variant.
  -- If the test variant is a newly detected one, or its status has not changed
  -- at all, this field is empty.
  -- With PreviousStatusUpdateTimes, they are used when exporting test variants
  -- to BigQuery, to determine the time ranges of the rows that happened when
  -- the test variant's status changed.
  PreviousStatuses ARRAY<INT64>,
  -- Previous status update times.
  -- Must have the same number of elements as PreviousStatuses.
  PreviousStatusUpdateTimes ARRAY<TIMESTAMP>,

  -- Compressed metadata for the test case.
  -- For example, the original test name, test location, etc.
  -- See TestResult.test_metadata for details.
  -- Test location is helpful for dashboards to get aggregated data by directories.
  TestMetadata BYTES(MAX),

  -- key:value pairs for the metadata of the test variant.
  -- For example the monorail component and team email.
  Tags ARRAY<STRING(MAX)>,

  -- Flake statistics, including flake rate, failure rate and counts.
  -- See FlakeStatistics proto.
  FlakeStatistics BYTES(MAX),
  -- Timestamp when the most recent flake statistics were computed.
  FlakeStatisticUpdateTime TIMESTAMP,
) PRIMARY KEY (Realm, TestId, VariantHash);

-- Used by finding test variants with FLAKY status on a builder in
-- CollectFlakeResults task.
CREATE NULL_FILTERED INDEX AnalyzedTestVariantsByBuilderAndStatus
ON AnalyzedTestVariants (Realm, Builder, Status);

-- Stores results of a test variant in one invocation.
CREATE TABLE Verdicts (
  -- Primary Key of the parent AnalyzedTestVariants.
  -- Security realm this test variant belongs to.
  Realm STRING(64) NOT NULL,
  -- Unique identifier of the test,
  -- see also luci.resultdb.v1.TestResult.test_id.
  TestId STRING(MAX) NOT NULL,
  -- A hex-encoded sha256 of concatenated "<key>:<value>\n" variant pairs.
  -- Combination of Realm, TestId and VariantHash can identify a test variant.
  VariantHash STRING(64) NOT NULL,

  -- Id of the build invocation the results belong to.
  InvocationId STRING(MAX) NOT NULL,

  -- Flag indicates if the verdict belongs to a try build.
  IsPreSubmit BOOL,

  -- Flag indicates if the try build the verdict belongs to contributes to
  -- a CL's submission.
  -- Verdicts with HasContributedToClSubmission as False will be filtered out
  -- for deciding the test variant's status because they could be noises.
  -- This field is only meaningful for PreSubmit verdicts.
  HasContributedToClSubmission BOOL,

  -- If the unexpected results in the verdict are exonerated.
  Exonerated BOOL,

  -- Status of the results for the parent test variant in this verdict,
  -- See VerdictStatus.
  Status INT64 NOT NULL,

  -- Result counts in the verdict.
  -- Note that SKIP results are ignored in either of the counts.
  UnexpectedResultCount INT64,
  TotalResultCount INT64,

  --Creation time of the invocation containing this verdict.
  InvocationCreationTime TIMESTAMP NOT NULL,

  -- Ingestion time of the verdict.
  IngestionTime TIMESTAMP NOT NULL,

  -- List of colon-separated key-value pairs, where key is the cluster algorithm
  -- and value is the cluster id.
  -- key can be repeated.
  -- The clusters the first test result of the verdict is in.
  -- Once the test result reaches its retention period in the clustering
  -- system, this will cease to be updated.
  Clusters ARRAY<STRING(MAX)>,

) PRIMARY KEY (Realm, TestId, VariantHash, InvocationId),
INTERLEAVE IN PARENT AnalyzedTestVariants ON DELETE CASCADE;

-- Used by finding most recent verdicts of a test variant to calculate status.
CREATE NULL_FILTERED INDEX VerdictsByTestVariantAndIngestionTime
 ON Verdicts (Realm, TestId, VariantHash, IngestionTime DESC),
 INTERLEAVE IN AnalyzedTestVariants;

-- FailureAssociationRules associate failures with bugs. When a rule
-- is used to match incoming test failures, the resultant cluster is
-- known as a 'bug cluster' because the failures in it are associated
-- with a bug (via the failure association rule).
-- The ID of a bug cluster corresponding to a rule is
-- (Project, RuleBasedClusteringAlgorithm, RuleID), where
-- RuleBasedClusteringAlgorithm is the algorithm name of the algorithm
-- that clusters failures based on failure association rules (e.g.
-- 'rules-v1'), and (Project, RuleId) is the ID of the rule.
CREATE TABLE FailureAssociationRules (
  -- The LUCI Project this bug belongs to.
  Project STRING(40) NOT NULL,
  -- The unique identifier for the rule. This is a randomly generated
  -- 128-bit ID, encoded as 32 lowercase hexadecimal characters.
  RuleId STRING(32) NOT NULL,
  -- The rule predicate, defining which failures are being associated.
  RuleDefinition STRING(4096) NOT NULL,
  -- The time the rule was created.
  CreationTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  -- The user which created the rule. If this was auto-filed by Weetbix
  -- itself, this is the special value 'weetbix'. Otherwise, it is
  -- an email address.
  -- 320 is the maximum length of an email address (64 for local part,
  -- 1 for the '@', and 255 for the domain part).
  CreationUser STRING(320) NOT NULL,
  -- The last time the rule was updated.
  LastUpdated TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  -- The user which last updated this rule. If this was Weetbix itself,
  -- (e.g. in case of an auto-filed bug which was created and never
  -- modified) this is 'weetbix'. Otherwise, it is an email address.
  LastUpdatedUser STRING(320) NOT NULL,
  -- The time the rule was last updated in a way that caused the
  -- matched failures to change, i.e. because of a change to RuleDefinition
  -- or IsActive. (For comparison, updating BugID does NOT change
  -- the matched failures, so does NOT update this field.)
  -- When this value changes, it triggers re-clustering.
  -- Basis for RulesVersion on ClusteringState and ReclusteringRuns.
  PredicateLastUpdated TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  -- The bug the failures are associated with (part 1). This is the
  -- bug tracking system containing the bug the failures are associated
  -- with. The only supported values are 'monorail' and 'buganizer'.
  BugSystem STRING(16) NOT NULL,
  -- The bug the failures are associated with (part 2). This is the
  -- identifier of the bug the failures are associated with, as identified
  -- by the bug tracking system itself. For monorail, the scheme is
  -- {project}/{numeric_id}, for buganizer, the scheme is {numeric_id}.
  BugId STRING(255) NOT NULL,
  -- Whether the bug must still be updated by Weetbix, and whether failures
  -- should still be matched against this rule. The only allowed
  -- values are true or NULL (to indicate false). Only if the bug has
  -- been closed and no failures have been observed for a while should
  -- this be NULL. This makes it easy to retrofit a NULL_FILTERED index
  -- in future, if it is needed for performance.
  IsActive BOOL,
  -- Whether this rule should manage the priority and verified status
  -- of the associated bug based on the impact of the cluster defined
  -- by this rule.
  -- The only allowed values are true or NULL (to indicate false).
  IsManagingBug BOOL,
  -- The suggested cluster this failure association rule was created from
  -- (if any) (part 1).
  -- This is the algorithm component of the suggested cluster this rule
  -- was created from.
  -- Until re-clustering is complete (and the residual impact of the source
  -- cluster has reduced to zero), SourceClusterAlgorithm and SourceClusterId
  -- tell bug filing to ignore the source suggested cluster when
  -- determining whether new bugs need to be filed.
  SourceClusterAlgorithm STRING(32) NOT NULL,
  -- The suggested cluster this failure association rule was created from
  -- (if any) (part 2).
  -- This is the algorithm-specific ID component of the suggested cluster
  -- this rule was created from.
  SourceClusterId STRING(32) NOT NULL,
) PRIMARY KEY (Project, RuleId);

-- The failure association rules associated with a bug. This also
-- enforces the constraint that there is at most one rule per bug
-- per project.
CREATE UNIQUE INDEX FailureAssociationRuleByBugAndProject ON FailureAssociationRules(BugSystem, BugId, Project);

-- Enforces the constraint that only one rule may manage a given bug
-- at once.
-- This is required to ensure that automatic bug filing does not attempt to
-- take conflicting actions (i.e. simultaneously increase and decrease
-- priority) on the same bug, because of differing priorities set by
-- different rules.
CREATE UNIQUE NULL_FILTERED INDEX FailureAssociationRuleByManagedBug ON FailureAssociationRules(BugSystem, BugId, IsManagingBug);

-- Clustering state records the clustering state of failed test results, organised
-- by chunk.
CREATE TABLE ClusteringState (
  -- The LUCI Project the test results belong to.
  Project STRING(40) NOT NULL,
  -- The identity of the chunk of test results. 32 lowercase hexadecimal
  -- characters assigned by the ingestion process.
  ChunkId STRING(32) NOT NULL,
  -- The start of the retention period of the test results in the chunk.
  PartitionTime TIMESTAMP NOT NULL,
  -- The identity of the blob storing the chunk's test results.
  ObjectId STRING(32) NOT NULL,
  -- The version of clustering algorithms used to cluster test results in this
  -- chunk. (This is a version over the set of algorithms, distinct from the
  -- version of a single algorithm, e.g.:
  -- v1 -> {failurereason-v1}, v2 -> {failurereason-v1, testname-v1},
  -- v3 -> {failurereason-v2, testname-v1}.)
  AlgorithmsVersion INT64 NOT NULL,
  -- The version of project configuration used by algorithms to match test
  -- results in this chunk.
  ConfigVersion TIMESTAMP NOT NULL,
  -- The version of the set of failure association rules used to match test
  -- results in this chunk. This is the maximum "Predicate Last Updated" time
  -- of any failure association rule in the snapshot of failure association
  -- rules used to match the test results.
  RulesVersion TIMESTAMP NOT NULL,
  -- Serialized ChunkClusters proto containing which test result is in which
  -- cluster.
  Clusters BYTES(MAX) NOT NULL,
  -- The Spanner commit timestamp of when the row was last updated.
  LastUpdated TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (Project, ChunkId)
-- Commented out for Cloud Spanner Emulator:
-- https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/32
-- but **should** be applied to real Spanner instances.
--, ROW DELETION POLICY (OLDER_THAN(PartitionTime, INTERVAL 90 DAY));

-- ReclusteringRuns contains details of runs used to re-cluster test results.
CREATE TABLE ReclusteringRuns (
  -- The LUCI Project.
  Project STRING(40) NOT NULL,
  -- The attempt. This is the timestamp the orchestrator run ends.
  AttemptTimestamp TIMESTAMP NOT NULL,
  -- The minimum algorithms version the reclustering run is trying to achieve.
  -- Chunks with an AlgorithmsVersion less than this value are eligible to be
  -- re-clustered.
  AlgorithmsVersion INT64 NOT NULL,
  -- The minimum config version the reclustering run is trying to achieve.
  -- Chunks with a ConfigVersion less than this value are eligible to be
  -- re-clustered.
  ConfigVersion TIMESTAMP NOT NULL,
  -- The minimum rules version the reclustering run is trying to achieve.
  -- Chunks with a RulesVersion less than this value are eligible to be
  -- re-clustered.
  RulesVersion TIMESTAMP NOT NULL,
  -- The number of shards created for this run (for this LUCI project).
  ShardCount INT64 NOT NULL,
  -- The number of shards that have reported progress (at least once).
  -- When this is equal to ShardCount, readers can have confidence Progress
  -- is a reasonable reflection of the progress made reclustering
  -- this project. Until then, it is a loose lower-bound.
  ShardsReported INT64 NOT NULL,
  -- The progress. This is a value between 0 and 1000*ShardCount.
  Progress INT64 NOT NULL,
) PRIMARY KEY (Project, AttemptTimestamp DESC)
-- Commented out for Cloud Spanner Emulator:
-- https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/32
-- but **should** be applied to real Spanner instances.
--, ROW DELETION POLICY (OLDER_THAN(AttemptTimestamp, INTERVAL 90 DAY));

-- Ingestions is used to synchronise and deduplicate the ingestion
-- of test results which require data from one or more sources.
--
-- Ingestion may only start after two events are received:
-- 1. The build has completed.
-- 2. The presubmit run has completed.
-- These events may occur in either order (e.g. 2 can occur before 1 if the
-- presubmit run fails before all builds are complete).
CREATE TABLE Ingestions (
  -- The unique key for the ingestion. The current scheme is:
  -- {buildbucket host name}/{build id}.
  BuildId STRING(1024) NOT NULL,
  -- The LUCI Project to which the build belongs. Populated at the same
  -- time as the build result.
  BuildProject STRING(40),
  -- The build result.
  BuildResult BYTES(MAX),
  -- Whether the record has any build result.
  -- Used in index to speed-up to some statistical queries.
  HasBuildResult BOOL NOT NULL AS (BuildResult IS NOT NULL) STORED,
  -- The Spanner commit time the build result was populated.
  BuildJoinedTime TIMESTAMP OPTIONS (allow_commit_timestamp=true),
  -- Is the build part of a presubmit run? If yes, then ingestion should
  -- wait for the presubmit result to be populated before commencing ingestion.
  -- Use 'true' to indicate true and NULL to indicate false.
  IsPresubmit BOOL,
  -- The LUCI Project to which the presubmit run belongs. Populated at the
  -- same time as the presubmit run result.
  PresubmitProject STRING(40),
  -- The presubmit result.
  PresubmitResult BYTES(MAX),
  -- Whether the record has any presubmit result.
  -- Used in index to speed-up to some statistical queries.
  HasPresubmitResult BOOL NOT NULL AS (PresubmitResult IS NOT NULL) STORED,
  -- The Spanner commit time the presubmit result was populated.
  PresubmitJoinedTime TIMESTAMP OPTIONS (allow_commit_timestamp=true),
  -- The Spanner commit time the row last last updated.
  LastUpdated TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (BuildId)
-- 90 days retention, plus some margin (10 days) to ensure ingestion records
-- are always retained longer than the ingested results (acknowledging
-- the partition time on ingested chunks may be later than the LastUpdated
-- time if clocks are not synchronised).
--
-- Commented out for Cloud Spanner Emulator:
-- https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/32
-- but **should** be applied to real Spanner instances.
--, ROW DELETION POLICY (OLDER_THAN(LastUpdated, INTERVAL 100 DAY));

-- Used to speed-up querying join statistics for presubmit runs.
CREATE NULL_FILTERED INDEX IngestionsByIsPresubmit
  ON Ingestions(IsPresubmit, BuildId)
  STORING (BuildProject,     HasBuildResult,     BuildJoinedTime,
           PresubmitProject, HasPresubmitResult, PresubmitJoinedTime);

-- Stores transactional tasks reminders.
-- See https://go.chromium.org/luci/server/tq. Scanned by tq-sweeper-spanner.
CREATE TABLE TQReminders (
    ID STRING(MAX) NOT NULL,
    FreshUntil TIMESTAMP NOT NULL,
    Payload BYTES(102400) NOT NULL,
) PRIMARY KEY (ID ASC);

CREATE TABLE TQLeases (
    SectionID STRING(MAX) NOT NULL,
    LeaseID INT64 NOT NULL,
    SerializedParts ARRAY<STRING(MAX)>,
    ExpiresAt TIMESTAMP NOT NULL,
) PRIMARY KEY (SectionID ASC, LeaseID ASC);

-- Stores combined results of all runs of a test variant in a single invocation.
CREATE TABLE TestVerdicts (
  -- The LUCI Project this test verdict belongs to.
  Project STRING(40) NOT NULL,

  -- Unique identifier of the test.
  -- This has the same value as luci.resultdb.v1.TestResult.test_id.
  TestId STRING(MAX) NOT NULL,

  -- Partition time, as determined by Weetbix ingestion. Start time of the
  -- ingested build (for postsubmit results) or start time of the presubmit run
  -- (for presubmit results). Defines date/time axis of test verdicts plotted
  -- by date/time.
  -- Including as part of Primary Key allows direct filtering of data for test
  -- to last N days. This could be used to improve performance for tests with
  -- many verdicts, or allow experimentation with keeping longer histories
  -- (e.g. 120 days) without incurring performance penalty on time-windowed
  -- queries.
  PartitionTime TIMESTAMP NOT NULL,

  -- A hex-encoded sha256 of concatenated "<key>:<value>\n" variant pairs.
  -- Computed as hex(sha256(<concatenated_key_value_pairs>)[:8]),
  -- where concatenated_key_value_pairs is the result of concatenating
  -- variant pairs formatted as "<key>:<value>\n" in ascending key order.
  -- Combination of Realm, TestId and VariantHash can identify a test variant.
  VariantHash STRING(16) NOT NULL,

  -- The invocation from which these test results were ingested.
  -- This is the top-level invocation that was ingested.
  IngestedInvocationId STRING(MAX) NOT NULL,

  -- The realm of the test result, excluding project. 62 as ResultDB allows
  -- at most 64 characters for the construction "<project>:<realm>" and project
  -- must be at least one character.
  SubRealm STRING(62) NOT NULL,

  -- Derived information from the included test results.
  -- Needed to compute the status of the verdict.
  -- Status can be computed with the following code:
  -- ```
  -- WHEN IsExonerated THEN @exonerated
  -- WHEN UnexpectedCount = 0 THEN @expected
  -- WHEN SkippedCount = UnexpectedCount AND ExpectedCount = 0 THEN @unexpectedlySkipped
  -- WHEN ExpectedCount = 0 THEN @unexpected
  -- ELSE @flaky
  -- ```

  -- How many expected test results this verdict includes.
  ExpectedCount INT64 NOT NULL,
  -- How many unexpected test results this verdict includes.
  UnexpectedCount INT64 NOT NULL,
  -- How many skipped test results this verdict includes.
  SkippedCount INT64 NOT NULL,
  -- Whether the test verdict was exonerated.
  IsExonerated BOOL NOT NULL,

  -- How long a passed test execution took on average, in microseconds.
  PassedAvgDurationUsec INT64,

  -- Whether the invocation was part of a build that has unsubmitted changes
  -- applied (such as Gerrit changes). (This includes unsubmitted changes
  -- that were later submitted, e.g. because of a successful presubmit run.)
  HasUnsubmittedChanges BOOL NOT NULL,

  -- Whether the invocation was part of a build that has unsubmitted changes
  -- applied (such as Gerrit changes) AND the changes were later submitted
  -- because the build was part of a successful presubmit run.
  HasContributedToClSubmission BOOL NOT NULL,
) PRIMARY KEY(Project, TestId, PartitionTime, VariantHash, IngestedInvocationId, SubRealm)
-- The following DDL query needs to be uncommented when applied to real Spanner
-- instances. But it is commented out for Cloud Spanner Emulator:
-- https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/32
--, ROW DELETION POLICY (OLDER_THAN(PartitionTime, INTERVAL 90 DAY));

-- Stores top-level invocations which were ingested.
--
-- TODO(crbug.com/1266759):
-- This forms part of an experiment embedded into the design.
-- If joining to this table is efficient, we may leave IsPresumbmit,
-- realm, commit position data here and drop it off the TestVerdicts table.
-- If not, we may decide to delete this table.
CREATE TABLE IngestedInvocations (
  -- The LUCI Project the invocation is a part of.
  Project STRING(40) NOT NULL,

  -- The (top-level) invocation which was ingested.
  IngestedInvocationId STRING(MAX) NOT NULL,

  -- The realm of the invocation, excluding project. 62 as ResultDB allows
  -- at most 64 characters for the construction "<project>:<realm>" and project
  -- must be at least one character.
  SubRealm STRING(62) NOT NULL,

  -- Partition time, as determined by Weetbix ingestion. Start time of the
  -- ingested build (for postsubmit results) or start time of the presubmit run
  -- (for presubmit results).
  PartitionTime TIMESTAMP NOT NULL,

  -- Whether the invocation was part of a build that has unmerged changes
  -- applied (such as Gerrit changes). (This includes unsubmitted changes
  -- that were later submitted, e.g. because of a successful presubmit run.)
  HasUnmergedChanges BOOL NOT NULL,

  -- Whether the invocation was part of a build that has unsubmitted changes
  -- applied (such as Gerrit changes) AND the changes were later submitted
  -- because the build was part of a successful presubmit run.
  HasContributedToClSubmission BOOL NOT NULL,
) PRIMARY KEY(Project, IngestedInvocationId)
-- The following DDL query needs to be uncommented when applied to real Spanner
-- instances. But it is commented out for Cloud Spanner Emulator:
-- https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/32
-- Use a slightly longer retention period to prevent the invocation being
-- dropped before the associated TestVerdicts.
--, ROW DELETION POLICY (OLDER_THAN(PartitionTime, INTERVAL 100 DAY));

-- Serves three purposes:
-- - Permits listing of distinct tests observed for a project, filtered by Realm.
--   Tests may be duplicated many times in this table because there is one entry
--   per test variant and realm. If this creates performance issues we can
--   create a table with only distinct TestId entries at the cost of some
--   additional complexity.
--
-- - Permits listing of distinct variants observed for a test in a project,
--   filtered by Realm.
--
-- - Provides a mapping back from VariantHash to variant.
--
-- TODO(crbug.com/1266759):
-- UniqueTestVariants table in ResultDB will be superseded by this table and
-- will need to be deleted.
CREATE TABLE TestVariantRealms (
  -- The LUCI Project in which the variant was observed.
  Project STRING(40) NOT NULL,

  -- Unique identifier of the test from which the variant was observed,
  -- This has the same value as luci.resultdb.v1.TestResult.test_id.
  TestId STRING(MAX) NOT NULL,

  -- A hex-encoded sha256 of concatenated "<key>:<value>\n" variant pairs.
  -- Computed as hex(sha256(<concatenated_key_value_pairs>)[:8]),
  -- where concatenated_key_value_pairs is the result of concatenating
  -- variant pairs formatted as "<key>:<value>\n" in ascending key order.
  -- Combination of Realm, TestId and VariantHash can identify a test variant.
  VariantHash STRING(16) NOT NULL,

  -- The realm of the test result from which the variant was observed, excluding
  -- project. 62 as ResultDB allows at most 64 characters for the construction
  -- "<project>:<realm>" and project must be at least one character.
  SubRealm STRING(62) NOT NULL,

  -- key:value pairs to specify the way of running the test.
  -- See also luci.resultdb.v1.TestResult.variant.
  Variant ARRAY<STRING(MAX)>,

  -- Other information about the test variant, like information from tags,
  -- could be captured here, as is currently the case for AnalyzedTestVariants.
  -- (e.g. test ownership).

  -- Last (ingestion) time this test variant was observed in the realm.
  -- This value may be out of date by up to 24 hours to allow for contention-
  -- reducing strategies.
  LastIngestionTime TIMESTAMP NOT NULL,
) PRIMARY KEY(Project, TestId, VariantHash, SubRealm)
-- The following DDL query needs to be uncommented when applied to real Spanner
-- instances. But it is commented out for Cloud Spanner Emulator:
-- https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/32
--, ROW DELETION POLICY (OLDER_THAN(LastIngestionTime, INTERVAL 90 DAY));
