// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

import (
	"encoding/csv"
	"io"
	"sort"
	"strings"
	"sync"

	"go.chromium.org/luci/common/data/sortby"
	"go.chromium.org/luci/common/errors"
)

// ReportDump is a mapping of all reports, generated via DumpReports(ctx).
//
// It maps the ReportID to a list of all Reports found for that ReportID.
type ReportDump struct {
	mu   sync.RWMutex
	data map[ReportID][]*Report
}

// UpdateFrom appends `other` to this ReportDump.
//
// Returns the number of Report records in `other`.
func (r *ReportDump) UpdateFrom(other *ReportDump) int {
	if r == other {
		panic("r.UpdateFrom(r) would cause deadlock")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.data == nil {
		r.data = map[ReportID][]*Report{}
	}

	numReports := 0
	other.Iterate(func(id ReportID, reports []*Report) bool {
		r.data[id] = append(r.data[id], reports...)
		numReports += len(reports)
		return true
	})

	return numReports
}

// Add appends one or more reports to this ReportDump.
func (r *ReportDump) Add(reports ...*Report) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.data == nil {
		r.data = map[ReportID][]*Report{}
	}

	for _, report := range reports {
		r.data[report.ReportID] = append(r.data[report.ReportID], report)
	}
}

// Iterate invokes `cb` for each ReportID with all Reports from that ReportID.
//
// `cb` will be called in sorted order on ReportID. If it returns `false`,
// iteration will stop.
func (r *ReportDump) Iterate(cb func(ReportID, []*Report) bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]ReportID, 0, len(r.data))
	for key := range r.data {
		keys = append(keys, key)
	}
	sort.Slice(keys, sortby.Chain{
		func(i, j int) bool { return keys[i].Project < keys[j].Project },
		func(i, j int) bool { return keys[i].ConfigFile < keys[j].ConfigFile },
	}.Use)
	for _, key := range keys {
		if !cb(key, r.data[key]) {
			break
		}
	}
}

// Clone makes a deep copy of this ReportDump.
func (r *ReportDump) Clone() *ReportDump {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ret := &ReportDump{
		data: make(map[ReportID][]*Report, len(r.data)),
	}
	for k, v := range r.data {
		reports := make([]*Report, 0, len(v))
		for _, report := range v {
			reports = append(reports, report.Clone())
		}
		ret.data[k] = reports
	}

	return ret
}

// Empty returns true iff this ReportDump has no entries.
func (r *ReportDump) Empty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.data) == 0
}

const (
	schemaPrefix  = "{schema="
	schemaSuffix  = "}"
	schemaVersion = "v2"
)

// parseSchemaVersion returns the "v1" from e.g. "{schema=v1}" or "" if `token`
// doesn't have the schema pattern.
func parseSchemaVersion(token string) string {
	if strings.HasPrefix(token, schemaPrefix) && strings.HasSuffix(token, schemaSuffix) {
		return token[len(schemaPrefix) : len(token)-len(schemaSuffix)]
	}
	return ""
}

var csvHeader = []string{
	"Project", "ConfigFile", "Tag", "Problem", "Actionable", "Metadata",
	schemaPrefix + schemaVersion + schemaSuffix,
}

func strSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, av := range a {
		if av != b[i] {
			return false
		}
	}
	return true
}

// NewReportDumpFromCSV reads a raw CSV and returns the *ReportDump.
//
// CSV must be a compatible schema version (i.e. produced with a similar version
// of ReportDump.WriteToCSV.
func NewReportDumpFromCSV(raw io.Reader) (*ReportDump, error) {
	reader := csv.NewReader(raw)
	reader.ReuseRecord = true
	reader.FieldsPerRecord = -1 // variable columns per row

	header, err := reader.Read()
	if err != nil {
		return nil, errors.New("header was missing; empty file")
	}
	version := parseSchemaVersion(header[len(header)-1])
	if version != "" && version != schemaVersion {
		return nil, errors.Reason("unexpected version: %q, expected %q",
			version, schemaVersion).Err()
	}
	if !strSliceEqual(header, csvHeader) {
		return nil, errors.Reason("unexpected header: %q", header).Err()
	}

	ret := &ReportDump{}
	rowIdx := 1
	for {
		row, err := reader.Read()
		if row == nil && err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		rowIdx++

		report, err := NewReportFromCSVRow(row)
		if err != nil {
			return nil, errors.Annotate(err, "reading row %d", rowIdx).Err()
		}

		ret.Add(report)
	}

	return ret, nil
}

// WriteToCSV writes this ReportDump out as CSV.
func (r *ReportDump) WriteToCSV(out io.Writer) error {
	cw := csv.NewWriter(out)
	defer cw.Flush()

	cw.Write(csvHeader)

	r.Iterate(func(key ReportID, reports []*Report) bool {
		for _, report := range reports {
			cw.Write(report.ToCSVRow())
		}
		return true
	})

	cw.Flush()
	return cw.Error()
}
