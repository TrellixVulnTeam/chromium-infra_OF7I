package main

import (
	"context"
	"fmt"
	"os"

	"go.chromium.org/luci/common/logging"
	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
	"golang.org/x/time/rate"

	"infra/appengine/cr-rev/backend/gitiles"
	"infra/appengine/cr-rev/backend/pubsub"
	"infra/appengine/cr-rev/backend/repoimport"
	"infra/appengine/cr-rev/common"
	"infra/appengine/cr-rev/config"
)

// rateLimit is the maximum number of requests per second that gitiles client
// will make to Gitiles server.
const rateLimit = 3

func setupImport(ctx context.Context, cfg *config.Config) {
	hosts := cfg.GetHosts()
	for _, host := range hosts {
		logging.Infof(ctx, "Scanning host %s for new repositories", host.Name)
		fullHost := fmt.Sprintf("%s.googlesource.com", host.Name)
		c, err := gitiles.NewThrottlingClient(
			fullHost, rate.NewLimiter(rateLimit, rateLimit))
		if err != nil {
			panic(fmt.Sprintf("Error creating gitiles rest client: %v", err))
		}

		ctx := gitiles.SetClient(ctx, c)

		// Create import controller for each host.
		importController := repoimport.NewController(repoimport.NewGitilesImporter)
		go importController.Start(ctx)

		initialHostImport(ctx, importController, host)

		pubsubSubscription := host.GetPubsubSubscription()
		if pubsubSubscription == "" {
			logging.Warningf(ctx, "No pubsub subscription found for host: %s", host.GetName())
			continue
		}

		pubsubClient, err := pubsub.NewClient(ctx, os.Getenv("GOOGLE_CLOUD_PROJECT"), pubsubSubscription)
		if err != nil {
			logging.Errorf(ctx, "Couldn't subscribe to host %s, pubsub: %s", host.GetName(), pubsubSubscription)
			continue
		}
		go pubsub.Subscribe(ctx, pubsubClient, pubsub.Processor(host))
	}
}

func initialHostImport(ctx context.Context, importController repoimport.Controller, host *config.Host) {
	c := gitiles.GetClient(ctx)
	req := &gitilesProto.ProjectsRequest{}
	resp, err := c.Projects(ctx, req, nil)
	if err != nil {
		panic(fmt.Sprintf("Error querying gitiles for projects: %v", err))
	}

	repoConfigs := map[string]*config.Repository{}
	for _, repo := range host.GetRepos() {
		repoConfigs[repo.GetName()] = repo
	}
	logging.Infof(ctx, "Found %d repositories in %s", len(resp.GetProjects()), host.Name)
	for _, repo := range resp.GetProjects() {
		repoConfig, ok := repoConfigs[repo]

		if ok {
			if repoConfig.GetDoNotIndex() {
				continue
			}
		}

		logging.Infof(ctx, "scheduling scan for: %s/%s", host.Name, repo)
		importController.Index(common.GitRepository{
			Host:   host.Name,
			Name:   repo,
			Config: repoConfig,
		})
	}
}
