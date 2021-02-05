// Copyright 2014 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

// ninja_log.go provides /ninja_log endpoints.

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"

	"infra/appengine/chromium_build_stats/logstore"
	"infra/appengine/chromium_build_stats/ninjalog"
)

type outputFunc func(context.Context, http.ResponseWriter, string, *ninjalog.NinjaLog) error

const (
	traceViewerScript = `
// https://bugs.chromium.org/p/chromium/issues/detail?id=1053869
function openPerfetto(button) {
  const uiLink = 'https://ui.perfetto.dev';
  const perfettoWindow = window.open(uiLink);
  if (perfettoWindow != null) {
    const pingInterval = setInterval(() => {
       perfettoWindow.postMessage('PING', uiLink);
    }, 100);
    window.addEventListener('message', (e) => {
       if (e.origin !== uiLink) return;
       if (e.data !== 'PONG') return;
       clearInterval(pingInterval);
       perfettoWindow.postMessage({'perfetto': {title: button.traceTitle, url: button.tracePage, buffer: button.trace}}, uiLink);
    });
  } else {
    console.warn('Unable to open Perfetto UI window');
  }
}

function openTraceView(button, traceUrl) {
  button.innerText = "fetching trace data...";
  button.disabled = true;
  button.tracePage = traceUrl.substr(0, traceUrl.lastIndexOf("/"));
  button.traceTitle = button.tracePage.substr(button.tracePage.lastIndexOf("/")+1);
  fetch(traceUrl).then((response) => {
    if (!response.ok) {
      button.innerText = "Failed to fetch trace. Please reload the page";
      throw new Error('Failed to fetch trace ' + response.status)
    }
    return response.arrayBuffer();
  }).then((trace) => {
    button.trace = trace;
    openPerfetto(button);
    button.innerText = "open trace on ui.perfetto.dev";
    button.disabled = false;
  });
  button.addEventListener('click', (e) => {
    if (!window.trace) return;
    openPerfetto(trace);
  });
}

`
	traceViewHTML = `
<html>
<head>
 <title>chromium_build_stats: trace view</title>
 <script>
` + traceViewerScript + `
document.addEventListener('DOMContentLoaded', () => {
  const traceUrl = window.location.href.replace('.html', '.json');
  const traceView = document.getElementById('trace_view');
  traceView.addEventListener('click', (e) => {
    if (!traceView.trace) {
      openTraceView(traceView, traceUrl);
      return;
    }
    openPerfetto(traceView.trace);
  });
  openTraceView(traceView, traceUrl);
});
 </script>
</head>
<body>
<button id="trace_view" disabled>open trace on ui.perfetto.dev</button>
</body>
</html>
`
)

