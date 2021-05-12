package main

import (
	"context"
	"flag"
	"infra/chromeperf/workflows"
	"infra/chromeperf/workflows_server/proto"
	"strings"

	configProto "go.chromium.org/luci/common/proto/config"
	"go.chromium.org/luci/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
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
	c := proto.WorkflowTemplatesConfig{}
	if err := prototext.Unmarshal(req.Content, &c); err != nil {
		// TODO(dberris): Provide richer error messages for debuggability.
		return nil, status.Errorf(codes.Internal, "Failed unmarshaling config; err: %v", err)
	}
	return &configProto.ValidationResponseMessage{}, nil
}

func (s *workflowTemplatesServer) ListWorkflowTemplates(ctx context.Context, req *workflows.ListWorkflowTemplatesRequest) (*workflows.ListWorkflowTemplatesResponse, error) {
	// TODO(dberris): Use a Redis cache for getting configurations?
	// Get a list of configurations.
	configs, err := s.luciConfigClient.GetConfig(ctx, configSetName, workflowTemplatesFile, false)
	if err != nil {
		// TODO(dberris): Provide richer error messages for debuggability.
		return nil, status.Errorf(codes.Internal, "Failed fetching configuration; err: %v", err)
	}
	c := proto.WorkflowTemplatesConfig{}
	if err := prototext.Unmarshal([]byte(configs.Content), &c); err != nil {
		// TODO(dberris): Provide richer error messages for debuggability.
		return nil, status.Errorf(codes.Internal, "Failed unmarshaling config; err: %v", err)
	}
	resp := &workflows.ListWorkflowTemplatesResponse{}
	for _, t := range c.Templates {
		resp.WorkflowTemplates = append(resp.WorkflowTemplates, t)
	}
	return resp, nil
}

func (s *workflowTemplatesServer) GetWorkflowTemplate(ctx context.Context, req *workflows.GetWorkflowTemplateRequest) (*workflows.WorkflowTemplate, error) {
	// TODO(dberris): Use a Redis cache ofr getting configurations?
	// Get the list of templates.
	configs, err := s.luciConfigClient.GetConfig(ctx, configSetName, workflowTemplatesFile, false)
	if err != nil {
		// TODO(dberris): Provide richer error messages for debuggability.
		return nil, status.Errorf(codes.Internal, "Failed fetching configuration; err: %v", err)
	}
	c := proto.WorkflowTemplatesConfig{}
	if err := prototext.Unmarshal([]byte(configs.Content), &c); err != nil {
		// TODO(dberris): Provide richer error messages for debuggability.
		return nil, status.Errorf(codes.Internal, "Failed unmarshaling config; err: %v", err)
	}
	for _, t := range c.Templates {
		qualName := "/workflow-template/" + t.Name
		if strings.Compare(qualName, req.Name) == 0 {
			return t, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "Template not found: %s", req.Name)
}

var luciConfigService = flag.String("luci-config-service", "https://luci-config.appspot.com/_ah/api", "luci-config service base URL")

func main() {
}
