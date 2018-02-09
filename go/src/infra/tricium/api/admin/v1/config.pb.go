// Code generated by protoc-gen-go. DO NOT EDIT.
// source: infra/tricium/api/admin/v1/config.proto

/*
Package admin is a generated protocol buffer package.

It is generated from these files:
	infra/tricium/api/admin/v1/config.proto
	infra/tricium/api/admin/v1/driver.proto
	infra/tricium/api/admin/v1/launcher.proto
	infra/tricium/api/admin/v1/reporter.proto
	infra/tricium/api/admin/v1/tracker.proto
	infra/tricium/api/admin/v1/workflow.proto

It has these top-level messages:
	ValidateRequest
	ValidateResponse
	GenerateWorkflowRequest
	GenerateWorkflowResponse
	TriggerRequest
	TriggerResponse
	CollectRequest
	CollectResponse
	LaunchRequest
	LaunchResponse
	ReportLaunchedRequest
	ReportLaunchedResponse
	ReportCompletedRequest
	ReportCompletedResponse
	ReportResultsRequest
	ReportResultsResponse
	WorkflowLaunchedRequest
	WorkflowLaunchedResponse
	WorkerLaunchedRequest
	WorkerLaunchedResponse
	WorkerDoneRequest
	WorkerDoneResponse
	Workflow
	Worker
*/
package admin

import prpc "go.chromium.org/luci/grpc/prpc"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import tricium3 "infra/tricium/api/v1"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ValidateRequest struct {
	// The project configuration to validate.
	ProjectConfig *tricium3.ProjectConfig `protobuf:"bytes,1,opt,name=project_config,json=projectConfig" json:"project_config,omitempty"`
	// The service config to use (optional).
	//
	// If not provided, the default service config will be used.
	ServiceConfig *tricium3.ServiceConfig `protobuf:"bytes,2,opt,name=service_config,json=serviceConfig" json:"service_config,omitempty"`
}

func (m *ValidateRequest) Reset()                    { *m = ValidateRequest{} }
func (m *ValidateRequest) String() string            { return proto.CompactTextString(m) }
func (*ValidateRequest) ProtoMessage()               {}
func (*ValidateRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *ValidateRequest) GetProjectConfig() *tricium3.ProjectConfig {
	if m != nil {
		return m.ProjectConfig
	}
	return nil
}

func (m *ValidateRequest) GetServiceConfig() *tricium3.ServiceConfig {
	if m != nil {
		return m.ServiceConfig
	}
	return nil
}

// TODO(emso): Return structured errors for invalid configs.
type ValidateResponse struct {
	// The config used for validation.
	//
	// This is the resulting config after flattening and merging the provided
	// project and service config.
	ValidatedConfig *tricium3.ProjectConfig `protobuf:"bytes,1,opt,name=validated_config,json=validatedConfig" json:"validated_config,omitempty"`
}

func (m *ValidateResponse) Reset()                    { *m = ValidateResponse{} }
func (m *ValidateResponse) String() string            { return proto.CompactTextString(m) }
func (*ValidateResponse) ProtoMessage()               {}
func (*ValidateResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *ValidateResponse) GetValidatedConfig() *tricium3.ProjectConfig {
	if m != nil {
		return m.ValidatedConfig
	}
	return nil
}

type GenerateWorkflowRequest struct {
	// The project to generate a workflow config for.
	//
	// The project name used must be known to Tricium.
	Project string `protobuf:"bytes,1,opt,name=project" json:"project,omitempty"`
	// The paths to generate the workflow config.
	//
	// These paths are used to decide which workers to include in the workflow.
	Paths []string `protobuf:"bytes,2,rep,name=paths" json:"paths,omitempty"`
}

func (m *GenerateWorkflowRequest) Reset()                    { *m = GenerateWorkflowRequest{} }
func (m *GenerateWorkflowRequest) String() string            { return proto.CompactTextString(m) }
func (*GenerateWorkflowRequest) ProtoMessage()               {}
func (*GenerateWorkflowRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *GenerateWorkflowRequest) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *GenerateWorkflowRequest) GetPaths() []string {
	if m != nil {
		return m.Paths
	}
	return nil
}

type GenerateWorkflowResponse struct {
	// The generated workflow.
	Workflow *Workflow `protobuf:"bytes,1,opt,name=workflow" json:"workflow,omitempty"`
}

