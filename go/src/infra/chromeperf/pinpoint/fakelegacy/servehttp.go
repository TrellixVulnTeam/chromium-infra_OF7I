package fakelegacy

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
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
	mu        sync.Mutex
	jobData   map[string]*Job
	templates *Templates
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
