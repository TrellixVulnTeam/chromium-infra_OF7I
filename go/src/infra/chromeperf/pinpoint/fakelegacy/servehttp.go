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

package fakelegacy

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"go.chromium.org/luci/common/errors"
)

// NewServer sets up a fake Pinpoint Legacy HTTP server configured to serve
// response templates from the provided templateDir. If jobData is not nil, the
// map is used as the backing store in the Server.
func NewServer(templateDir string, jobData map[string]*Job) (*Server, error) {
	tmpls, err := loadTemplates(templateDir)
	if err != nil {
		return nil, err
	}
	if jobData == nil {
		jobData = make(map[string]*Job)
	}
	return &Server{
		jobData:   jobData,
		templates: tmpls,
	}, nil
}

// Server implements a fake Legacy Pinpoint service. Use NewServer to make one.
type Server struct {
	templates *Templates

	mu       sync.Mutex
	jobData  map[string]*Job
	nextUUID uint64
}

// A Job represents some fake legacy pinpoint job.
type Job struct {
	ID        string
	Status    Status
	UserEmail string
}

// legacyHandler represents a REST API implementation, satisfying http.Handler.
// If an error is returned, the returned code will be used as the HTTP response
// code to serve. Otherwise, the returned legacyResult will be returned to the
// user (by executing the template with the data).
type legacyHandler func(req *http.Request) (_ *legacyResult, code int, _ error)

func (h legacyHandler) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	res, code, err := h(req)
	if err != nil {
		http.Error(wr, err.Error(), code)
		return
	}
	// Buffer up the template output before writing to the ResponseWriter: upon
	// the first Write a 200 OK status is proactively transmitted to the client,
	// preventing us from being able to give an error code for a template error.
	buf := new(bytes.Buffer)
	if err := res.tmpl.Execute(buf, res.data); err != nil {
		http.Error(wr, err.Error(), 500)
		return
	}
	buf.WriteTo(wr)
}

type legacyResult struct {
	tmpl Template
	data interface{}
}

// Handler returns a Handler which implements the entire REST API for Server.
func (s *Server) Handler() http.Handler {
	mux := new(http.ServeMux)
	mux.Handle("/api/job/", legacyHandler(s.getJob))
	mux.Handle("/api/jobs", legacyHandler(s.listJobs))
	mux.Handle("/api/new", legacyHandler(s.newJob))
	return mux
}

// getJob implements individual job lookup requests.
func (s *Server) getJob(req *http.Request) (*legacyResult, int, error) {
	id := strings.Split(req.URL.Path, "/")[3]

	s.mu.Lock()
	defer s.mu.Unlock()

	j, ok := s.jobData[id]
	if !ok {
		return nil, 404, fmt.Errorf("job %q not found", id)
	}

	return &legacyResult{
		s.templates.Job,
		j,
	}, 0, nil
}

// listJobs implements listing job requests.
func (s *Server) listJobs(req *http.Request) (*legacyResult, int, error) {
	args := req.URL.Query()
	if args.Get("filter") != "" {
		return nil, 400, errors.Reason("TODO: implement filter").Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var list []Job
	for _, j := range s.jobData {
		// Copy in the Job data while we have the lock to avoid races after this
		// function returns (while the returned data is being templated).
		list = append(list, *j)
	}

	return &legacyResult{
		s.templates.List,
		list,
	}, 0, nil
}

// newJob implements creating new (fake) pinpoint jobs.
// As of now it ignores all parameters other than 'user'.
func (s *Server) newJob(req *http.Request) (*legacyResult, int, error) {
	if req.Method != "POST" {
		return nil, 400, errors.Reason("only POST supported").Err()
	}
	if err := req.ParseForm(); err != nil {
		return nil, 400, errors.Annotate(err, "error parsing HTTP request").Err()
	}
	q := req.Form

	user := q.Get("user")
	if user == "" {
		return nil, 400, errors.Reason("must set user").Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := formatUUID(s.nextUUID)
	s.nextUUID++
	j := &Job{
		ID:        id,
		Status:    QueuedStatus,
		UserEmail: user,
	}
	s.jobData[id] = j

	return &legacyResult{
		s.templates.New,
		*j,
	}, 0, nil
}

// formatUUID formats the provided integral uuid as a legacy hex ID.
// Specifically, it produces a 14 digit hex number with the first
// digit hard-coded to 1.
func formatUUID(uuid uint64) string {
	return fmt.Sprintf("1%013x", uuid)
}