func (m *GenerateWorkflowResponse) Reset()                    { *m = GenerateWorkflowResponse{} }
func (m *GenerateWorkflowResponse) String() string            { return proto.CompactTextString(m) }
func (*GenerateWorkflowResponse) ProtoMessage()               {}
func (*GenerateWorkflowResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *GenerateWorkflowResponse) GetWorkflow() *Workflow {
	if m != nil {
		return m.Workflow
	}
	return nil
}

func init() {
	proto.RegisterType((*ValidateRequest)(nil), "admin.ValidateRequest")
	proto.RegisterType((*ValidateResponse)(nil), "admin.ValidateResponse")
	proto.RegisterType((*GenerateWorkflowRequest)(nil), "admin.GenerateWorkflowRequest")
	proto.RegisterType((*GenerateWorkflowResponse)(nil), "admin.GenerateWorkflowResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Config service

type ConfigClient interface {
	// Validates a Tricium config.
	//
	// The config to validate is specified in the request.
	// TODO(emso): Make this RPC public to let users validate configs when they
	// want, or via luci-config.
	Validate(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateResponse, error)
	// Generates a workflow config from a Tricium config.
	//
	// The Tricium config to generate for is specified by the request.
	GenerateWorkflow(ctx context.Context, in *GenerateWorkflowRequest, opts ...grpc.CallOption) (*GenerateWorkflowResponse, error)
}
type configPRPCClient struct {
	client *prpc.Client
}

func NewConfigPRPCClient(client *prpc.Client) ConfigClient {
	return &configPRPCClient{client}
}

func (c *configPRPCClient) Validate(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateResponse, error) {
	out := new(ValidateResponse)
	err := c.client.Call(ctx, "admin.Config", "Validate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *configPRPCClient) GenerateWorkflow(ctx context.Context, in *GenerateWorkflowRequest, opts ...grpc.CallOption) (*GenerateWorkflowResponse, error) {
	out := new(GenerateWorkflowResponse)
	err := c.client.Call(ctx, "admin.Config", "GenerateWorkflow", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type configClient struct {
	cc *grpc.ClientConn
}

func NewConfigClient(cc *grpc.ClientConn) ConfigClient {
	return &configClient{cc}
}

func (c *configClient) Validate(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateResponse, error) {
	out := new(ValidateResponse)
	err := grpc.Invoke(ctx, "/admin.Config/Validate", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *configClient) GenerateWorkflow(ctx context.Context, in *GenerateWorkflowRequest, opts ...grpc.CallOption) (*GenerateWorkflowResponse, error) {
	out := new(GenerateWorkflowResponse)
	err := grpc.Invoke(ctx, "/admin.Config/GenerateWorkflow", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Config service

type ConfigServer interface {
	// Validates a Tricium config.
	//
	// The config to validate is specified in the request.
	// TODO(emso): Make this RPC public to let users validate configs when they
	// want, or via luci-config.
	Validate(context.Context, *ValidateRequest) (*ValidateResponse, error)
	// Generates a workflow config from a Tricium config.
	//
	// The Tricium config to generate for is specified by the request.
	GenerateWorkflow(context.Context, *GenerateWorkflowRequest) (*GenerateWorkflowResponse, error)
}

func RegisterConfigServer(s prpc.Registrar, srv ConfigServer) {
	s.RegisterService(&_Config_serviceDesc, srv)
}

func _Config_Validate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ValidateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfigServer).Validate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/admin.Config/Validate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfigServer).Validate(ctx, req.(*ValidateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Config_GenerateWorkflow_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GenerateWorkflowRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfigServer).GenerateWorkflow(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/admin.Config/GenerateWorkflow",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfigServer).GenerateWorkflow(ctx, req.(*GenerateWorkflowRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Config_serviceDesc = grpc.ServiceDesc{
	ServiceName: "admin.Config",
	HandlerType: (*ConfigServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Validate",
			Handler:    _Config_Validate_Handler,
		},
		{
			MethodName: "GenerateWorkflow",
			Handler:    _Config_GenerateWorkflow_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "infra/tricium/api/admin/v1/config.proto",
}

func init() { proto.RegisterFile("infra/tricium/api/admin/v1/config.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 312 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x52, 0x4f, 0x4b, 0xfb, 0x30,
	0x18, 0xa6, 0xfb, 0xb1, 0xfd, 0xb6, 0x57, 0xb4, 0x25, 0xc8, 0x56, 0x7a, 0xd0, 0xd9, 0x8b, 0x13,
	0xa1, 0xc5, 0x79, 0x14, 0x0f, 0xe2, 0x61, 0x78, 0x93, 0x0e, 0xf5, 0x28, 0xb1, 0x4d, 0x35, 0xba,
	0x35, 0x31, 0xc9, 0xba, 0x8f, 0xe1, 0xdd, 0x4f, 0x2b, 0x36, 0x49, 0x1d, 0x2d, 0x13, 0x8f, 0xef,
	0xfb, 0xfc, 0xe9, 0xf3, 0xe4, 0x2d, 0x1c, 0xd3, 0x22, 0x17, 0x38, 0x56, 0x82, 0xa6, 0x74, 0xb5,
	0x8c, 0x31, 0xa7, 0x31, 0xce, 0x96, 0xb4, 0x88, 0xcb, 0xb3, 0x38, 0x65, 0x45, 0x4e, 0x9f, 0x23,
	0x2e, 0x98, 0x62, 0xa8, 0x5b, 0xad, 0x83, 0x93, 0x5f, 0xf8, 0x6b, 0x26, 0xde, 0xf2, 0x05, 0x5b,
	0x6b, 0x45, 0x70, 0xd4, 0xa6, 0x36, 0x4c, 0xc3, 0x0f, 0x07, 0xdc, 0x7b, 0xbc, 0xa0, 0x19, 0x56,
	0x24, 0x21, 0xef, 0x2b, 0x22, 0x15, 0xba, 0x84, 0x3d, 0x2e, 0xd8, 0x2b, 0x49, 0xd5, 0xa3, 0xe6,
	0xfa, 0xce, 0xd8, 0x99, 0xec, 0x4c, 0x87, 0x91, 0x71, 0x8a, 0x6e, 0x35, 0x7c, 0x5d, 0xa1, 0xc9,
	0x2e, 0xdf, 0x1c, 0xbf, 0xe5, 0x92, 0x88, 0x92, 0xa6, 0xc4, 0xca, 0x3b, 0x0d, 0xf9, 0x5c, 0xc3,
	0x56, 0x2e, 0x37, 0xc7, 0xf0, 0x0e, 0xbc, 0x9f, 0x40, 0x92, 0xb3, 0x42, 0x12, 0x74, 0x05, 0x5e,
	0x69, 0x76, 0xd9, 0xdf, 0x32, 0xb9, 0x35, 0xdf, 0xd8, 0xde, 0xc0, 0x68, 0x46, 0x0a, 0x22, 0xb0,
	0x22, 0x0f, 0xe6, 0x95, 0x6c, 0x5f, 0x1f, 0xfe, 0x9b, 0x06, 0x95, 0xe9, 0x20, 0xb1, 0x23, 0xda,
	0x87, 0x2e, 0xc7, 0xea, 0x45, 0xfa, 0x9d, 0xf1, 0xbf, 0xc9, 0x20, 0xd1, 0x43, 0x38, 0x03, 0xbf,
	0x6d, 0x65, 0x92, 0x9e, 0x42, 0xdf, 0x1e, 0xc1, 0x24, 0x74, 0xa3, 0xea, 0x3c, 0x51, 0x4d, 0xad,
	0x09, 0xd3, 0x4f, 0x07, 0x7a, 0xe6, 0xd1, 0x2e, 0xa0, 0x6f, 0x5b, 0xa3, 0xa1, 0x51, 0x34, 0xee,
	0x12, 0x8c, 0x5a, 0x7b, 0xf3, 0xd1, 0x39, 0x78, 0xcd, 0x40, 0xe8, 0xc0, 0x90, 0xb7, 0x94, 0x0e,
	0x0e, 0xb7, 0xe2, 0xda, 0xf4, 0xa9, 0x57, 0xfd, 0x20, 0xe7, 0x5f, 0x01, 0x00, 0x00, 0xff, 0xff,
	0x1a, 0x87, 0x71, 0xd3, 0xa0, 0x02, 0x00, 0x00,
}
