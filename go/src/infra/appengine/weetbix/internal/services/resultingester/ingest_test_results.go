// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"context"
	"fmt"
	"regexp"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/appstatus"
	rdbbutil "go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/analysis/clusteredfailures"
	"infra/appengine/weetbix/internal/buildbucket"
	"infra/appengine/weetbix/internal/clustering/chunkstore"
	"infra/appengine/weetbix/internal/clustering/ingestion"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/resultdb"
	"infra/appengine/weetbix/internal/services/resultcollector"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	pb "infra/appengine/weetbix/proto/v1"
)

const (
	resultIngestionTaskClass = "result-ingestion"
	resultIngestionQueue     = "result-ingestion"
)

// Options configures test result ingestion.
type Options struct {
}

type resultIngester struct {
	clustering *ingestion.Ingester
}

var resultIngestion = tq.RegisterTaskClass(tq.TaskClass{
	ID:        resultIngestionTaskClass,
	Prototype: &taskspb.IngestTestResults{},
	Queue:     resultIngestionQueue,
	Kind:      tq.NonTransactional,
})

// realmProjectRe extracts the LUCI project name from a LUCI Realm.
var realmProjectRe = regexp.MustCompile(`^([a-z0-9\-_]{1,40}):.+$`)

// RegisterTaskHandler registers the handler for result ingestion tasks.
func RegisterTaskHandler(srv *server.Server) error {
	ctx := srv.Context
	cfg, err := config.Get(ctx)
	if err != nil {
		return err
	}
	chunkStore, err := chunkstore.NewClient(ctx, cfg.ChunkGcsBucket)
	if err != nil {
		return err
	}
	srv.RegisterCleanup(func(ctx context.Context) {
		chunkStore.Close()
	})
	cf := clusteredfailures.NewClient(srv.Options.CloudProject)
	analysis := analysis.NewClusteringHandler(cf)
	ri := &resultIngester{
		clustering: ingestion.New(chunkStore, analysis),
	}
	handler := func(ctx context.Context, payload proto.Message) error {
		task := payload.(*taskspb.IngestTestResults)
		return ri.ingestTestResults(ctx, task)
	}
	resultIngestion.AttachHandler(handler)
	return nil
}

// Schedule enqueues a task to ingest test results from a build.
func Schedule(ctx context.Context, task *taskspb.IngestTestResults) error {
	// Note that currently we don't need to deduplicate tasks, because for
	// Chromium use case Weetbix only ingest test results of the try builds that
	// contribute to CL submission, so each build should be processed only once.
	// This may not be true in ChromeOS use case where Weetbix ingests test
	// of all try builds.
	return tq.AddTask(ctx, &tq.Task{
		Title:   fmt.Sprintf("%s-%d", task.Build.Host, task.Build.Id),
		Payload: task,
	})
}

func (i *resultIngester) ingestTestResults(ctx context.Context, payload *taskspb.IngestTestResults) error {
	if err := validateRequest(payload); err != nil {
		return err
	}
	b, err := getBuilderAndResultDBInfo(ctx, payload)
	st, ok := appstatus.Get(err)
	if ok && st.Code() == codes.NotFound {
		// Build not found, end the task gracefully.
		logging.Warningf(ctx, "Buildbucket build %d not found (or Weetbix does not have access to read it).", payload.Build.Id)
		return nil
	}
	if err != nil {
		return err
	}

	rdbHost := b.Infra.Resultdb.Hostname
	invName := b.Infra.Resultdb.Invocation
	builder := b.Builder.Builder
	rc, err := resultdb.NewClient(ctx, rdbHost)
	if err != nil {
		return err
	}
	inv, err := rc.GetInvocation(ctx, invName)
	if err != nil {
		return err
	}
	project := projectFromRealm(inv.Realm)
	if project == "" {
		return fmt.Errorf("invocation has invalid realm: %q", inv.Realm)
	}

	// Setup clustering ingestion.
	invID, err := rdbbutil.ParseInvocationName(invName)
	opts := ingestion.Options{
		Project:       project,
		InvocationID:  invID,
		PartitionTime: payload.PartitionTime.AsTime(),
		Realm:         inv.Realm,
	}
	if payload.CvRun != nil {
		opts.PresubmitRunID = &pb.PresubmitRunId{System: "luci-cv", Id: payload.CvRun.Id}
	}
	clusterIngestion := i.clustering.Open(opts)

	// Query test variants from ResultDB and save/update the corresponding
	// AnalyzedTestVariant rows.
	// We read test variants from ResultDB in pages, and the func will be called
	// once per page of test variants.
	err = rc.QueryTestVariants(ctx, invName, func(tvs []*rdbpb.TestVariant) error {
		if shouldIngestForTestVariants(payload) {
			if err := createOrUpdateAnalyzedTestVariants(ctx, inv.Realm, builder, tvs); err != nil {
				return errors.Annotate(err, "ingesting for test variant analysis").Err()
			}
		}
		// Clustering ingestion is designed to behave gracefully in case of
		// a task retry. Given the same options and same test variants (in
		// the same order), the IDs and content of the chunks it writes is
		// designed to be stable. If chunks already exist, it will skip them.
		if err := clusterIngestion.Put(ctx, tvs); err != nil {
			return errors.Annotate(err, "ingesting for clustering").Err()
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := clusterIngestion.Flush(ctx); err != nil {
		return errors.Annotate(err, "ingesting for clustering").Err()
	}

	// Currently only Chromium CI results are ingested.
	isPreSubmit, contributedToCLSubmission := false, false
	if err = resultcollector.Schedule(ctx, inv, rdbHost, b.Builder.Builder, isPreSubmit, contributedToCLSubmission); err != nil {
		return err
	}

	return nil
}

func validateRequest(payload *taskspb.IngestTestResults) error {
	if payload.PartitionTime == nil {
		return errors.New("partition time must be specified")
	}
	return nil
}

func projectFromRealm(realm string) string {
	match := realmProjectRe.FindStringSubmatch(realm)
	if match != nil {
		return match[1]
	}
	return ""
}

func getBuilderAndResultDBInfo(ctx context.Context, payload *taskspb.IngestTestResults) (*bbpb.Build, error) {
	bbHost := payload.Build.Host
	bId := payload.Build.Id
	bc, err := buildbucket.NewClient(ctx, bbHost)
	if err != nil {
		return nil, err
	}
	b, err := bc.GetBuildWithBuilderAndRDBInfo(ctx, bId)
	switch {
	case err != nil:
		return nil, err
	case b.GetInfra().GetResultdb() == nil || b.Infra.Resultdb.GetInvocation() == "":
		return nil, tq.Fatal.Apply(errors.Reason("build %s-%d not have ResultDB invocation", bbHost, bId).Err())
	}
	return b, nil
}
