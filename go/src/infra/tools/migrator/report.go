// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package migrator

import (
	"fmt"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/config"
)

// ReportID is a simple Project/ConfigFile tuple and identifies the object which
// generated the report.
type ReportID struct {
	Project    string
	ConfigFile string
}

// ConfigSet returns the luci-config "config.Set" for this report.
//
// e.g. "projects/${Project}"
func (r ReportID) ConfigSet() config.Set {
	return config.ProjectSet(r.Project)
}

// GenerateReport creates a new *Report from this ReportID.
func (r ReportID) GenerateReport(tag, problem string, opts ...ReportOption) *Report {
	ret := &Report{
		ReportID: r,
		Tag:      tag,
		Problem:  problem,
	}
	for _, o := range opts {
		o(ret)
	}
	return ret
}

func (r ReportID) String() string {
	if r.ConfigFile != "" {
		return r.Project
	}
	return fmt.Sprintf("%s|%s", r.Project, r.ConfigFile)
}

// Report stores a single tagged problem (and metadata).
type Report struct {
	ReportID

	Tag     string
	Problem string

	Metadata map[string]stringset.Set
}

// Clone returns a deep copy of this Report.
func (r *Report) Clone() *Report {
	ret := *r
	if len(ret.Metadata) > 0 {
		meta := make(map[string]stringset.Set, len(r.Metadata))
		for k, vals := range r.Metadata {
			meta[k] = vals.Dup()
		}
		ret.Metadata = meta
	}
	return &ret
}

// ToCSVRow returns a CSV row:
//    Project, ConfigFile, Tag, Problem, Metadata*
//
// Where Metadata* is one key:value entry per value in Metadata.
func (r *Report) ToCSVRow() []string {
	ret := []string{r.Project, r.ConfigFile, r.Tag, r.Problem}
	for key, values := range r.Metadata {
		for _, value := range values.ToSortedSlice() {
			ret = append(ret, fmt.Sprintf("%s:%s", key, value))
		}
	}
	return ret
}

// ReportOption allows attaching additional optional data to reports.
type ReportOption func(*Report)

// MetadataOption returns a ReportOption which allows attaching a string-string
// multimap of metadatadata to a Report.
func MetadataOption(key string, values ...string) ReportOption {
	return func(r *Report) {
		if r.Metadata == nil {
			r.Metadata = map[string]stringset.Set{}
		}
		set, ok := r.Metadata[key]
		if !ok {
			r.Metadata[key] = stringset.NewFromSlice(values...)
			return
		}
		set.AddAll(values)
	}
}
