// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"fmt"
	"infra/chromium/bootstrapper/gerrit"
	"infra/chromium/bootstrapper/gitiles"
	"strings"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/exe"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// PropertyBootstrapper provides the functionality for computing the properties
// for the bootstrapped executable.
type PropertyBootstrapper struct {
	gitiles *gitiles.Client
	gerrit  *gerrit.Client
}

func NewPropertyBootstrapper(gitiles *gitiles.Client, gerrit *gerrit.Client) *PropertyBootstrapper {
	return &PropertyBootstrapper{gitiles: gitiles, gerrit: gerrit}
}

// gitilesCommit is a simple wrapper around *buildbucketpb.GitilesCommit with
// the gitiles URI as the string representation.
type gitilesCommit struct {
	*buildbucketpb.GitilesCommit
}

func (c *gitilesCommit) String() string {
	revision := c.Ref
	if c.Id != "" {
		revision = c.Id
	}
	return fmt.Sprintf("%s/%s/+/%s", c.Host, c.Project, revision)
}

// gerritChange is a simple wrapper around *buildbucketpb.GerritChange with the
// gerrit URI as the string representation.
type gerritChange struct {
	*buildbucketpb.GerritChange
}

func (c *gerritChange) String() string {
	return fmt.Sprintf("%s/c/%s/+/%d/%d", c.Host, c.Project, c.Change, c.Patchset)
}

type BootstrapConfig struct {
	// commit is the gitiles commit to read the properties file from.
	commit *gitilesCommit
	// change is gerrit change that may potentially modify the properties
	// file.
	//
	// nil indicates that the build does not contain any gerrit changes that
	// may modify the properties file.
	change *gerritChange

	// buildProperties is the properties that were set on the build.
	buildProperties *structpb.Struct
	// builderProperties is the properties read from the builder's
	// properties file.
	builderProperties *structpb.Struct
}

// GetBootstrapConfig does the necessary work to extract the properties from the
// appropriate version of the properties file.
//
// The properties will be composed of multiple elements:
//   * The properties read from the properties file identified by the
//     config_project and properties_file fields of the build's $bootstrap
//     property.
//   * The $build/chromium_bootstrap property will be set with information about
//     the bootstrapping process that the bootstrapped executable can use to
//     ensure it operates in a manner that is consistent with the bootstrapping
//     process. See chromium_bootstrap.proto for more information.
//   * The build's input properties with the $bootstrap property removed. Values
//     specified in the build's properties override properties in the properties
//     file.
func (b *PropertyBootstrapper) GetBootstrapConfig(ctx context.Context, input *Input) (*BootstrapConfig, error) {
	var config *BootstrapConfig
	switch x := input.properties.ConfigProject.(type) {
	case *BootstrapProperties_TopLevelProject_:
		var err error
		if config, err = b.getTopLevelConfig(ctx, input, x.TopLevelProject); err != nil {
			return nil, err
		}

	default:
		return nil, errors.Reason("config_project handling for type %T is not implemented", x).Err()
	}

	if err := b.populateCommitId(ctx, config.commit); err != nil {
		return nil, errors.Annotate(err, "failed to resolve ID for config commit %s", config.commit).Err()
	}

	config.buildProperties = input.buildProperties
	if err := b.getPropertiesFromFile(ctx, input.properties.PropertiesFile, config); err != nil {
		return nil, errors.Annotate(err, "failed to get properties from properties file %s", input.properties.PropertiesFile).Err()
	}

	return config, nil
}

func (b *PropertyBootstrapper) getTopLevelConfig(ctx context.Context, input *Input, topLevel *BootstrapProperties_TopLevelProject) (*BootstrapConfig, error) {
	ref := topLevel.Ref
	change := findMatchingGerritChange(input.changes, topLevel.Repo)
	if change != nil {
		logging.Infof(ctx, "getting target ref for config change %s", change)
		var err error
		ref, err = b.gerrit.GetTargetRef(ctx, change.Host, change.Project, change.Change)
		if err != nil {
			return nil, errors.Annotate(err, "failed to get target ref for config change %s", change).Err()
		}
	}
	var commit *gitilesCommit
	if matchGitilesCommit(input.commit, topLevel.Repo) {
		commit = &gitilesCommit{proto.Clone(input.commit).(*buildbucketpb.GitilesCommit)}
	}
	if commit == nil {
		commit = &gitilesCommit{&buildbucketpb.GitilesCommit{
			Host:    topLevel.Repo.Host,
			Project: topLevel.Repo.Project,
			Ref:     ref,
		}}
	}
	return &BootstrapConfig{
		commit: commit,
		change: change,
	}, nil
}

