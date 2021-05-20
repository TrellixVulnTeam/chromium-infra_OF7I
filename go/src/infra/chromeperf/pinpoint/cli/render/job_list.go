// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package render

import (
	"infra/chromeperf/pinpoint/proto"
	"io"
	"text/template"
	"time"

	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
)

const (
	jobsTmpl = `{{range .}}{{renderCreateTime .}}:  {{renderJobKind .}} ({{renderState .}}) [{{.JobSpec.Config}}] by {{.CreatedBy}}
  {{JobURL .}}
{{end}}
`
)

func renderCreateTime(j *proto.Job) string {
	ct, err := ptypes.Timestamp(j.CreateTime)
	if err != nil {
		return "(invalid date)"
	}
	return ct.Local().Format(time.RFC3339)
}

func renderJobKind(j *proto.Job) string {
	switch j.JobSpec.JobKind.(type) {
	case *proto.JobSpec_Bisection:
		return j.JobSpec.ComparisonMode.String() + " BISECTION"
	case *proto.JobSpec_Experiment:
		return j.JobSpec.ComparisonMode.String() + " EXPERIMENT"
	default:
		return j.JobSpec.ComparisonMode.String()
	}
}

func renderState(j *proto.Job) string {
	return j.State.String()
}

var listTmpl = template.Must(template.New("Jobs").Funcs(
	template.FuncMap{
		"renderCreateTime": renderCreateTime,
		"renderJobKind":    renderJobKind,
		"renderState":      renderState,
		"JobURL":           JobURL,
	},
).Parse(jobsTmpl))

// JobListText takes a slice of jobs and renders a human-readable list of jobs.
func JobListText(out io.Writer, jobs []*proto.Job) error {
	if err := listTmpl.Execute(out, jobs); err != nil {
		return errors.Annotate(err, "could not render jobs list").Err()
	}
	return nil
}
