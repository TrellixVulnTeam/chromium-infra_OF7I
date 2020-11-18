package main

import (
	"context"
	"flag"
	"infra/chromeperf/workflows"
	"log"

	"google.golang.org/protobuf/encoding/prototext"

	configProto "go.chromium.org/luci/common/proto/config"
	"go.chromium.org/luci/config"
)

// Scopes to use for OAuth2.0 credentials.
var scopesRequired = []string{
	// Provide access to the email address of the user.
	"https://www.googleapis.com/auth/userinfo.email",
}

// Configuration path we're looking for to support project-defined templates.
const workflowTemplatesFile = "workflow-templates.cfg"
const configSetName = "services/chromeperf-workflow-templates"

type workflowTemplatesServer struct {
	workflows.UnimplementedWorkflowTemplatesServer
	luciConfigClient config.Interface
}

func (*workflowTemplatesServer) ValidateConfig(ctx context.Context, req *configProto.ValidationRequestMessage) (*configProto.ValidationResponseMessage, error) {
	c := WorkflowTemplatesConfig{}
	if err := prototext.Unmarshal(req.Content, &c); err != nil {
		log.Printf("Failed unmarshaling input; error: %v", err)
		return nil, err
	}
	return &configProto.ValidationResponseMessage{}, nil
}

func (s *workflowTemplatesServer) ListWorkflowTemplates(ctx context.Context, req *workflows.ListWorkflowTemplatesRequest) (*workflows.ListWorkflowTemplatesResponse, error) {
	// TODO(dberris): Use a Redis cache for getting configurations?
	// Get a list of configurations.
	configs, err := s.luciConfigClient.GetConfig(ctx, configSetName, workflowTemplatesFile, false)
	if err != nil {
		return nil, err
	}
	c := WorkflowTemplatesConfig{}
	if err := prototext.Unmarshal([]byte(configs.Content), &c); err != nil {
		log.Printf("Failed unmarshaling input; error: %v", err)
		return nil, err
	}
	resp := &workflows.ListWorkflowTemplatesResponse{}
	for _, t := range c.Templates {
		resp.WorkflowTemplates = append(resp.WorkflowTemplates, t)
	}
	return resp, nil
}

var luciConfigService = flag.String("luci-config-service", "https://luci-config.appspot.com/_ah/api", "luci-config service base URL")

func main() {
}
