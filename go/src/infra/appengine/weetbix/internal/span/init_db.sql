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

  -- Status of the flaky test variant, see AnalyzedTestVariantStatus.
  Status INT64 NOT NULL,
  -- Timestamp when the status field was last updated.
  StatusUpdateTime TIMESTAMP,

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
CREATE NULL_FILTERED INDEX AnalyzedTestVariantsPerBuilderAndStatus
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

  -- ID of the build invocation the results belong to.
  InvocationId STRING(MAX) NOT NULL,

  -- Flag indicates if the verdict belongs to a try build.
  IsPreSubmit BOOL,

  -- Flag indicates if the try build the verdict belongs to contributes to
  -- a CL's submission.
  -- Verdicts with HasContributedToClSubmission as False will be filtered out
  -- for deciding the test variant's status because they could be noises.
  -- This field is only meaningful for PreSubmit verdicts.
  HasContributedToClSubmission BOOL,

  -- Status of the results for the parent test variant in this verdict,
  -- See VerdictStatus.
  Status INT64 NOT NULL,

  -- Result counts in the verdict.
  -- Note that SKIP results are ignored in either of the counts.
  UnexpectedResultCount INT64,
  TotalResultCount INT64,

  --Creation time of the invocation containing this verdict.
  InvocationCreationTime TIMESTAMP NOT NULL,

  -- List of colon-separated key-value pairs, where key is the cluster algorithm
  -- and value is the cluster id.
  -- key can be repeated.
  -- The clusters the first test result of the verdict is in.
  -- Once the test result reaches its retention period in the clustering
  -- system, this will cease to be updated.
  Clusters ARRAY<STRING(MAX)>,

) PRIMARY KEY (Realm, TestId, VariantHash, InvocationId),
INTERLEAVE IN PARENT AnalyzedTestVariants ON DELETE CASCADE;

-- Used by finding most recent verdicts to calculate flakiness statistics.
CREATE NULL_FILTERED INDEX VerdictsByTInvocationCreationTime
 ON Verdicts (Realm, TestId, VariantHash, InvocationCreationTime DESC);
