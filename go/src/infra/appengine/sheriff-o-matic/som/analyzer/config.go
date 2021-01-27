package analyzer

import (
	"context"
	"encoding/json"
	"path/filepath"

	"infra/appengine/sheriff-o-matic/som/client"

	"go.chromium.org/luci/common/logging"
)

const configURL = "https://chromium.googlesource.com/infra/infra/+/HEAD/go/src/infra/appengine/sheriff-o-matic/config/config.json?format=text"

// ConfigRules is a parsed representation of the config.json file, which
// specifies builders and steps to exclude.
type ConfigRules struct {
	IgnoredSteps     []string                      `json:"ignored_steps"`
	BuilderGroupCfgs map[string]BuilderGroupConfig `json:"builder_groups"`
}

// BuilderGroupConfig is a parsed representation of the inner per builder group
// value, which contains a list of builders to exclude for that builder group.
type BuilderGroupConfig struct {
	ExcludedBuilders []string `json:"excluded_builders"`
}

// GetConfigRules fetches the latest version of the config from Gitiles.
func GetConfigRules(c context.Context) (*ConfigRules, error) {
	b, err := client.GetGitilesCached(c, configURL)
	if err != nil {
		return nil, err
	}

	return ParseConfigRules(b)
}

// ParseConfigRules parses the given byte array into a ConfigRules object.
// Public so that parse_config_test can use it.
func ParseConfigRules(cfgJSON []byte) (*ConfigRules, error) {
	cr := &ConfigRules{}
	if err := json.Unmarshal(cfgJSON, cr); err != nil {
		return nil, err
	}

	return cr, nil
}

// ExcludeFailure determines whether a particular failure should be ignored,
// according to the rules in the config.
func (r *ConfigRules) ExcludeFailure(ctx context.Context, builderGroup, builder, step string) bool {
	if cfg, ok := r.BuilderGroupCfgs[builderGroup]; ok {
		if contains(cfg.ExcludedBuilders, builder) {
			return true
		}
	}

	for _, stepPattern := range r.IgnoredSteps {
		matched, err := filepath.Match(stepPattern, step)
		if err != nil {
			logging.Errorf(ctx, "Malformed step pattern: %s", stepPattern)
		} else if matched {
			return true
		}
	}

	return false
}

func contains(arr []string, s string) bool {
	for _, itm := range arr {
		if itm == s {
			return true
		}
	}

	return false
}
