// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testvariantbqexporter

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/spanner"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/googleapi"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/bqutil"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/pbutil"
	bqpb "infra/appengine/weetbix/proto/bq"
	pb "infra/appengine/weetbix/proto/v1"
)

func testVariantName(realm, testID, variantHash string) string {
	return fmt.Sprintf("realms/%s/tests/%s/variants/%s", realm, url.PathEscape(testID), variantHash)
}

// generateStatement generates a spanner statement from a text template.
func generateStatement(tmpl *template.Template, input interface{}) (spanner.Statement, error) {
	sql := &bytes.Buffer{}
	err := tmpl.Execute(sql, input)
	if err != nil {
		return spanner.Statement{}, err
	}
	return spanner.NewStatement(sql.String()), nil
}

func (b *BQExporter) populateQueryParameters() (inputs, params map[string]interface{}, err error) {
	inputs = map[string]interface{}{
		"TestIdFilter": b.options.Predicate.TestIdRegexp != "",
		"StatusFilter": b.options.Predicate.Status != pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED,
	}

	params = map[string]interface{}{
		"realm":              b.options.Realm,
		"flakyVerdictStatus": int(pb.VerdictStatus_VERDICT_FLAKY),
	}

	st, err := pbutil.AsTime(b.options.TimeRange.GetEarliest())
	if err != nil {
		return nil, nil, err
	}
	params["startTime"] = st

	et, err := pbutil.AsTime(b.options.TimeRange.GetLatest())
	if err != nil {
		return nil, nil, err
	}
	params["endTime"] = et

	if re := b.options.Predicate.GetTestIdRegexp(); re != "" && re != ".*" {
		params["testIdRegexp"] = fmt.Sprintf("^%s$", re)
	}

	if status := b.options.Predicate.GetStatus(); status != pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED {
		params["status"] = int(status)
	}

	switch p := b.options.Predicate.GetVariant().GetPredicate().(type) {
	case *pb.VariantPredicate_Equals:
		inputs["VariantHashEquals"] = true
		params["variantHashEquals"] = pbutil.VariantHash(p.Equals)
	case *pb.VariantPredicate_Contains:
		if len(p.Contains.Def) > 0 {
			inputs["VariantContains"] = true
			params["variantContains"] = pbutil.VariantToStrings(p.Contains)
		}
	case nil:
		// No filter.
	default:
		return nil, nil, errors.Reason("unexpected variant predicate %q", p).Err()
	}
	return
}

type result struct {
	UnexpectedResultCount spanner.NullInt64
	TotalResultCount      spanner.NullInt64
	FlakyVerdictCount     spanner.NullInt64
	TotalVerdictCount     spanner.NullInt64
	Invocations           []string
}

func (b *BQExporter) populateTimeRange(tv *bqpb.TestVariantRow, statusUpdateTime spanner.NullTime) {
	tv.TimeRange = b.options.TimeRange
	tv.PartitionTime = tv.TimeRange.Latest
}

func (b *BQExporter) populateVerdicts(tv *bqpb.TestVariantRow, vs []string) error {
	verdicts := make([]*bqpb.Verdict, 0, len(vs))
	for _, v := range vs {
		parts := strings.Split(v, "/")
		if len(parts) != 3 {
			return fmt.Errorf("verdict %s in wrong format", v)
		}
		verdict := &bqpb.Verdict{
			Invocation: parts[0],
		}
		s, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}
		verdict.Status = pb.VerdictStatus(s).String()

		t, err := time.Parse(time.RFC3339Nano, parts[2])
		if err != nil {
			return err
		}
		verdict.CreateTime = timestamppb.New(t)
		verdicts = append(verdicts, verdict)
	}
	tv.Verdicts = verdicts
	return nil
}

