package api

import (
	"context"
	"infra/appengine/cr-rev/frontend/redirect"
	"infra/appengine/cr-rev/models"
	"infra/appengine/cr-rev/utils"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	redirect redirect.GitRedirect
	rules    *redirect.Rules
}

// NewServer returns new instance of CrrevServer. It uses datastore to retrieve
// commit information. Passed rules should match cr-rev main redirect rules.
func NewServer(rules *redirect.Rules) CrrevServer {
	s := &server{
		redirect: redirect.NewGitilesRedirect(),
		rules:    rules,
	}

	return s
}

// Redirect behaves the same as cr-rev main redirect logic, assuming exact
// rules are passed.
func (s *server) Redirect(ctx context.Context, req *RedirectRequest) (*RedirectResponse, error) {
	q := req.GetQuery()
	url, commit, err := s.rules.FindRedirectURL(ctx, q)
	switch err {
	case nil:
		break
	case redirect.ErrNoMatch:
		return nil, status.Error(codes.NotFound, "Query returned empty result")
	default:
		return nil, err
	}

	res := &RedirectResponse{
		RedirectUrl: url,
	}
	if commit != nil {
		res.GitHash = commit.CommitHash
		res.Host = commit.Host
		res.Repository = commit.Repository
	}
	return res, nil
}

// Numbering looks up for a specific commit given request parameters.
// NumberingRequest.PositionRef can be either SVN footer position or Git and
// must match exactly.
// If there are more than one results, only first one is returned while the
// rest is discarded. This may happen if there are identical footer entries in
// commits.
func (s *server) Numbering(ctx context.Context, req *NumberingRequest) (*NumberingResponse, error) {
	q := datastore.NewQuery("Commit").
		Eq("PositionNumber", req.GetPositionNumber()).
		Eq("Host", req.GetHost()).
		Eq("Repository", req.GetRepository()).
		Eq("PositionRef", req.GetPositionRef())
	commits := []models.Commit{}
	err := datastore.GetAll(ctx, q, &commits)
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, status.Error(codes.NotFound, "Query returned empty result")
	}
	if len(commits) > 1 {
		logging.Warningf(ctx, "Found more than 1 commit for query: %+v", q)
	}

	resp := &NumberingResponse{
		GitHash:        commits[0].CommitHash,
		PositionNumber: int64(commits[0].PositionNumber),
		Host:           commits[0].Host,
		Repository:     commits[0].Repository,
	}
	return resp, nil
}

// Commit looks up a commit given CommitRequest.GitHash. If there are multiple
// commits with the same hash (e.g. mirrored and forked repositories), it tries
// to determine the best one to return (check utils.FindBestCommit).
func (s *server) Commit(ctx context.Context, req *CommitRequest) (*CommitResponse, error) {
	commits, err := models.FindCommitsByHash(ctx, req.GetGitHash())
	if err != nil {
		return nil, err
	}
	commit := utils.FindBestCommit(ctx, commits)
	if commit == nil {
		return nil, status.Error(codes.NotFound, "Query returned empty result")
	}
	url, err := s.redirect.Commit(*commit, "")
	if err != nil {
		return nil, err
	}
	return &CommitResponse{
		GitHash:        commit.CommitHash,
		Host:           commit.Host,
		Repository:     commit.Repository,
		PositionNumber: int64(commit.PositionNumber),
		RedirectUrl:    url,
	}, nil
}
