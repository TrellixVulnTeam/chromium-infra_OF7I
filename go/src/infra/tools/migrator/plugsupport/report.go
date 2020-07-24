// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"sort"
	"sync"

	"infra/tools/migrator"

	"go.chromium.org/luci/common/data/sortby"
)

type reportSink struct {
	mu  sync.Mutex
	dat map[migrator.ReportID][]*migrator.Report
}

func (s *reportSink) add(r *migrator.Report) {
	s.mu.Lock()
	s.dat[r.ReportID] = append(s.dat[r.ReportID], r)
	s.mu.Unlock()
}

func (s *reportSink) dump() *ReportDump {
	s.mu.Lock()
	defer s.mu.Unlock()

	ret := &ReportDump{
		data: make(map[migrator.ReportID][]*migrator.Report, len(s.dat)),
	}
	for k, v := range s.dat {
		reports := make([]*migrator.Report, len(v))
		for i, report := range v {
			reports[i] = report.Clone()
		}
		ret.data[k] = reports
	}
	return ret
}

func (s *reportSink) empty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.dat) == 0
}

var reportSinkKey = "holds a *reportSink"

func getReportSink(ctx context.Context) *reportSink {
	return ctx.Value(&reportSinkKey).(*reportSink)
}

func addReport(ctx context.Context, report *migrator.Report) {
	getReportSink(ctx).add(report)
}

// InitReportSink adds a new empty ReportSink to context and returns the new
// context.
//
// If there's an existing ReportSink, it will be hidden by this.
func InitReportSink(ctx context.Context) context.Context {
	return context.WithValue(ctx, &reportSinkKey, &reportSink{
		dat: map[migrator.ReportID][]*migrator.Report{},
	})
}

// ReportDump is a mapping of all reports, generated via DumpReports(ctx).
//
// It maps the ReportID to a list of all Reports found for that ReportID.
type ReportDump struct {
	mu   sync.RWMutex
	data map[migrator.ReportID][]*migrator.Report
}

// Update appends `other` to this ReportDump.
//
// Returns the number of Report records in `other`.
func (r *ReportDump) Update(other *ReportDump) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.data == nil {
		r.data = map[migrator.ReportID][]*migrator.Report{}
	}

	numReports := 0
	other.Iterate(func(id migrator.ReportID, reports []*migrator.Report) bool {
		r.data[id] = append(r.data[id], reports...)
		numReports += len(reports)
		return false
	})

	return numReports
}

// Iterate invokes `cb` for each ReportID with all Reports from that ReportID.
//
// `cb` will be called in sorted order on ReportID. If it returns `true`,
// iteration will stop.
func (r *ReportDump) Iterate(cb func(migrator.ReportID, []*migrator.Report) bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]migrator.ReportID, 0, len(r.data))
	for key := range r.data {
		keys = append(keys, key)
	}
	sort.Slice(keys, sortby.Chain{
		func(i, j int) bool { return keys[i].Project < keys[j].Project },
		func(i, j int) bool { return keys[i].ConfigFile < keys[j].ConfigFile },
	}.Use)
	for _, key := range keys {
		if cb(key, r.data[key]) {
			break
		}
	}
}

// DumpReports returns all collected Report information within `ctx`.
func DumpReports(ctx context.Context) *ReportDump {
	return getReportSink(ctx).dump()
}

// HasReports returns `true` if `ctx` contains any Reports.
func HasReports(ctx context.Context) bool {
	return !getReportSink(ctx).empty()
}
