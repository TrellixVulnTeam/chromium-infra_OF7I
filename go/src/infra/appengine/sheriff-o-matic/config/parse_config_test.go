package config

import (
	"io/ioutil"
	"testing"

	"infra/appengine/sheriff-o-matic/som/analyzer"
)

// This test reads config.json and parses it. This tests both that the config is
// valid, and that our parsing code is at least vaguely correct.
func TestConfigParses(t *testing.T) {
	b, err := ioutil.ReadFile("config.json")
	if err != nil {
		t.Errorf("Failed to read config.json: %s", err)
		return
	}

	cr, err := analyzer.ParseConfigRules(b)
	if err != nil {
		t.Errorf("Failed to parse config.json: %s", err)
		return
	}

	// Basic sanity check - there should be at least one master.
	if len(cr.MasterCfgs) == 0 {
		t.Errorf("Expected non-empty config, got %v", cr)
	}
}
