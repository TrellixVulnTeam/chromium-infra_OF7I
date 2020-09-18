package api

import (
	"net/http"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"google.golang.org/grpc/status"
)

var marshaler = &jsonpb.Marshaler{}

type restAPIServer struct {
	grpcServer CrrevServer
}

func (s *restAPIServer) handleRedirect(c *router.Context) {
	// gRPC expects a leading slash. However, router doesn't include it in
	// named parameter.
	q := "/" + c.Params.ByName("query")
	req := &RedirectRequest{
		Query: q,
	}
	resp, err := s.grpcServer.Redirect(c.Context, req)
	if err != nil {
		handleError(c, err)
	}
	marshaler.Marshal(c.Writer, resp)
}

func (s *restAPIServer) handleNumbering(c *router.Context) {
	queryValues := c.Request.URL.Query()
	n, err := strconv.Atoi(queryValues.Get("number"))
	if err != nil {
		http.Error(c.Writer, "Parameter number is not an integer", http.StatusBadRequest)
		return
	}

	req := &NumberingRequest{
		Host:           queryValues.Get("project"),
		Repository:     queryValues.Get("repo"),
		PositionRef:    queryValues.Get("numbering_identifier"),
		PositionNumber: int64(n),
	}
	resp, err := s.grpcServer.Numbering(c.Context, req)
	if err != nil {
		handleError(c, err)
		return
	}
	marshaler.Marshal(c.Writer, resp)
}

func (s *restAPIServer) handleCommit(c *router.Context) {
	req := &CommitRequest{
		GitHash: c.Params.ByName("hash"),
	}
	resp, err := s.grpcServer.Commit(c.Context, req)
	if err != nil {
		handleError(c, err)
		return
	}
	marshaler.Marshal(c.Writer, resp)
}

func handleError(c *router.Context, err error) {
	if err, ok := status.FromError(err); ok {
		http.NotFound(c.Writer, c.Request)
	} else {
		logging.Errorf(c.Context, "Error in API while handling redirect: %w", err)
		http.Error(c.Writer, "Internal server errror", http.StatusInternalServerError)
	}
}

// NewRESTServer installs REST handlers to provided router.
func NewRESTServer(r *router.Router, grpcServer CrrevServer) {
	s := &restAPIServer{
		grpcServer: grpcServer,
	}
	mw := router.MiddlewareChain{}

	r.GET("/redirect/:query", mw, s.handleRedirect)
	r.GET("/get_numbering", mw, s.handleNumbering)
	r.GET("/commit/:hash", mw, s.handleCommit)
}
