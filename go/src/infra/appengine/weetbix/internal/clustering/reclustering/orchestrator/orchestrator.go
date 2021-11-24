// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package orchestrator

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/server/span"

	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/runs"
	"infra/appengine/weetbix/internal/clustering/state"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/services/reclustering"
	"infra/appengine/weetbix/internal/tasks/taskspb"
)

var (
	orchestratorCounter = metric.NewCounter(
		"weetbix/reclustering/orchestrator",
		"The status of the Weetbix reclustering orchestrator by LUCI project.",
		nil,
		// The LUCI project.
		field.String("project"),
		// The status of the orchestrator run. This covers only the
		// orchestrator and not the success/failure of workers.
		// "disabled", "success", "transient-failure" or "permanent-failure".
		field.String("status"),
		// The number of workers allocated to recluster the project.
		field.Int("workers"),
		// The progress achieved by the previous orchestrator run. Value
		// out of 1000. If this is less than 1000 for a while, this may
		// be an indication that re-clustering is slow and falling behind.
		field.Int("progress"))
)

// CronHandler is the entry-point to the orchestrator that creates
// reclustering jobs. It is triggered by a cron job configured in
// cron.yaml.
func CronHandler(ctx context.Context) error {
	err := orchestrate(ctx)
	if err != nil {
		logging.Errorf(ctx, "Reclustering orchestrator encountered errors: ", err)
		return err
	}
	return nil
}

func orchestrate(ctx context.Context) error {
	projectCfg, err := config.Projects(ctx)
	if err != nil {
		return errors.Annotate(err, "get projects config").Err()
	}
	var projects []string
	for project := range projectCfg {
		projects = append(projects, project)
	}
	// The order of projects affects worker allocations if projects
	// are entitled to fractional workers. Ensure the project order
	// is stable to keep orchestrator behaviour as stable as possible.
	sort.Strings(projects)

	cfg, err := config.Get(ctx)
	if err != nil {
		reportStatus(ctx, projects, "transient-failure")
		return errors.Annotate(err, "get service config").Err()
	}

	workers := int(cfg.ReclusteringWorkers)
	intervalMinutes := int(cfg.ReclusteringIntervalMinutes)
	if workers <= 0 {
		reportStatus(ctx, projects, "disabled")
		logging.Warningf(ctx, "Reclustering is disabled because configured worker count is zero.")
		return nil
	}
	if intervalMinutes <= 0 {
		reportStatus(ctx, projects, "disabled")
		logging.Warningf(ctx, "Reclustering is disabled because configured reclustering interval is zero.")
		return nil
	}

	err = orchestrateWithOptions(ctx, projects, workers, intervalMinutes)
	return err
}

// reportStatus reports the given orchestrator run status as the metric value
// for all the specified projects.
func reportStatus(ctx context.Context, projects []string, status string) {
	for _, project := range projects {
		workers := 0
		progress := 0
		orchestratorCounter.Add(ctx, 1, project, status, workers, progress)
	}
}

func orchestrateWithOptions(ctx context.Context, projects []string, workers, intervalMins int) error {
	currentMinute := clock.Now(ctx).Truncate(time.Minute)
	intervalDuration := time.Duration(intervalMins) * time.Minute
	attemptStart := clock.Now(ctx).Truncate(intervalDuration)
	if attemptStart != currentMinute {
		logging.Infof(ctx, "Orchestrator ran, but determined the current run start %v"+
			" does not match the current minute %v.", attemptStart, currentMinute)
		return nil
	}
	attemptEnd := attemptStart.Add(intervalDuration)

	workerCounts, err := projectWorkerCounts(ctx, projects, workers)
	if err != nil {
		reportStatus(ctx, projects, "transient-failure")
		return err
	}

	var errs []error
	for _, project := range projects {
		projectWorkers := workerCounts[project]
		err := orchestrateProject(ctx, project, attemptStart, attemptEnd, projectWorkers)
		if err != nil {
			// If an error occurs with one project, capture it, but continue
			// to avoid impacting other projects.
			errs = append(errs, errors.Annotate(err, "project %s", project).Err())
		}
	}
	if len(errs) > 0 {
		return errors.NewMultiError(errs...)
	}
	return nil
}

// projectWorkerCounts distributes workers between LUCI projects.
// The workers are allocated proportionately to the number of chunks in each
// project, with a minimum of one worker per project.
func projectWorkerCounts(ctx context.Context, projects []string, workers int) (map[string]int, error) {
	chunksByProject := make(map[string]int64)
	var totalChunks int64

	txn, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	for _, project := range projects {
		estimate, err := state.EstimateChunks(txn, project)
		if err != nil {
			return nil, errors.Annotate(err, "estimating rows for project %s", project).Err()
		}
		chunksByProject[project] = int64(estimate)
		totalChunks += int64(estimate)
	}

	// Each project gets at least one worker. The rest can be divided up
	// according to project size.
	freeWorkers := workers - len(projects)
	if freeWorkers < 0 {
		return nil, errors.New("more projects configured than workers")
	}

	result := make(map[string]int)
	for _, project := range projects {
		var additionalWorkers int

		projectChunks := chunksByProject[project]
		additionalWorkers = int((projectChunks * int64(freeWorkers)) / totalChunks)

		totalChunks -= projectChunks
		freeWorkers -= additionalWorkers

		// Every project gets at least one worker, plus
		// a number of workers depending on it size.
		result[project] = 1 + additionalWorkers
	}
	return result, nil
}

