// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/gae/service/datastore"
	_ "go.chromium.org/luci/gae/service/datastore/crbug1242998safeget"
	"go.chromium.org/luci/gae/service/info"
	"go.chromium.org/luci/server/caching"
)

var projectCacheSlot = caching.RegisterCacheSlot()

const projectConfigKind = "weetbix.ProjectConfig"

var (
	importAttemptCounter = metric.NewCounter(
		"weetbix/project_config/import_attempt",
		"The number of import attempts of project config",
		nil,
		// status can be "success" or "failure".
		field.String("project"), field.String("status"))
)

type cachedProjectConfig struct {
	_extra datastore.PropertyMap `gae:"-,extra"`
	_kind  string                `gae:"$kind,weetbix.ProjectConfig"`

	ID     string      `gae:"$id"` // The name of the project for which the config is.
	Config []byte      `gae:",noindex"`
	Meta   config.Meta `gae:",noindex"`
}

func init() {
	// Registers validation of the given configuration paths with cfgmodule.
	validation.Rules.Add("regex:projects/.*", "${appid}.cfg", func(ctx *validation.Context, configSet, path string, content []byte) error {
		// Discard the returned deserialized message.
		validateProjectConfigRaw(ctx, string(content))
		return nil
	})
}

// updateProjects fetches fresh project-level configuration from LUCI Config
// service and stores it in datastore.
func updateProjects(ctx context.Context) error {
	// Fetch freshest configs from the LUCI Config.
	fetchedConfigs, err := fetchLatestProjectConfigs(ctx)
	if err != nil {
		return err
	}

	var errs []error
	parsedConfigs := make(map[string]*fetchedProjectConfig)
	for project, fetch := range fetchedConfigs {
		valCtx := validation.Context{Context: ctx}
		valCtx.SetFile(fetch.Path)
		msg := validateProjectConfigRaw(&valCtx, fetch.Content)
		if err := valCtx.Finalize(); err != nil {
			blocking := err.(*validation.Error).WithSeverity(validation.Blocking)
			if blocking != nil {
				// Continue through validation errors to ensure a validation
				// error in one project does not affect other projects.
				errs = append(errs, errors.Annotate(blocking, "validation errors for %q", project).Err())
				msg = nil
			}
		}
		// We create an entry even for invalid config (where msg == nil),
		// because we want to signal that config for this project still exists
		// and existing config should be retained instead of being deleted.
		parsedConfigs[project] = &fetchedProjectConfig{
			Config: msg,
			Meta:   fetch.Meta,
		}
	}
	forceUpdate := false
	success := true
	if err := updateStoredConfig(ctx, parsedConfigs, forceUpdate); err != nil {
		errs = append(errs, err)
		success = false
	}
	// Report success for all projects that passed validation, assuming the
	// update succeeded.
	for project, config := range parsedConfigs {
		status := "success"
		if !success || config.Config == nil {
			status = "failure"
		}
		importAttemptCounter.Add(ctx, 1, project, status)
	}

	if len(errs) > 0 {
		return errors.NewMultiError(errs...)
	}
	return nil
}

type fetchedProjectConfig struct {
	// config is the project-level configuration, if it has passed validation,
	// and nil otherwise.
	Config *ProjectConfig
	// meta is populated with config metadata.
	Meta config.Meta
}

// updateStoredConfig updates the config stored in datastore. fetchedConfigs
// contains the new configs to store, forceUpdate forces overwrite of existing
// configuration (ignoring whether the config revision is newer).
func updateStoredConfig(ctx context.Context, fetchedConfigs map[string]*fetchedProjectConfig, forceUpdate bool) error {
	// Drop out of any existing datastore transactions.
	ctx = cleanContext(ctx)

	currentConfigs, err := fetchProjectConfigEntities(ctx)
	if err != nil {
		return err
	}

	var errs []error
	var toPut []*cachedProjectConfig
	for project, fetch := range fetchedConfigs {
		if fetch.Config == nil {
			// Config did not pass validation.
			continue
		}
		blob, err := proto.Marshal(fetch.Config)
		if err != nil {
			// Continue through errors to ensure bad config for one project
			// does not affect others.
			errs = append(errs, errors.Annotate(err, "").Err())
			continue
		}
		cur, ok := currentConfigs[project]
		if !ok {
			cur = &cachedProjectConfig{
				ID: project,
			}
		}
		if !forceUpdate && cur.Meta.Revision == fetch.Meta.Revision {
			logging.Infof(ctx, "Cached config %s is up-to-date at rev %q", cur.ID, cur.Meta.Revision)
			continue
		}
		logging.Infof(ctx, "Updating cached config %s: %q -> %q", cur.ID, cur.Meta.Revision, fetch.Meta.Revision)
		toPut = append(toPut, &cachedProjectConfig{
			ID:     cur.ID,
			Config: blob,
			Meta:   fetch.Meta,
		})
	}
	if err := datastore.Put(ctx, toPut); err != nil {
		errs = append(errs, errors.Annotate(err, "updating project configs").Err())
	}

	var toDelete []*datastore.Key
	for project, cur := range currentConfigs {
		if _, ok := fetchedConfigs[project]; ok {
			continue
		}
		toDelete = append(toDelete, datastore.KeyForObj(ctx, cur))
	}

	if err := datastore.Delete(ctx, toDelete); err != nil {
		errs = append(errs, errors.Annotate(err, "deleting stale project configs").Err())
	}

	if len(errs) > 0 {
		return errors.NewMultiError(errs...)
	}
	return nil
}

