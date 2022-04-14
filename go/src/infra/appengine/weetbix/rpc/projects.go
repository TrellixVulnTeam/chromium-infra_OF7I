// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	pb "infra/appengine/weetbix/proto/v1"
)

type projectServer struct{}

func NewProjectsServer() *pb.DecoratedProjects {
	return &pb.DecoratedProjects{
		Prelude:  checkAllowedPrelude,
		Service:  &projectServer{},
		Postlude: gRPCifyAndLogPostlude,
	}
}

func (*projectServer) List(ctx context.Context, request *pb.ListProjectsRequest) (*pb.ListProjectsResponse, error) {
	projects, err := config.Projects(ctx)

	if err != nil {
		return nil, errors.Annotate(err, "fetching project configs").Err()
	}

	return &pb.ListProjectsResponse{
		Projects: createProjectPbs(projects),
	}, nil
}

func createProjectPbs(projectConfigs map[string]*configpb.ProjectConfig) []*pb.Project {
	projectsPbs := make([]*pb.Project, 0, len(projectConfigs))
	for key, projectConfig := range projectConfigs {
		var usedDisplayName string
		projectMetadata := projectConfig.ProjectMetadata
		if projectMetadata != nil && projectMetadata.DisplayName != "" {
			usedDisplayName = projectConfig.ProjectMetadata.DisplayName
		} else {
			usedDisplayName = strings.Title(key)
		}
		projectsPbs = append(projectsPbs, &pb.Project{
			Name:        fmt.Sprintf("projects/%s", key),
			DisplayName: usedDisplayName,
			Project:     key,
		})
	}
	return projectsPbs
}