// orchestrateProject starts a new reclustering run for the given project,
// with the specified start and end time, and number of workers.
func orchestrateProject(ctx context.Context, project string, attemptStart, attemptEnd time.Time, workers int) error {
	status := "transient-failure"
	progress := 0
	defer func() {
		orchestratorCounter.Add(ctx, 1, project, status, workers, progress)
	}()

	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		var err error
		progress, err = createProjectRun(ctx, project, attemptStart, attemptEnd, workers)
		return err
	})
	if err != nil {
		return errors.Annotate(err, "create run").Err()
	}
	err = scheduleWorkers(ctx, project, attemptEnd, workers)
	if err != nil {
		return errors.Annotate(err, "schedule workers").Err()
	}
	status = "success"
	return nil
}

// createProjectRun creates a new run entry for a project, returning whether
// the previous run achieved its re-clustering goal (and any errors).
func createProjectRun(ctx context.Context, project string, attemptStart, attemptEnd time.Time, workers int) (progress int, err error) {
	lastRun, err := runs.ReadLast(ctx, project)
	if err != nil {
		return 0, errors.Annotate(err, "read last run").Err()
	}
	// run.Progress is a value between 0 and 1000 * lastRun.ShardCount.
	progress = int(lastRun.Progress / lastRun.ShardCount)

	if lastRun.AttemptTimestamp.After(attemptStart) {
		return progress, errors.New("an attempt which overlaps the proposed attempt already exists")
	}
	newRun := &runs.ReclusteringRun{
		Project:          project,
		AttemptTimestamp: attemptEnd,
		ShardCount:       int64(workers),
		ShardsReported:   0,
		Progress:         0,
	}
	if progress == 1000 {
		rulesVersion, err := rules.ReadLastUpdated(ctx, project)
		if err != nil {
			return progress, err
		}
		newRun.RulesVersion = rulesVersion
		newRun.AlgorithmsVersion = algorithms.AlgorithmsVersion
	} else {
		// It is foreseeable that re-clustering rules could have changed
		// every time the orchestrator runs. If we update the rules and
		// algorithms version for each new run, we may be continuously
		// re-clustering chunks early on in the keyspace without ever
		// getting around to later chunks. To ensure progress, and ensure
		// that every chunk gets a fair slice of re-clustering resources,
		// keep the same re-clustering goals until the last run has completed.
		newRun.RulesVersion = lastRun.RulesVersion
		newRun.AlgorithmsVersion = lastRun.AlgorithmsVersion
	}
	err = runs.Create(ctx, newRun)
	if err != nil {
		return progress, errors.Annotate(err, "create new run").Err()
	}
	return progress, err
}

// scheduleWorkers creates reclustering tasks for the given project
// and attempt. Workers are each assigned an equally large slice
// of the keyspace to recluster.
func scheduleWorkers(ctx context.Context, project string, attemptEnd time.Time, count int) error {
	splits := workerSplits(count)
	for i := 0; i < count; i++ {
		start := splits[i]
		end := splits[i+1]
		task := &taskspb.ReclusterChunks{
			Project:      project,
			AttemptTime:  timestamppb.New(attemptEnd),
			StartChunkId: start,
			EndChunkId:   end,
		}
		title := fmt.Sprintf("%s-%s-shard-%v", project, attemptEnd.Format("20060102-150405"), (i + 1))
		err := reclustering.Schedule(ctx, title, task)
		if err != nil {
			return err
		}
	}
	return nil
}

// workerSplits divides the chunk ID key space evenly into the given
// number of partitions. count + 1 entries are returned; with the chunk ID
// range for each partition being the range between two adjacent entries,
// i.e. partition 0 is from result[0] (exclusive) to result[1] (inclusive),
// partition 1 is from result[1] to result[2], and so on.
func workerSplits(count int) []string {
	var result []string
	// "" indicates table start, which is the start of the first partition.
	result = append(result, "")

	var keyspaceSize big.Int
	// keyspaceSize = 1 << 128  (for 128-bits of keyspace).
	keyspaceSize.Lsh(big.NewInt(1), 128)

	for i := 0; i < count; i++ {
		// Identify the split point between two partitions.
		// split = keyspaceSize * (i + 1) / count
		var split big.Int
		split.Mul(&keyspaceSize, big.NewInt(int64(i+1)))
		split.Div(&split, big.NewInt(int64(count)))

		// Subtract one to adjust for the upper bound being inclusive
		// and not exclusive. (e.g. the last split should be (1 << 128) - 1,
		// which is fffffff .... ffffff in hexadecimal,  not (1 << 128),
		// which is a "1" with 32 zeroes in hexadecimal).
		split.Sub(&split, big.NewInt(1))

		// Convert the split to a hexadecimal string.
		key := split.Text(16)
		if len(key) < 32 {
			// Pad the value with "0"s to get to the 32 hexadecimal
			// character length of a chunk ID.
			key = strings.Repeat("0", 32-len(key)) + key
		}
		result = append(result, key)
	}
	return result
}
