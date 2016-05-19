package analyzer

import (
	"infra/monitoring/messages"
)

// GatekeeperRules implements the rule checks that gatekeeper performs
// on failures to determine if the failure should close the tree.
type GatekeeperRules struct {
	cfgs []*messages.GatekeeperConfig
}

// NewGatekeeperRules returns a new instance of GatekeeperRules initialized
// with cfg.
func NewGatekeeperRules(cfgs []*messages.GatekeeperConfig) *GatekeeperRules {
	ret := &GatekeeperRules{cfgs}
	for i, cfg := range cfgs {
		for master, masterCfgs := range cfg.Masters {
			if len(masterCfgs) != 1 {
				errLog.Printf("Multiple configs for master: %s", master)
			}
			ret.cfgs[i].Masters[master] = masterCfgs
		}
	}
	return ret
}

func (r *GatekeeperRules) findMaster(master *messages.MasterLocation) ([]messages.MasterConfig, bool) {
	for _, cfg := range r.cfgs {
		if mcs, ok := cfg.Masters[master.String()]; ok {
			return mcs, ok
		}
	}
	return nil, false
}

// WouldCloseTree returns true if a step failure on given builder/master would
// cause it to close the tree.
func (r *GatekeeperRules) WouldCloseTree(master *messages.MasterLocation, builder, step string) bool {
	mcs, ok := r.findMaster(master)
	if !ok {
		errLog.Printf("Missing master cfg: %s", master)
		return false
	}
	mc := mcs[0]
	bc, ok := mc.Builders[builder]
	if !ok {
		bc, ok = mc.Builders["*"]
		if !ok {
			return false
		}
	}

	// TODO: Check for cfg.Categories
	for _, xstep := range bc.ExcludedSteps {
		if xstep == step {
			return false
		}
	}

	csteps := []string{}
	csteps = append(csteps, bc.ClosingSteps...)
	csteps = append(csteps, bc.ClosingOptional...)

	for _, cs := range csteps {
		if cs == "*" || cs == step {
			return true
		}
	}

	return false
}

// ExcludeFailure returns true if a step failure whould be ignored.
func (r *GatekeeperRules) ExcludeFailure(master *messages.MasterLocation, builder, step string) bool {
	mcs, ok := r.findMaster(master)
	if !ok {
		errLog.Printf("Can't filter unknown master %s", master)
		return false
	}
	mc := mcs[0]

	for _, ebName := range mc.ExcludedBuilders {
		if ebName == "*" || ebName == builder {
			return true
		}
	}

	// Not clear that builder_alerts even looks at the rest of these conditions
	// even though they're specified in gatekeeper.json
	for _, s := range mc.ExcludedSteps {
		if step == s {
			return true
		}
	}

	bc, ok := mc.Builders[builder]
	if !ok {
		if bc, ok = mc.Builders["*"]; !ok {
			errLog.Printf("Unknown %s builder %s", master, builder)
			return true
		}
	}

	for _, esName := range bc.ExcludedSteps {
		if esName == step || esName == "*" {
			return true
		}
	}

	return false
}