func (b *BQExporter) populateFlakeStatistics(tv *bqpb.TestVariantRow, res *result) {
	zero64 := int64(0)
	if res.TotalVerdictCount.Valid && res.TotalVerdictCount.Int64 == zero64 {
		tv.FlakeStatistics = &pb.FlakeStatistics{
			FlakyVerdictCount:     0,
			TotalVerdictCount:     0,
			FlakyVerdictRate:      float32(0),
			UnexpectedResultCount: 0,
			TotalResultCount:      0,
			UnexpectedResultRate:  float32(0),
		}
		return
	}
	tv.FlakeStatistics = &pb.FlakeStatistics{
		FlakyVerdictCount:     res.FlakyVerdictCount.Int64,
		TotalVerdictCount:     res.TotalVerdictCount.Int64,
		FlakyVerdictRate:      float32(res.FlakyVerdictCount.Int64) / float32(res.TotalVerdictCount.Int64),
		UnexpectedResultCount: res.UnexpectedResultCount.Int64,
		TotalResultCount:      res.TotalResultCount.Int64,
		UnexpectedResultRate:  float32(res.UnexpectedResultCount.Int64) / float32(res.TotalResultCount.Int64),
	}
}

func (b *BQExporter) generateTestVariantRow(row *spanner.Row, bf spanutil.Buffer) (*bqpb.TestVariantRow, error) {
	tv := &bqpb.TestVariantRow{}
	va := &pb.Variant{}
	var vs []*result
	var statusUpdateTime spanner.NullTime
	var tmd spanutil.Compressed
	var status pb.AnalyzedTestVariantStatus
	if err := bf.FromSpanner(
		row,
		&tv.Realm,
		&tv.TestId,
		&tv.VariantHash,
		&va,
		&tv.Tags,
		&tmd,
		&status,
		&statusUpdateTime,
		&vs,
	); err != nil {
		return nil, err
	}

	tv.Name = testVariantName(tv.Realm, tv.TestId, tv.VariantHash)
	if len(vs) != 1 {
		return nil, fmt.Errorf("fail to get verdicts for test variant %s", tv.Name)
	}

	tv.Variant = pbutil.VariantToStringPairs(va)
	tv.Status = status.String()

	if len(tmd) > 0 {
		tv.TestMetadata = &pb.TestMetadata{}
		if err := proto.Unmarshal(tmd, tv.TestMetadata); err != nil {
			return nil, errors.Annotate(err, "error unmarshalling test_metadata for %s", tv.Name).Err()
		}
	}

	b.populateTimeRange(tv, statusUpdateTime)
	b.populateFlakeStatistics(tv, vs[0])
	if err := b.populateVerdicts(tv, vs[0].Invocations); err != nil {
		return nil, err
	}
	return tv, nil
}

func (b *BQExporter) query(ctx context.Context, f func(*bqpb.TestVariantRow) error) error {
	inputs, params, err := b.populateQueryParameters()
	if err != nil {
		return err
	}
	st, err := generateStatement(testVariantRowsTmpl, inputs)
	if err != nil {
		return err
	}
	st.Params = params

	var bf spanutil.Buffer
	return span.Query(ctx, st).Do(
		func(row *spanner.Row) error {
			tvr, err := b.generateTestVariantRow(row, bf)
			if err != nil {
				return err
			}
			return f(tvr)
		},
	)
}

func (b *BQExporter) queryTestVariantsToExport(ctx context.Context, batchC chan []*bqpb.TestVariantRow) error {
	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	tvrs := make([]*bqpb.TestVariantRow, 0, maxBatchRowCount)
	rowCount := 0
	err := b.query(ctx, func(tvr *bqpb.TestVariantRow) error {
		tvrs = append(tvrs, tvr)
		rowCount++
		if len(tvrs) >= maxBatchRowCount {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case batchC <- tvrs:
			}
			tvrs = make([]*bqpb.TestVariantRow, 0, maxBatchRowCount)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(tvrs) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batchC <- tvrs:
		}
	}

	logging.Infof(ctx, "fetched %d rows for exporting %s test variants", rowCount, b.options.Realm)
	return nil
}

// inserter is implemented by bigquery.Inserter.
type inserter interface {
	// PutWithRetries uploads one or more rows to the BigQuery service.
	PutWithRetries(ctx context.Context, src []*bq.Row) error
}

