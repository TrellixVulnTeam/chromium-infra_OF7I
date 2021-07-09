// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"

	"infra/chromium/bootstrapper/cipd"
	"infra/chromium/bootstrapper/gitiles"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/exe"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Bootstrapper reads the $bootstrap properties and communicates with external
// services to prepare the bootstrapped recipe to run.
type Bootstrapper struct {
	commit          *buildbucketpb.GitilesCommit
	buildProperties *structpb.Struct
	properties      *BootstrapProperties
}

// NewBootstrapper creates a new Bootstrapper, returning an error if the
// $bootstrap property on the build is missing or invalid.
func NewBootstrapper(build *buildbucketpb.Build) (*Bootstrapper, error) {
	bootstrapProperties := &BootstrapProperties{}
	properties := build.GetInput().GetProperties()
	if properties == nil {
		properties = &structpb.Struct{}
	}
	if err := exe.ParseProperties(properties, map[string]interface{}{
		"$bootstrap": bootstrapProperties,
	}); err != nil {
		return nil, errors.Annotate(err, "failed to parse $bootstrap property").Err()
	}

	if err := validate(bootstrapProperties, "$bootstrap"); err != nil {
		return nil, errors.Annotate(err, "failed to validate $bootstrap property").Err()
	}

	properties = proto.Clone(properties).(*structpb.Struct)
	delete(properties.Fields, "$bootstrap")

	bootstrapper := &Bootstrapper{
		commit:          proto.Clone(build.Input.GitilesCommit).(*buildbucketpb.GitilesCommit),
		buildProperties: properties,
		properties:      bootstrapProperties,
	}
	return bootstrapper, nil
}

// ComputeBootstrappedProperties gets the properties that should be set for the
// bootstrapped recipe.
//
// The properties will be composed of multiple elements:
//   * The properties read from the properties file identified by the
//     config_project and properties_file fields of the build's $bootstrap
//     property.
//   * The $build/chromium_bootstrap property will be set with information about
//     the bootstrapping process that the bootstrapped recipe can use to ensure
//     it operates in a manner that is consistent with the bootstrapping
//     process. See chromium_bootstrap.proto for more information.
//   * The build's input properties with the $bootstrap property removed. Values
//     specified in the build's properties override properties in the properties
//     file.
func (b *Bootstrapper) ComputeBootstrappedProperties(ctx context.Context, gitilesClient *gitiles.Client) (*structpb.Struct, error) {
	modProperties := &ChromiumBootstrapModuleProperties{}
	var configCommit *buildbucketpb.GitilesCommit

	switch config := b.properties.ConfigProject.(type) {
	case *BootstrapProperties_TopLevelProject_:
		topLevel := config.TopLevelProject
		if matchGitilesCommit(b.commit, topLevel.Repo) {
			configCommit = b.commit
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

	if err := b.populateCommitId(ctx, gitilesClient, configCommit); err != nil {
		return nil, errors.Annotate(err, "failed to resolve ID for config commit: {%s}", configCommit).Err()
	}
	modProperties.Commits = append(modProperties.Commits, configCommit)

	logging.Infof(ctx, "downloading %s/%s/+/%s", configCommit.Host, configCommit.Project, configCommit.Ref, configCommit.Id)
	contents, err := gitilesClient.DownloadFile(ctx, configCommit.Host, configCommit.Project, configCommit.Id, b.properties.PropertiesFile)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get %s/%s/+/%s/%s", configCommit.Host, configCommit.Project, configCommit.Id, b.properties.PropertiesFile).Err()
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
	for key := range b.buildProperties.Fields {
		delete(properties.Fields, key)
	}
	if err := exe.WriteProperties(properties, b.buildProperties.AsMap()); err != nil {
		return nil, errors.Annotate(err, "failed to write out properties from the build: {%s}", b.buildProperties).Err()
	}

	return properties, nil
}

func (b *Bootstrapper) populateCommitId(ctx context.Context, gitilesClient *gitiles.Client, commit *buildbucketpb.GitilesCommit) error {
	if commit.Id == "" {
		logging.Infof(ctx, "getting revision for %s/%s/+/%s", commit.Host, commit.Project, commit.Ref)
		revision, err := gitilesClient.FetchLatestRevision(ctx, commit.Host, commit.Project, commit.Ref)
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

// SetupExe fetches the CIPD bundle identified by the exe field of the build's
// $bootstrap property and returns the command for invoking the executable.
func (b *Bootstrapper) SetupExe(ctx context.Context, cipdClient *cipd.Client) ([]string, error) {
	logging.Infof(ctx, "downloading CIPD package %s@%s", b.properties.Exe.CipdPackage, b.properties.Exe.CipdVersion)
	packagePath, err := cipdClient.DownloadPackage(ctx, b.properties.Exe.CipdPackage, b.properties.Exe.CipdVersion)
	if err != nil {
		return nil, err
	}
	var cmd []string
	cmd = append(cmd, b.properties.Exe.Cmd...)
	cmd[0] = filepath.Join(packagePath, cmd[0])
	return cmd, nil
}
