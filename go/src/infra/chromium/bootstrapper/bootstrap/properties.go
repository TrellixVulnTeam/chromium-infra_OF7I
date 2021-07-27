// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"fmt"
	"infra/chromium/bootstrapper/gitiles"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/exe"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// PropertyBootstrapper provides the functionality for computing the properties
// for the bootstrapped executable.
type PropertyBootstrapper struct {
	gitiles *gitiles.Client
}

func NewPropertyBootstrapper(gitiles *gitiles.Client) *PropertyBootstrapper {
	return &PropertyBootstrapper{gitiles: gitiles}
}

// ComputeBootstrappedProperties gets the properties that should be set for the
// bootstrapped executable.
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
func (b *PropertyBootstrapper) ComputeBootstrappedProperties(ctx context.Context, input *Input) (*structpb.Struct, error) {
	modProperties := &ChromiumBootstrapModuleProperties{}
	var configCommit *buildbucketpb.GitilesCommit

	switch config := input.properties.ConfigProject.(type) {
	case *BootstrapProperties_TopLevelProject_:
		topLevel := config.TopLevelProject
		if matchGitilesCommit(input.commit, topLevel.Repo) {
			configCommit = input.commit
		} else {
			configCommit = &buildbucketpb.GitilesCommit{
				Host:    topLevel.Repo.Host,
				Project: topLevel.Repo.Project,
				Ref:     topLevel.Ref,
			}
		}

	default:
		panic(fmt.Sprintf("config_project handling for type %T is not implemented", config))
	}

	if err := b.populateCommitId(ctx, configCommit); err != nil {
		return nil, errors.Annotate(err, "failed to resolve ID for config commit: {%s}", configCommit).Err()
	}
	modProperties.Commits = append(modProperties.Commits, configCommit)

	logging.Infof(ctx, "downloading %s/%s/+/%s", configCommit.Host, configCommit.Project, configCommit.Ref, configCommit.Id)
	contents, err := b.gitiles.DownloadFile(ctx, configCommit.Host, configCommit.Project, configCommit.Id, input.properties.PropertiesFile)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get %s/%s/+/%s/%s", configCommit.Host, configCommit.Project, configCommit.Id, input.properties.PropertiesFile).Err()
	}

	properties := &structpb.Struct{}
	logging.Infof(ctx, "unmarshalling builder properties file")
	if err := protojson.Unmarshal([]byte(contents), properties); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshall builder properties file: {%s}", contents).Err()
	}
	if err := exe.WriteProperties(properties, map[string]interface{}{
		"$build/chromium_bootstrap": modProperties,
	}); err != nil {
		return nil, errors.Annotate(err, "failed to write out properties for chromium_bootstrap module: {%s}", modProperties).Err()
	}
	for key := range input.buildProperties.Fields {
		delete(properties.Fields, key)
	}
	if err := exe.WriteProperties(properties, input.buildProperties.AsMap()); err != nil {
		return nil, errors.Annotate(err, "failed to write out properties from the build: {%s}", input.buildProperties).Err()
	}

	return properties, nil
}

func (b *PropertyBootstrapper) populateCommitId(ctx context.Context, commit *buildbucketpb.GitilesCommit) error {
	if commit.Id == "" {
		logging.Infof(ctx, "getting revision for %s/%s/+/%s", commit.Host, commit.Project, commit.Ref)
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