func fetchLatestProjectConfigs(ctx context.Context) (map[string]config.Config, error) {
	configs, err := cfgclient.Client(ctx).GetProjectConfigs(ctx, "${appid}.cfg", false)
	if err != nil {
		return nil, err
	}
	result := make(map[string]config.Config)
	for _, cfg := range configs {
		project := cfg.ConfigSet.Project()
		if project != "" {
			result[project] = cfg
		}
	}
	return result, nil
}

// fetchProjectConfigEntities retrieves project configuration entities
// from datastore, including metadata.
func fetchProjectConfigEntities(ctx context.Context) (map[string]*cachedProjectConfig, error) {
	var configs []*cachedProjectConfig
	err := datastore.GetAll(ctx, datastore.NewQuery(projectConfigKind), &configs)
	if err != nil {
		return nil, errors.Annotate(err, "fetching project configs from datastore").Err()
	}
	result := make(map[string]*cachedProjectConfig)
	for _, cfg := range configs {
		result[cfg.ID] = cfg
	}
	return result, nil
}

// Projects returns all project configurations, in a map by project name.
// Uses in-memory cache to avoid hitting datastore all the time.
func Projects(ctx context.Context) (map[string]*ProjectConfig, error) {
	val, err := projectCacheSlot.Fetch(ctx, func(interface{}) (val interface{}, exp time.Duration, err error) {
		var pc map[string]*ProjectConfig
		if pc, err = fetchProjects(ctx); err != nil {
			return nil, 0, err
		}
		return pc, time.Minute, nil
	})
	switch {
	case err == caching.ErrNoProcessCache:
		// A fallback useful in unit tests that may not have the process cache
		// available. Production environments usually have the cache installed
		// by the framework code that initializes the root context.
		return fetchProjects(ctx)
	case err != nil:
		return nil, err
	default:
		pc := val.(map[string]*ProjectConfig)
		return pc, nil
	}
}

// fetchProjects retrieves all project configurations from datastore.
func fetchProjects(ctx context.Context) (map[string]*ProjectConfig, error) {
	ctx = cleanContext(ctx)

	cachedCfgs, err := fetchProjectConfigEntities(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "fetching cached config").Err()
	}
	result := make(map[string]*ProjectConfig)
	for project, cached := range cachedCfgs {
		cfg := &ProjectConfig{}
		if err := proto.Unmarshal(cached.Config, cfg); err != nil {
			return nil, errors.Annotate(err, "unmarshalling cached config").Err()
		}
		result[project] = cfg
	}
	return result, nil
}

// cleanContext returns a context with datastore using the default namespace
// and not using transactions.
func cleanContext(ctx context.Context) context.Context {
	return datastore.WithoutTransaction(info.MustNamespace(ctx, ""))
}

// SetTestProjectConfig sets test project configuration in datastore.
// It should be used from unit/integration tests only.
func SetTestProjectConfig(ctx context.Context, cfg map[string]*ProjectConfig) error {
	fetchedConfigs := make(map[string]*fetchedProjectConfig)
	for project, pcfg := range cfg {
		fetchedConfigs[project] = &fetchedProjectConfig{
			Config: pcfg,
			Meta:   config.Meta{},
		}
	}
	forceUpdate := true
	if err := updateStoredConfig(ctx, fetchedConfigs, forceUpdate); err != nil {
		return err
	}
	testable := datastore.GetTestable(ctx)
	if testable == nil {
		return errors.New("SetTestProjectConfig should only be used with testable datastore implementations")
	}
	// An up-to-date index is required for fetch to retrieve the project
	// entities we just saved.
	testable.CatchupIndexes()
	return nil
}
