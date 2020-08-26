// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"strings"
	"text/template"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
)

const (
	// originalFormatTagKey is a key of the tag indicating the format of the
	// source data. Possible values: FormatJTR, FormatGTest.
	originalFormatTagKey = "orig_format"

	// FormatGTest is Chromium's GTest format.
	formatGTest = "chromium_gtest"
)

// summaryTmpl is used to generate SummaryHTML in GTest and JTR-based test
// results.
var summaryTmpl = template.Must(template.New("summary").Parse(`
{{ define "gtest" -}}
{{- template "links" .links -}}
{{- if .snippet }}<div><pre>{{.snippet}}</pre></div>{{ end -}}
{{- end}}

{{ define "jtr" -}}
{{- template "links" .links -}}
{{- end}}

{{ define "links" -}}
{{- if . -}}
<ul>
{{- range $name, $url := . -}}
  <li><a href="{{ $url }}">{{ $name }}</a></li>
{{- end -}}
</ul>
{{- end -}}
{{- end -}}
`))

// msToDuration converts a time in milliseconds to duration.Duration.
func msToDuration(t float64) *duration.Duration {
	return ptypes.DurationProto(time.Duration(t) * time.Millisecond)
}

// ensureLeadingDoubleSlash ensures that the path starts with "//".
func ensureLeadingDoubleSlash(path string) string {
	return "//" + strings.TrimLeft(path, "/")
}