var (
	outputs = map[string]outputFunc{
		"lastbuild":     outputFunc(lastBuild),
		"table":         outputFunc(table),
		"metadata.json": outputFunc(metadataJSON),
		"trace.json": outputFunc(
			func(ctx context.Context, w http.ResponseWriter, logPath string, njl *ninjalog.NinjaLog) error {
				return traceJSON(ctx, w, logPath, njl, false)
			}),
		"trace_sort_by_end.json": outputFunc(
			func(ctx context.Context, w http.ResponseWriter, logPath string, njl *ninjalog.NinjaLog) error {
				return traceJSON(ctx, w, logPath, njl, true)
			}),
		// for `?trace`, or explicit url
		"trace.html":             traceHTML,
		"trace_sort_by_end.html": traceHTML,
	}

	formTmpl = template.Must(template.New("form").Parse(`
<html>
<head>
 <title>ninja_log</title>
</head>
<body>
{{if .User}}
 <div align="right">{{.User.Email}} <a href="{{.Logout}}">logout</a></div>
<form action="/ninja_log/" method="post" enctype="multipart/form-data">
<label for="file">.ninja_log file:</label><input type="file" name="file" />
<input type="submit" name="upload" value="upload"><input type="submit" name="trace" value="trace"><input type="reset">
</form>
{{else}}
 <div align="right"><a href="{{.Login}}">login</a></div>
You need to <a href="{{.Login}}">login</a> to upload .ninja_log file.
{{end}}
</body>
</html>`))

	// logstore.URL(path) won't be accessible by user.
	indexTmpl = template.Must(template.New("index").Parse(`
<html>
<head>
 <title>{{.Path}}</title>

<script>
` + traceViewerScript + `
document.addEventListener('DOMContentLoaded', () => {
  const traceView = document.getElementById('trace_view');
  traceView.addEventListener('click', (e) => {
    if (!traceView.trace) {
      openTraceView(traceView, window.location + '/trace.json');
      return;
    }
    openPerfetto(traceView.trace);
  });
  traceView.disabled = false;
  const traceViewSBE = document.getElementById('trace_view_sort_by_end');
  traceViewSBE.addEventListener('click', (e) => {
    if (!traceViewSBE.trace) {
      openTraceView(traceViewSBE, window.location + '/trace_sort_by_end.json');
      return;
    }
    openPerfetto(traceViewSBE.trace);
  });
  traceViewSBE.disabled = false;
});
</script>
</head>
<body>
<h1><a href="/file/{{.Path}}">{{.Path}}</a></h1>
<ul>
 <li><a href="{{.Filename}}/lastbuild">.ninja_log</a>
 <li><a href="{{.Filename}}/table?dedup=true">.ninja_log in table format</a> (<a href="{{.Filename}}/table">full</a>)
 <li><a href="{{.Filename}}/metadata.json">metadata.json</a>
 <li><button id="trace_view" disabled>open trace on ui.perfetto.dev</button> [<a href="{{.Filename}}/trace.json">trace.json</a>]
 <li><button id="trace_view_sort_by_end" disabled>open trace (sort by end) on ui.perfetto.dev</button> [<a href="{{.Filename}}/trace_sort_by_end.json">trace_sort_by_end.json</a>]
</ul>
</body>
</html>
`))

	tableTmpl = template.Must(template.New("table").Parse(`
<html>
<head>
 <title>{{.Filename}}</title>
</head>
<body>
<h1>{{.Filename}}</h1>
Platform: {{.Metadata.Platform}}
Cmdline: {{.Metadata.Cmdline}}
Exit:{{.Metadata.Exit}}
{{if .Metadata.Error}}Error: {{.Metadata.Error}}
{{.Metadata.Raw}}{{end}}
<hr />
<h2>Summary</h2>
{{ .RunTime }} weighted time ({{ .CPUTime }} CPU time, {{printf "%1.1fx" .Parallelism}} parallelism) <br />
ninja startup: {{ .StartupTime }} <br />
ninja end: {{ .EndTime }} <br />
{{ len .Steps }} build steps completed, average of {{printf "%1.2f/s" .StepsPerSec }}

<hr />
<h2>Time by build-step type</h2>
<table border=1>
<tr>
 <th>
 <th>count
 <th>weighted
 <th>duration
 <th>build-step type
</tr>
{{range $i, $stat := .Stats }}
<tr>
 <td>{{$i}}
 <td>{{$stat.Count}}
 <td>{{$stat.Weighted}}
 <td>{{$stat.Time}}
 <td>{{$stat.Type}}
</tr>
{{end}}
</table>

<hr />
<h2>Time by each build-step</h2>
{{$w := .WeightedTimes}}
<table border=1>
<tr>
 <th>n
 <th>weighted
 <th>duration
 <th>start
 <th>end
 <th>restat
 <th>output
</tr>
{{range $i, $step := .Steps}}
<tr>
 <td><a name="{{$i}}" href="#{{$i}}">{{$i}}</a>
 <td>{{index $w $step.Out}}
 <td>{{$step.Duration}}
 <td>{{$step.Start}}
 <td>{{$step.End}}
 <td>{{if gt $step.Restat 0}}{{$step.Restat}}{{end}}
 <td>{{$step.Out}}
  {{range $step.Outs}}<br/>{{.}}{{end}}
</tr>
{{end}}
</table>
</html>
`))
)