// getPropertiesFromFile updates config to include the properties contained in
// the builder's properties file.
func (b *PropertyBootstrapper) getPropertiesFromFile(ctx context.Context, propsFile string, config *BootstrapConfig) error {
	var diff string
	if change := config.change; change != nil {
		// check if it affects the builder properties file and apply change
		logging.Infof(ctx, "determining if properties file %s was affected by %s", propsFile, change)
		info, err := b.gerrit.GetAffectedFileInfo(ctx, change.Host, change.Project, change.Change, change.Patchset, propsFile)
		if err != nil {
			return errors.Annotate(err, "failed to determine if properties file %s was affected by %s", propsFile, change).Err()
		}
		if info == nil {
			logging.Infof(ctx, "properties file %s was not affected by %s", propsFile, change)
		} else {
			// TODO(gbeaty) Investigate which statuses actually make sense to handle
			switch info.Status {
			case gerrit.MODIFIED:
				logging.Infof(ctx, "properties file %s was modified by %s", propsFile, change)
			default:
				return errors.Reason("Unhandled status for properties file %s: %s", propsFile, gerrit.FileStatusName[info.Status]).Err()
			}
			logging.Infof(ctx, "getting revision for %s", change)
			revision, err := b.gerrit.GetRevision(ctx, change.Host, change.Project, change.Change, int32(change.Patchset))
			if err != nil {
				return errors.Annotate(err, "failed to get revision for %s", change).Err()
			}
			logging.Infof(ctx, "getting diff %s", change)
			diff, err = b.gitiles.DownloadDiff(ctx, convertGerritHostToGitilesHost(change.Host), change.Project, revision)
			if err != nil {
				return errors.Annotate(err, "failed to get diff for %s from %s", propsFile, change).Err()
			}
		}
	}

	logging.Infof(ctx, "downloading properties file %s/%s", config.commit, propsFile)
	contents, err := b.gitiles.DownloadFile(ctx, config.commit.Host, config.commit.Project, config.commit.Id, propsFile)
	if err != nil {
		return errors.Annotate(err, "failed to get %s/%s", config.commit, propsFile).Err()
	}
	if diff != "" {
		logging.Infof(ctx, "patching properties file %s", propsFile)
		contents, err = patchFile(ctx, propsFile, contents, diff)
		if err != nil {
			return errors.Annotate(err, "failed to patch properties file %s", propsFile).Err()
		}
	}

	properties := &structpb.Struct{}
	logging.Infof(ctx, "unmarshalling builder properties file")
	if err := protojson.Unmarshal([]byte(contents), properties); err != nil {
		return errors.Annotate(err, "failed to unmarshall builder properties file: {%s}", contents).Err()
	}
	config.builderProperties = properties

	return nil

}

func (b *PropertyBootstrapper) populateCommitId(ctx context.Context, commit *gitilesCommit) error {
	if commit.Id == "" {
		logging.Infof(ctx, "getting revision for %s", commit)
		revision, err := b.gitiles.FetchLatestRevision(ctx, commit.Host, commit.Project, commit.Ref)
		if err != nil {
			return err
		}
		commit.Id = revision
	}
	return nil
}

func matchGitilesCommit(commit *buildbucketpb.GitilesCommit, repo *GitilesRepo) bool {
	return commit != nil && commit.Host == repo.Host && commit.Project == repo.Project
}

func findMatchingGerritChange(changes []*buildbucketpb.GerritChange, repo *GitilesRepo) *gerritChange {
	for _, change := range changes {
		if convertGerritHostToGitilesHost(change.Host) == repo.Host && change.Project == repo.Project {
			return &gerritChange{change}
		}
	}
	return nil
}

func convertGerritHostToGitilesHost(host string) string {
	pieces := strings.SplitN(host, ".", 2)
	pieces[0] = strings.TrimSuffix(pieces[0], "-review")
	return strings.Join(pieces, ".")
}

// GetProperties gets the properties to use for the bootstrapped build.
//
// The properties will be composed of multiple elements:
//   * The properties read from the properties file identified by the
//     config_project and properties_file fields of the build's $bootstrap
//     property.
//   * The $build/chromium_bootstrap property will be set with information about
//     the bootstrapping process that the bootstrapped executable can use to
//     ensure it operates in a manner that is consistent with the bootstrapping
//     process. See chromium_bootstrap.proto for more information.
//   * The build's input properties with the $bootstrap property removed. Values
//     specified in the build's properties override properties in the properties
//     file.
func (c *BootstrapConfig) GetProperties(bootstrappedExe *BootstrappedExe) (*structpb.Struct, error) {
	properties := proto.Clone(c.builderProperties).(*structpb.Struct)

	modProperties := &ChromiumBootstrapModuleProperties{
		Commits: []*buildbucketpb.GitilesCommit{c.commit.GitilesCommit},
		Exe:     bootstrappedExe,
	}
	if err := exe.WriteProperties(properties, map[string]interface{}{
		"$build/chromium_bootstrap": modProperties,
	}); err != nil {
		return nil, errors.Annotate(err, "failed to write out properties for chromium_bootstrap module: {%s}", modProperties).Err()
	}

	for key := range c.buildProperties.Fields {
		delete(properties.Fields, key)
	}
	if err := exe.WriteProperties(properties, c.buildProperties.AsMap()); err != nil {
		return nil, errors.Annotate(err, "failed to write out properties from the build: {%s}", c.buildProperties).Err()
	}

	return properties, nil
}
