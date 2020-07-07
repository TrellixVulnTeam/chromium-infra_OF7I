package analyzer

import (
	"path/filepath"

	"golang.org/x/net/context"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/logging"

	"infra/monitoring/messages"
)

// GatekeeperRules implements the rule checks that gatekeeper performs
// on failures to determine if the failure should close the tree.
type GatekeeperRules struct {
	cfgs     []*messages.GatekeeperConfig
	treeCfgs map[string][]messages.TreeMasterConfig
}

type categoryAggregator struct {
	excludedSteps stringset.Set
}

func aggregatorFromBuilderConfig(b messages.BuilderConfig) categoryAggregator {
	return categoryAggregator{
		excludedSteps: stringset.NewFromSlice(b.ExcludedSteps...),
	}
}

func (c *categoryAggregator) addCategoryConfig(categoryCfg messages.CategoryConfig) {
	c.excludedSteps = c.excludedSteps.Union(stringset.NewFromSlice(categoryCfg.ExcludedSteps...))
}

func (c categoryAggregator) toBuilderConfig() messages.BuilderConfig {
	return messages.BuilderConfig{
		ExcludedSteps: c.excludedSteps.ToSlice(),
	}
}

// NewGatekeeperRules returns a new instance of GatekeeperRules initialized
// with cfg.
func NewGatekeeperRules(ctx context.Context, cfgs []*messages.GatekeeperConfig, treeCfgs map[string][]messages.TreeMasterConfig) *GatekeeperRules {
	for i := range cfgs {
		cfg := cfgs[i]
		for master := range cfg.Masters {
			masterCfgs := cfg.Masters[master]
			if len(masterCfgs) != 1 {
				logging.Errorf(ctx, "Multiple configs for master: %s", master)
			}

			masterCfg := masterCfgs[0]
			for builder, builderCfg := range masterCfg.Builders {
				aggregator := aggregatorFromBuilderConfig(builderCfg)

				for _, category := range builderCfg.Categories {
					categoryCfg, ok := cfg.Categories[category]
					if !ok {
						logging.Errorf(ctx, "Category %s referenced but not defined for %s:%s", category, master, builder)
						continue
					}
					aggregator.addCategoryConfig(categoryCfg)
				}

				for _, category := range masterCfg.Categories {
					categoryCfg, ok := cfg.Categories[category]
					if !ok {
						logging.Errorf(ctx, "Category %s referenced but not defined for %s:%s", category, master, builder)
						continue
					}
					aggregator.addCategoryConfig(categoryCfg)
				}

				masterCfg.Builders[builder] = aggregator.toBuilderConfig()
			}
		}
	}
	return &GatekeeperRules{cfgs, treeCfgs}
}

func (r *GatekeeperRules) findMaster(master *messages.MasterLocation) ([]messages.MasterConfig, bool) {
	for _, cfg := range r.cfgs {
		if mcs, ok := cfg.Masters[master.String()]; ok {
			return mcs, ok
		}
	}
	return nil, false
}

func (r *GatekeeperRules) getAllowedBuilders(tree string, master *messages.MasterLocation) []string {
	allowed := []string{}

	for _, cfg := range r.treeCfgs[tree] {
		allowed = append(allowed, cfg.Masters[*master]...)
	}

	return allowed
}

func contains(arr []string, s string) bool {
	for _, itm := range arr {
		if itm == s {
			return true
		}
	}

	return false
}

// ExcludeBuilder returns true if a builder should be ignored.
func (r *GatekeeperRules) ExcludeBuilder(ctx context.Context, tree string, master *messages.MasterLocation, builder string) bool {
	mcs, ok := r.findMaster(master)
	if !ok {
		logging.Errorf(ctx, "Can't filter unknown master %s (tree %s)", master, tree)
		return false
	}
	mc := mcs[0]

	allowedBuilders := r.getAllowedBuilders(tree, master)
	if !(contains(allowedBuilders, "*") || contains(allowedBuilders, builder)) {
		return true
	}

	for _, ebName := range mc.ExcludedBuilders {
		if ebName == "*" || ebName == builder {
			return true
		}
	}

	return false
}

// ExcludeFailure returns true if a step failure whould be ignored.
func (r *GatekeeperRules) ExcludeFailure(ctx context.Context, tree string, master *messages.MasterLocation, builder, step string) bool {
	if r.ExcludeBuilder(ctx, tree, master, builder) {
		return true
	}

	mcs, ok := r.findMaster(master)
	if !ok {
		logging.Errorf(ctx, "Can't filter unknown master %s (tree %s)", master, tree)
		return false
	}
	mc := mcs[0]

	for _, ebName := range mc.ExcludedBuilders {
		switch matched, err := filepath.Match(ebName, builder); {
		case err != nil:
			logging.Errorf(ctx, "Malformed builder pattern: %s", ebName)
			return false
		case matched:
			return true
		}
	}

	// Not clear that builder_alerts even looks at the rest of these conditions
	// even though they're specified in gatekeeper.json
	for _, esName := range mc.ExcludedSteps {
		switch matched, err := filepath.Match(esName, step); {
		case err != nil:
			logging.Errorf(ctx, "Malformed step pattern: %s", esName)
			return false
		case matched:
			return true
		}
	}

	bc, ok := mc.Builders[builder]
	if !ok {
		if bc, ok = mc.Builders["*"]; !ok {
			logging.Errorf(ctx, "Unknown %s builder %s", master, builder)
			return true
		}
	}

	for _, esName := range bc.ExcludedSteps {
		switch matched, err := filepath.Match(esName, step); {
		case err != nil:
			logging.Errorf(ctx, "Malformed step pattern: %s", esName)
			return false
		case matched:
			return true
		}
	}

	return false
}
