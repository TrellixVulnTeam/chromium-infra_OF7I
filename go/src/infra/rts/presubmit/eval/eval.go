// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"bytes"
	"container/heap"
	"context"
	"flag"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/rts"
	evalpb "infra/rts/presubmit/eval/proto"
)

const defaultConcurrency = 100

// Eval estimates safety and efficiency of a given selection strategy.
type Eval struct {
	// The number of goroutines to spawn for each metric.
	// If <=0, defaults to 100.
	Concurrency int

	// Rejections is a path to a directory with rejection records.
	// For format details, see comments of Rejection protobuf message.
	Rejections string

	// Durations is a path to a directory with test duration records.
	// For format details, see comments of TestDurationRecord protobuf message.
	Durations string

	// LogFurthest instructs to log rejections for which failed tests have large
	// distance, as concluded by the selection strategy.
	// LogFurthest is the number of rejections to print, ordered by descending
	// distance.
	// This can help diagnosing the selection strategy.
	//
	// TODO(nodir): implement this.
	LogFurthest int

	// LogProgressInterval indicates how often to log the number of processed
	// historical records. The field value is the number of records between
	// progress reports. If zero or less, progress is not logged.
	LogProgressInterval int
}

// RegisterFlags registers flags for the Eval fields.
func (e *Eval) RegisterFlags(fs *flag.FlagSet) error {
	fs.IntVar(&e.Concurrency, "j", defaultConcurrency, "Number of job to run parallel")
	fs.StringVar(&e.Rejections, "rejections", "", text.Doc(`
		Path to a directory with test rejection records.
		For format details, see comments of Rejection protobuf message.
		Used for safety evaluation.
	`))
	fs.StringVar(&e.Durations, "durations", "", text.Doc(`
		Path to a directory with test duration records.
		For format details, see comments of TestDurationRecord protobuf message.
		Used for efficiency evaluation.
	`))
	fs.IntVar(&e.LogFurthest, "log-furthest", 10, text.Doc(`
		Log rejections for which failed tests have large distance,
		as concluded by the selection strategy.
		The flag value is the number of rejections to print, ordered by descending
		distance.
		This can help diagnosing the selection strategy.
	`))
	return nil
}

// ValidateFlags validates values of flags registered using RegisterFlags.
func (e *Eval) ValidateFlags() error {
	if e.Rejections == "" {
		return errors.New("-rejections is required")
	}
	if e.Durations == "" {
		return errors.New("-durations is required")
	}
	return nil
}

// Run evaluates the candidate strategy.
func (e *Eval) Run(ctx context.Context, strategy Strategy) (*evalpb.Results, error) {
	logging.Infof(ctx, "Evaluating safety...")
	res, err := e.EvaluateSafety(ctx, strategy)
	if err != nil {
		return nil, errors.Annotate(err, "failed to evaluate safety").Err()
	}

	logging.Infof(ctx, "Evaluating efficiency...")
	if err := e.evaluateEfficiency(ctx, strategy, res); err != nil {
		return nil, errors.Annotate(err, "failed to evaluate efficiency").Err()
	}
	return res, nil
}

