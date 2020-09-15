package utils

import (
	"context"
	"infra/appengine/cr-rev/config"
	"infra/appengine/cr-rev/models"

	"go.chromium.org/luci/common/logging"
)

// FindBestCommit finds the best commit to redirect to based on configuration:
// * if commit's repository has a priority set, it's returned immedietely
// * if commit's repository has do not index, it won't be returned unless it's
// only available commit
// If config can't be retrieved, very first commit is returned.
func FindBestCommit(ctx context.Context, commits []*models.Commit) *models.Commit {
	if len(commits) == 0 {
		return nil
	}

	cfg, err := config.Get(ctx)
	if err != nil {
		logging.Errorf(ctx, "Couldn't get config, using first commit as the best")
		return commits[0]
	}
	repoPriorityMap := map[string]map[string]*config.Repository{}
	for _, host := range cfg.Hosts {
		m := map[string]*config.Repository{}
		for _, repo := range host.GetRepos() {
			m[repo.GetName()] = repo
		}
		repoPriorityMap[host.GetName()] = m
	}
	ret := commits[0]

	for _, commit := range commits {
		m, ok := repoPriorityMap[commit.Host]
		if !ok {
			continue
		}
		cfg, ok := m[commit.Repository]
		if !ok {
			continue
		}
		if cfg.GetPriority() {
			return commit
		}
		if cfg.GetDoNotIndex() {
			continue
		}
		ret = commit
	}
	return ret
}