func hasReason(apiErr *googleapi.Error, reason string) bool {
	for _, e := range apiErr.Errors {
		if e.Reason == reason {
			return true
		}
	}
	return false
}

func (b *BQExporter) batchExportRows(ctx context.Context, ins inserter, batchC chan []*bqpb.TestVariantRow) error {
	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	for rows := range batchC {
		rows := rows
		if err := b.batchSem.Acquire(ctx, 1); err != nil {
			return err
		}

		eg.Go(func() error {
			defer b.batchSem.Release(1)
			err := b.insertRows(ctx, ins, rows)
			if bqutil.FatalError(err) {
				err = tq.Fatal.Apply(err)
			}
			return err
		})
	}

	return eg.Wait()
}

// insertRows inserts rows into BigQuery.
// Retries on transient errors.
func (b *BQExporter) insertRows(ctx context.Context, ins inserter, rowProtos []*bqpb.TestVariantRow) error {
	if err := b.putLimiter.Wait(ctx); err != nil {
		return err
	}

	rows := make([]*bq.Row, 0, len(rowProtos))
	for _, ri := range rowProtos {
		row := &bq.Row{
			Message:  ri,
			InsertID: bigquery.NoDedupeID,
		}
		rows = append(rows, row)
	}

	return ins.PutWithRetries(ctx, rows)
}

func (b *BQExporter) exportTestVariantRows(ctx context.Context, ins inserter) error {
	batchC := make(chan []*bqpb.TestVariantRow)
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return b.batchExportRows(ctx, ins, batchC)
	})

	eg.Go(func() error {
		defer close(batchC)
		return b.queryTestVariantsToExport(ctx, batchC)
	})

	return eg.Wait()
}

var testVariantRowsTmpl = template.Must(template.New("testVariantRowsTmpl").Parse(`
	@{USE_ADDITIONAL_PARALLELISM=TRUE}
	WITH test_variants AS (
		SELECT
			Realm,
			TestId,
			VariantHash,
		FROM AnalyzedTestVariants
		WHERE Realm = @realm
		{{if .StatusFilter}}
			AND Status = @status
    {{end}}
		{{/* Filter by TestId */}}
		{{if .TestIdFilter}}
			AND REGEXP_CONTAINS(TestId, @testIdRegexp)
		{{end}}
		{{/* Filter by Variant */}}
		{{if .VariantHashEquals}}
			AND VariantHash = @variantHashEquals
		{{end}}
		{{if .VariantContains }}
			AND (SELECT LOGICAL_AND(kv IN UNNEST(Variant)) FROM UNNEST(@variantContains) kv)
		{{end}}
		And StatusUpdateTime < @endTime
	)

	SELECT
		Realm,
		TestId,
		VariantHash,
		Variant,
		Tags,
		TestMetadata,
		Status,
		StatusUpdateTime,
		ARRAY(
		SELECT
			AS STRUCT SUM(UnexpectedResultCount) UnexpectedResultCount,
			SUM(TotalResultCount) TotalResultCount,
			COUNTIF(Status=30) FlakyVerdictCount,
			COUNT(*) TotalVerdictCount,
			-- Using struct here will trigger the "null-valued array of struct" query shape
			-- which is not supported by Spanner.
			-- Use a string to work around it.
			ARRAY_AGG(FORMAT('%s/%d/%s', InvocationId, Status, FORMAT_TIMESTAMP("%FT%H:%M:%E*S%Ez", InvocationCreationTime))) Invocations
		FROM
			Verdicts
		WHERE
			Verdicts.Realm = test_variants.Realm
			AND Verdicts.TestId=test_variants.TestId
			AND Verdicts.VariantHash=test_variants.VariantHash
			AND IngestionTime >= @startTime
			AND IngestionTime < @endTime ) Results
	FROM
		test_variants
	JOIN
		AnalyzedTestVariants
	USING
		(Realm,
			TestId,
			VariantHash)
`))