func init() {
	http.Handle("/ninja_log/", http.StripPrefix("/ninja_log/", http.HandlerFunc(ninjaLogHandler)))
	http.Handle("/upload_ninja_log/", http.HandlerFunc(uploadNinjaLogHandler))
}

func uploadNinjaLogHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// X-AppEngine-Trusted-IP-Request=1 means the request is coming from a corp machine.
	if r.Header.Get("X-AppEngine-Trusted-IP-Request") != "1" {
		log.Warningf(ctx, "request from non trusted ip")
		http.Error(w, "Access Denied: You're not on corp.", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		log.Warningf(ctx, "request is not post: %s", r.Method)
		http.Error(w, "Only POST method is allowed.", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	gr, err := gzip.NewReader(r.Body)
	if err != nil {
		log.Warningf(ctx, "request is not gzipped?")
		http.Error(w, "Only gzipped body is allowed.", http.StatusUnsupportedMediaType)
		return
	}
	defer gr.Close()
	ninjalog, err := ninjalog.Parse("ninjalog", gr)
	if err != nil {
		log.Errorf(ctx, "failed to parse ninjalog: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Infof(ctx, "ninjalog metadata: %v", ninjalog.Metadata)

	if err := SendToBigquery(ctx, ninjalog, "user"); err != nil {
		http.Error(w, "failed to send BigQuery", http.StatusInternalServerError)
		log.Errorf(ctx, "failed to send BigQuery: %v", err)
		return
	}
	fmt.Fprintln(w, "OK")
}

// ninjaLogHandler handles /<path>/<format> for ninja_log file in gs://chrome-goma-log/<path>
func ninjaLogHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "" {
		ninjalogForm(w, req)
		return
	}

	ctx := appengine.NewContext(req)
	log.Infof(ctx, "/ninja_log: %s", req.URL.Path)

	err := req.ParseForm()
	if err != nil {
		log.Errorf(ctx, "failed to parse form: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// should we set access control like /file?

	logPath, outFunc, err := ninjalogPath(req.URL.Path)
	if err != nil {
		log.Errorf(ctx, "failed to parse request path: %s: %v", req.URL.Path, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Infof(ctx, "fetch ninja_log: %s", logPath)
	njl, err := ninjalogFetch(ctx, logPath)
	if err != nil {
		log.Errorf(ctx, "failed to fetch %s: %v", logPath, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if len(req.Form["dedup"]) > 0 {
		njl.Steps = ninjalog.Dedup(njl.Steps)
	}

	err = outFunc(ctx, w, logPath, njl)
	if err != nil {
		log.Errorf(ctx, "failed to output %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ninjalogForm(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		ninjalogUpload(w, req)
		return
	}
	ctx := appengine.NewContext(req)
	u := user.Current(ctx)
	authPage(w, req, http.StatusOK, formTmpl, u, "/ninja_log/")
}

func ninjalogUpload(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)
	u := user.Current(ctx)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// TODO(ukai): allow only @google.com and @chromium.org?
	log.Infof(ctx, "upload by %s", u.Email)
	f, _, err := req.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()
	njl, err := ninjalog.Parse("ninja_log", f)
	if err != nil {
		log.Errorf(ctx, "bad format: %v", err)
		http.Error(w, fmt.Sprintf("malformed ninja_log file %v", err), http.StatusBadRequest)
		return
	}
	var data bytes.Buffer
	err = ninjalog.Dump(&data, njl.Steps)
	if err != nil {
		log.Errorf(ctx, "dump error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logPath, err := logstore.Upload(ctx, "ninja_log", data.Bytes())
	if err != nil {
		log.Errorf(ctx, "upload error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Infof(ctx, "%s upload to %s", u.Email, logPath)

	if req.FormValue("trace") != "" {
		http.Redirect(w, req, "/ninja_log/"+logPath+"/trace.html", http.StatusSeeOther)
		return
	}
	http.Redirect(w, req, "/ninja_log/"+logPath, http.StatusSeeOther)
}

func ninjalogPath(reqPath string) (string, outputFunc, error) {
	basename := path.Base(reqPath)
	for format, f := range outputs {
		if basename == format {
			logPath := path.Dir(reqPath)
			return logPath, f, nil
		}
	}
	if !strings.HasPrefix(basename, "ninja_log.") {
		return "", nil, fmt.Errorf("unexpected path %s", reqPath)
	}
	return strings.TrimSuffix(reqPath, "/"), outputFunc(indexPage), nil
}

func ninjalogFetch(ctx context.Context, logPath string) (*ninjalog.NinjaLog, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}
	defer client.Close()

	r, err := logstore.Fetch(ctx, client, logPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	rd, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to uncompress: %v", err)
	}
	nl, err := ninjalog.Parse(logPath, rd)
	return nl, err
}

func indexPage(ctx context.Context, w http.ResponseWriter, logPath string, njl *ninjalog.NinjaLog) error {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	data := struct {
		Path     string
		Filename string
	}{
		Path:     logPath,
		Filename: path.Base(njl.Filename),
	}
	return indexTmpl.Execute(w, data)
}

func lastBuild(ctx context.Context, w http.ResponseWriter, logPath string, njl *ninjalog.NinjaLog) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	return ninjalog.Dump(w, njl.Steps)
}

type tableData struct {
	*ninjalog.NinjaLog
	StartupTime   time.Duration
	EndTime       time.Duration
	CPUTime       time.Duration
	WeightedTimes map[string]time.Duration
	Stats         []ninjalog.Stat
}

func (t tableData) RunTime() time.Duration {
	return t.EndTime - t.StartupTime
}

func (t tableData) Parallelism() float64 {
	return float64(t.CPUTime) / float64(t.RunTime())
}

func (t tableData) StepsPerSec() float64 {
	return float64(len(t.Steps)) / t.RunTime().Seconds()
}

func typeFromExt(s ninjalog.Step) string {
	target := s.Out
	target = filepath.Base(target)

	ext := filepath.Ext(target)
	switch ext {
	case ".pdb", ".dll", ".exe":
		return "PEFile (linking)"
	}

	pos := strings.IndexByte(target, '.')
	if pos == -1 {
		return "(no extension found)"
	}

	return target[pos:]
}

func table(ctx context.Context, w http.ResponseWriter, logPath string, njl *ninjalog.NinjaLog) error {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.WriteHeader(http.StatusOK)
	data := tableData{
		NinjaLog: njl,
	}
	data.StartupTime, data.EndTime, data.CPUTime = ninjalog.TotalTime(njl.Steps)
	data.WeightedTimes = ninjalog.WeightedTime(njl.Steps)
	data.Stats = ninjalog.StatsByType(njl.Steps, data.WeightedTimes, typeFromExt)
	// TODO(ukai): sort by req parameter, or sort by javascript.
	sort.Sort(sort.Reverse(ninjalog.ByWeightedTime{
		Weighted: data.WeightedTimes,
		Steps:    njl.Steps,
	}))
	return tableTmpl.Execute(w, data)
}

func metadataJSON(ctx context.Context, w http.ResponseWriter, logPath string, njl *ninjalog.NinjaLog) error {
	js, err := json.Marshal(njl.Metadata)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(js)
	return err
}

func traceJSON(ctx context.Context, w http.ResponseWriter, logPath string, njl *ninjalog.NinjaLog, sortByEnd bool) error {
	log.Infof(ctx, "traceJSON for %s sort_by_end=%t", logPath, sortByEnd)
	steps := ninjalog.Dedup(njl.Steps)
	flow := ninjalog.Flow(steps, sortByEnd)
	traces := ninjalog.ToTraces(flow, 1)
	js, err := json.Marshal(traces)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	// it doesn't work as ui.perfetto.dev doesn't allow *.appspot.com.
	// https://bugs.chromium.org/p/chromium/issues/detail?id=1053869#c6
	w.Header().Set("Access-Control-Allow-Origin", "https://ui.perfetto.dev")
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(js)
	return err
}

func traceHTML(ctx context.Context, w http.ResponseWriter, logPath string, njl *ninjalog.NinjaLog) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(traceViewHTML))
	return err
}
