// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"

	"infra/tools/migrator"
)

type reportSink struct {
	dat *migrator.ReportDump
}

func (s *reportSink) add(id migrator.ReportID, tag, problem string, opts ...migrator.ReportOption) {
	report := &migrator.Report{
		ReportID:   id,
		Tag:        tag,
		Problem:    problem,
		Actionable: true,
	}
	for _, o := range opts {
		o(report)
	}

	s.dat.Add(report)
}

var reportSinkKey = "holds a *reportSink"

func getReportSink(ctx context.Context) *reportSink {
	return ctx.Value(&reportSinkKey).(*reportSink)
}

// InitReportSink adds a new empty ReportSink to context and returns the new
// context.
//
// If there's an existing ReportSink, it will be hidden by this.
func InitReportSink(ctx context.Context) context.Context {
	return context.WithValue(ctx, &reportSinkKey, &reportSink{
		dat: &migrator.ReportDump{},
	})
}

// DumpReports returns all collected Report information within `ctx`.
func DumpReports(ctx context.Context) *migrator.ReportDump {
	return getReportSink(ctx).dat.Clone()
}

// HasActionableReports returns `true` if `ctx` contains any Reports where
// Actionable is true.
func HasActionableReports(ctx context.Context) (actionable bool) {
	getReportSink(ctx).dat.Iterate(func(id migrator.ReportID, reports []*migrator.Report) bool {
		for _, report := range reports {
			if report.Actionable {
				actionable = true
				return false
			}
		}
		return true
	})
	return
}