// EvaluateSafety evaluates the strategy's safety.
// The returned Result has all but efficiency-related fields populated.
func (e *Eval) EvaluateSafety(ctx context.Context, strategy Strategy) (*evalpb.Results, error) {
	var changeAffectedness []rts.Affectedness
	var testAffectedness []rts.Affectedness
	furthest := make(furthestRejections, 0, e.LogFurthest)
	maxNonInf := 0.0
	var mu sync.Mutex

	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	// Play back the history.
	rejC := make(chan *evalpb.Rejection)
	eg.Go(func() error {
		defer close(rejC)
		err := readRejections(ctx, e.Rejections, rejC)
		return errors.Annotate(err, "failed to read rejection records").Err()
	})

	res := &evalpb.Results{}
	e.goMany(eg, func() error {
		for rej := range rejC {
			// TODO(crbug.com/1112125): skip the patchset if it has a ton of failed tests.
			// Most selection strategies would reject such a patchset, so it represents noise.

			// Invoke the strategy.
			in := Input{TestVariants: rej.FailedTestVariants}
			in.ensureChangedFilesInclude(rej.Patchsets...)
			out := &Output{TestVariantAffectedness: make([]rts.Affectedness, len(in.TestVariants))}
			if err := strategy(ctx, in, out); err != nil {
				return errors.Annotate(err, "the selection strategy failed").Err()
			}

			// The affectedness of a change is based on the most affected failed test.
			mostAffected, err := mostAffected(out.TestVariantAffectedness)
			if err != nil {
				return err
			}

			mu.Lock()
			changeAffectedness = append(changeAffectedness, mostAffected)
			testAffectedness = append(testAffectedness, out.TestVariantAffectedness...)
			furthest.Consider(affectedRejection{Rejection: rej, MostAffected: mostAffected})
			if !math.IsInf(mostAffected.Distance, 1) && maxNonInf < mostAffected.Distance {
				maxNonInf = mostAffected.Distance
			}
			if e.LogProgressInterval > 0 && len(changeAffectedness)%e.LogProgressInterval == 0 {
				logging.Infof(ctx, "processed %d rejections", len(changeAffectedness))
			}
			mu.Unlock()
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if len(changeAffectedness) == 0 {
		return nil, errors.New("no change rejections")
	}

	if len(furthest) > 0 {
		furthest.LogAndClear(ctx)
	}

	// Compute distance thresholds by taking their percentiles in
	// changeAffectedness. Element indexes represent ChangeRecall scores.
	res.RejectionClosestDistanceStats = &evalpb.DistanceStats{
		Percentiles: distanceQuantiles(changeAffectedness, 100),
		MaxNonInf:   float32(maxNonInf),
	}
	logging.Infof(ctx, "Distance percentiles: %v", res.RejectionClosestDistanceStats.Percentiles)
	logging.Infof(ctx, "Maximum non-inf distance: %f", res.RejectionClosestDistanceStats.MaxNonInf)
	res.Thresholds = make([]*evalpb.Threshold, len(res.RejectionClosestDistanceStats.Percentiles))
	for i, distance := range res.RejectionClosestDistanceStats.Percentiles {
		res.Thresholds[i] = &evalpb.Threshold{MaxDistance: float32(distance)}
	}

	// Now compute recall scores off of the chosen thresholds.

	losses := func(afs []rts.Affectedness) bucketSlice {
		buckets := make(bucketSlice, len(res.Thresholds)+1)
		for _, af := range afs {
			buckets.inc(res.Thresholds, af, 1)
		}
		buckets.makeCumulative()
		return buckets
	}

	res.TotalRejections = int64(len(changeAffectedness))
	lostRejections := losses(changeAffectedness)

	res.TotalTestFailures = int64(len(testAffectedness))
	lostFailures := losses(testAffectedness)

	for i, t := range res.Thresholds {
		t.PreservedRejections = int64(res.TotalRejections) - int64(lostRejections[i+1])
		t.PreservedTestFailures = int64(res.TotalTestFailures) - int64(lostFailures[i+1])
		t.ChangeRecall = float32(t.PreservedRejections) / float32(res.TotalRejections)
		t.TestRecall = float32(t.PreservedTestFailures) / float32(res.TotalTestFailures)
	}
	return res, nil
}

// evaluateEfficiency computes total and saved durations.
func (e *Eval) evaluateEfficiency(ctx context.Context, strategy Strategy, res *evalpb.Results) error {
	// Process test durations in parallel and increment appropriate counters.
	savedDurations := make(bucketSlice, len(res.Thresholds)+1)
	var totalDuration int64

	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	// Play back the history.
	recordC := make(chan *evalpb.TestDurationRecord)
	eg.Go(func() error {
		defer close(recordC)
		err := readTestDurations(ctx, e.Durations, recordC)
		return errors.Annotate(err, "failed to read test duration records").Err()
	})

	records := int64(0)
	e.goMany(eg, func() error {
		in := Input{}
		out := &Output{}
		for rec := range recordC {
			// Invoke the strategy.
			if cap(in.TestVariants) < len(rec.TestDurations) {
				in.TestVariants = make([]*evalpb.TestVariant, len(rec.TestDurations))
			}
			in.TestVariants = in.TestVariants[:len(rec.TestDurations)]
			for i, td := range rec.TestDurations {
				in.TestVariants[i] = td.TestVariant
			}
			in.ChangedFiles = in.ChangedFiles[:0]
			in.ensureChangedFilesInclude(rec.Patchsets...)

			out.TestVariantAffectedness = make([]rts.Affectedness, len(in.TestVariants))
			if err := strategy(ctx, in, out); err != nil {
				return errors.Annotate(err, "the selection strategy failed").Err()
			}

			// Record results.
			durSum := int64(0)
			for i, td := range rec.TestDurations {
				dur := int64(td.Duration.AsDuration())
				durSum += dur
				savedDurations.inc(res.Thresholds, out.TestVariantAffectedness[i], dur)
			}
			atomic.AddInt64(&totalDuration, durSum)

			if count := atomic.AddInt64(&records, 1); e.LogProgressInterval > 0 && int(count)%e.LogProgressInterval == 0 {
				logging.Infof(ctx, "processed %d test duration records", count)
			}
		}
		return ctx.Err()
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	if totalDuration == 0 {
		return errors.New("sum of test durations is 0")
	}

	// Incroporate the counters into res.

	res.TotalDuration = durationpb.New(time.Duration(totalDuration))
	savedDurations.makeCumulative()
	for i, t := range res.Thresholds {
		t.SavedDuration = durationpb.New(time.Duration(savedDurations[i+1]))
		t.Savings = float32(float64(savedDurations[i+1]) / float64(totalDuration))
	}
	return nil
}

func (e *Eval) goMany(eg *errgroup.Group, f func() error) {
	concurrency := e.Concurrency
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}
	for i := 0; i < concurrency; i++ {
		eg.Go(func() error {
			return f()
		})
	}
}

// distanceQuantiles returns distance quantiles. Panics if afs is empty.
func distanceQuantiles(afs []rts.Affectedness, count int) (distances []float32) {
	if len(afs) == 0 {
		panic("s is empty")
	}
	allDistances := make([]float64, len(afs))
	for i, af := range afs {
		allDistances[i] = af.Distance
	}
	sort.Float64s(allDistances)
	distances = make([]float32, count)
	for i := 0; i < count; i++ {
		boundary := int(math.Ceil(float64(len(afs)*(i+1)) / float64(count)))
		distances[i] = float32(allDistances[boundary-1])
	}
	return
}

// bucketSlice is an auxulary data structure to compute cumulative counters.
// Each element contains the number of data points lost by the bucket that
// the element represents.
//
// bucketSlice is used in two phases:
//   1) For each data point, call inc().
//   2) Call makeCumulative() and incorporate bucketSlice into thresholds.
//
// The structure of bucketSlice is similar to []*Threshold used in
// evaluateSafety and evaluateEfficiency, except bucketSlice element i
// corresponds to threshold i-1. This is because the bucketSlice is padded with
// extra element 0 for data points that were not lost by any threshold.
type bucketSlice []int64

// inc increments the counter for the largest distance less than af.Distance,
// i.e. the largest thresholds that missed the data point.
//
// Goroutine-safe.
func (b bucketSlice) inc(ts []*evalpb.Threshold, af rts.Affectedness, delta int64) {
	if len(b) != len(ts)+1 {
		panic("wrong bucket slice length")
	}

	dist32 := float32(af.Distance)
	i := sort.Search(len(ts), func(i int) bool {
		return ts[i].MaxDistance >= dist32
	})

	// We need the *largest* threshold *not* satisfied by af, i.e. the preceding
	// index. Indexes in bucketSlice are already shifted by one, so use i as is.
	atomic.AddInt64(&b[i], delta)
}

// makeCumulative makes all counters cumulative.
// Not idempotent.
func (b bucketSlice) makeCumulative() {
	for i := len(b) - 2; i >= 0; i-- {
		b[i] += b[i+1]
	}
}

// mostAffected returns the most significant Affectedness by comparing distance.
func mostAffected(afs []rts.Affectedness) (rts.Affectedness, error) {
	if len(afs) == 0 {
		return rts.Affectedness{}, errors.New("empty")
	}
	most := afs[0]
	for _, af := range afs {
		if most.Distance > af.Distance {
			most = af
		}
	}
	return most, nil
}

type furthestRejections []affectedRejection
type affectedRejection struct {
	Rejection    *evalpb.Rejection
	MostAffected rts.Affectedness
}

// Consider pushes the item if there is unused capacity, or replaces the closest
// item in the heap if the former is further than the latter.
// Does nothing if cap(*f) == 0.
func (f *furthestRejections) Consider(item affectedRejection) {
	switch {
	case cap(*f) == 0:
		return

	// If the heap has a free slot, just add the rejection.
	case len(*f) < cap(*f):
		heap.Push(f, item)

	// Otherwise, if the rejection is further than heap's closest one, then
	// replace the latter.
	case (*f)[0].MostAffected.Distance < item.MostAffected.Distance:
		(*f)[0] = item
		heap.Fix(f, 0)
	}
}

func (f *furthestRejections) LogAndClear(ctx context.Context) {
	buf := &bytes.Buffer{}
	p := rejectionPrinter{printer: newPrinter(buf)}
	p.printf("%d furthest rejections:\n", len(*f))
	p.Level++
	for len(*f) > 0 {
		r := heap.Pop(f).(affectedRejection)
		p.rejection(r.Rejection, r.MostAffected)
	}
	p.Level--
	logging.Infof(ctx, "%s", buf.Bytes())
}

func (f furthestRejections) Len() int { return len(f) }
func (f furthestRejections) Less(i, j int) bool {
	return f[i].MostAffected.Distance < f[j].MostAffected.Distance
}
func (f furthestRejections) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f *furthestRejections) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*f = append(*f, x.(affectedRejection))
}
func (f *furthestRejections) Pop() interface{} {
	old := *f
	n := len(old)
	x := old[n-1]
	*f = old[0 : n-1]
	return x
}
