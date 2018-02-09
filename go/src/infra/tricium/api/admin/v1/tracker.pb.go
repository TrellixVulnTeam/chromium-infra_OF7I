// Code generated by protoc-gen-go. DO NOT EDIT.
// source: infra/tricium/api/admin/v1/tracker.proto

package admin

import prpc "go.chromium.org/luci/grpc/prpc"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import tricium "infra/tricium/api/v1"
import tricium4 "infra/tricium/api/v1"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// WorkflowLaunchedRequest specified details need to mark a workflow as
// launched.
//
// This message should be sent by the Launcher after a workflow has been launched.
type WorkflowLaunchedRequest struct {
	RunId int64 `protobuf:"varint,1,opt,name=run_id,json=runId" json:"run_id,omitempty"`
}

func (m *WorkflowLaunchedRequest) Reset()                    { *m = WorkflowLaunchedRequest{} }
func (m *WorkflowLaunchedRequest) String() string            { return proto.CompactTextString(m) }
func (*WorkflowLaunchedRequest) ProtoMessage()               {}
func (*WorkflowLaunchedRequest) Descriptor() ([]byte, []int) { return fileDescriptor4, []int{0} }

func (m *WorkflowLaunchedRequest) GetRunId() int64 {
	if m != nil {
		return m.RunId
	}
	return 0
}

type WorkflowLaunchedResponse struct {
}

func (m *WorkflowLaunchedResponse) Reset()                    { *m = WorkflowLaunchedResponse{} }
func (m *WorkflowLaunchedResponse) String() string            { return proto.CompactTextString(m) }
func (*WorkflowLaunchedResponse) ProtoMessage()               {}
func (*WorkflowLaunchedResponse) Descriptor() ([]byte, []int) { return fileDescriptor4, []int{1} }

// WorkerLaunchedRequest specifies details needed to mark a worker as launched.
// This includes details useful for the tracking UI.
//
// This message should be sent by the Driver after a swarming task for the
// worker has been triggered.
type WorkerLaunchedRequest struct {
	RunId             int64  `protobuf:"varint,1,opt,name=run_id,json=runId" json:"run_id,omitempty"`
	Worker            string `protobuf:"bytes,2,opt,name=worker" json:"worker,omitempty"`
	IsolatedInputHash string `protobuf:"bytes,3,opt,name=isolated_input_hash,json=isolatedInputHash" json:"isolated_input_hash,omitempty"`
	SwarmingTaskId    string `protobuf:"bytes,4,opt,name=swarming_task_id,json=swarmingTaskId" json:"swarming_task_id,omitempty"`
}

func (m *WorkerLaunchedRequest) Reset()                    { *m = WorkerLaunchedRequest{} }
func (m *WorkerLaunchedRequest) String() string            { return proto.CompactTextString(m) }
func (*WorkerLaunchedRequest) ProtoMessage()               {}
func (*WorkerLaunchedRequest) Descriptor() ([]byte, []int) { return fileDescriptor4, []int{2} }

func (m *WorkerLaunchedRequest) GetRunId() int64 {
	if m != nil {
		return m.RunId
	}
	return 0
}

func (m *WorkerLaunchedRequest) GetWorker() string {
	if m != nil {
		return m.Worker
	}
	return ""
}

func (m *WorkerLaunchedRequest) GetIsolatedInputHash() string {
	if m != nil {
		return m.IsolatedInputHash
	}
	return ""
}

func (m *WorkerLaunchedRequest) GetSwarmingTaskId() string {
	if m != nil {
		return m.SwarmingTaskId
	}
	return ""
}

type WorkerLaunchedResponse struct {
}

func (m *WorkerLaunchedResponse) Reset()                    { *m = WorkerLaunchedResponse{} }
func (m *WorkerLaunchedResponse) String() string            { return proto.CompactTextString(m) }
func (*WorkerLaunchedResponse) ProtoMessage()               {}
func (*WorkerLaunchedResponse) Descriptor() ([]byte, []int) { return fileDescriptor4, []int{3} }

// WorkerDoneRequest specifies details needed to mark a worker as done.
// This includes details useful for the tracking UI.
//
// This message should be sent by the Driver after results from the swarming
// task for a worker have been collected.
type WorkerDoneRequest struct {
	RunId              int64             `protobuf:"varint,1,opt,name=run_id,json=runId" json:"run_id,omitempty"`
	Worker             string            `protobuf:"bytes,2,opt,name=worker" json:"worker,omitempty"`
	IsolatedOutputHash string            `protobuf:"bytes,3,opt,name=isolated_output_hash,json=isolatedOutputHash" json:"isolated_output_hash,omitempty"`
	Provides           tricium.Data_Type `protobuf:"varint,4,opt,name=provides,enum=tricium.Data_Type" json:"provides,omitempty"`
	State              tricium4.State    `protobuf:"varint,5,opt,name=state,enum=tricium.State" json:"state,omitempty"`
}

func (m *WorkerDoneRequest) Reset()                    { *m = WorkerDoneRequest{} }
func (m *WorkerDoneRequest) String() string            { return proto.CompactTextString(m) }
func (*WorkerDoneRequest) ProtoMessage()               {}
func (*WorkerDoneRequest) Descriptor() ([]byte, []int) { return fileDescriptor4, []int{4} }

func (m *WorkerDoneRequest) GetRunId() int64 {
	if m != nil {
		return m.RunId
	}
	return 0
}

func (m *WorkerDoneRequest) GetWorker() string {
	if m != nil {
		return m.Worker
	}
	return ""
}

func (m *WorkerDoneRequest) GetIsolatedOutputHash() string {
	if m != nil {
		return m.IsolatedOutputHash
	}
	return ""
}

func (m *WorkerDoneRequest) GetProvides() tricium.Data_Type {
	if m != nil {
		return m.Provides
	}
	return tricium.Data_NONE
}

func (m *WorkerDoneRequest) GetState() tricium4.State {
	if m != nil {
		return m.State
	}
	return tricium4.State_PENDING
}

type WorkerDoneResponse struct {
}

func (m *WorkerDoneResponse) Reset()                    { *m = WorkerDoneResponse{} }
func (m *WorkerDoneResponse) String() string            { return proto.CompactTextString(m) }
func (*WorkerDoneResponse) ProtoMessage()               {}
func (*WorkerDoneResponse) Descriptor() ([]byte, []int) { return fileDescriptor4, []int{5} }

func init() {
	proto.RegisterType((*WorkflowLaunchedRequest)(nil), "admin.WorkflowLaunchedRequest")
	proto.RegisterType((*WorkflowLaunchedResponse)(nil), "admin.WorkflowLaunchedResponse")
	proto.RegisterType((*WorkerLaunchedRequest)(nil), "admin.WorkerLaunchedRequest")
	proto.RegisterType((*WorkerLaunchedResponse)(nil), "admin.WorkerLaunchedResponse")
	proto.RegisterType((*WorkerDoneRequest)(nil), "admin.WorkerDoneRequest")
	proto.RegisterType((*WorkerDoneResponse)(nil), "admin.WorkerDoneResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Tracker service

type TrackerClient interface {
	// WorkflowLaunched marks the workflow as launched for a specified run.
	WorkflowLaunched(ctx context.Context, in *WorkflowLaunchedRequest, opts ...grpc.CallOption) (*WorkflowLaunchedResponse, error)
	// WorkerLaunched marks the specified worker as launched.
	WorkerLaunched(ctx context.Context, in *WorkerLaunchedRequest, opts ...grpc.CallOption) (*WorkerLaunchedResponse, error)
	// WorkerDone marks the specified worker as done.
	WorkerDone(ctx context.Context, in *WorkerDoneRequest, opts ...grpc.CallOption) (*WorkerDoneResponse, error)
}
type trackerPRPCClient struct {
	client *prpc.Client
}

func NewTrackerPRPCClient(client *prpc.Client) TrackerClient {
	return &trackerPRPCClient{client}
}

func (c *trackerPRPCClient) WorkflowLaunched(ctx context.Context, in *WorkflowLaunchedRequest, opts ...grpc.CallOption) (*WorkflowLaunchedResponse, error) {
	out := new(WorkflowLaunchedResponse)
	err := c.client.Call(ctx, "admin.Tracker", "WorkflowLaunched", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trackerPRPCClient) WorkerLaunched(ctx context.Context, in *WorkerLaunchedRequest, opts ...grpc.CallOption) (*WorkerLaunchedResponse, error) {
	out := new(WorkerLaunchedResponse)
	err := c.client.Call(ctx, "admin.Tracker", "WorkerLaunched", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trackerPRPCClient) WorkerDone(ctx context.Context, in *WorkerDoneRequest, opts ...grpc.CallOption) (*WorkerDoneResponse, error) {
	out := new(WorkerDoneResponse)
	err := c.client.Call(ctx, "admin.Tracker", "WorkerDone", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type trackerClient struct {
	cc *grpc.ClientConn
}

func NewTrackerClient(cc *grpc.ClientConn) TrackerClient {
	return &trackerClient{cc}
}

func (c *trackerClient) WorkflowLaunched(ctx context.Context, in *WorkflowLaunchedRequest, opts ...grpc.CallOption) (*WorkflowLaunchedResponse, error) {
	out := new(WorkflowLaunchedResponse)
	err := grpc.Invoke(ctx, "/admin.Tracker/WorkflowLaunched", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trackerClient) WorkerLaunched(ctx context.Context, in *WorkerLaunchedRequest, opts ...grpc.CallOption) (*WorkerLaunchedResponse, error) {
	out := new(WorkerLaunchedResponse)
	err := grpc.Invoke(ctx, "/admin.Tracker/WorkerLaunched", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trackerClient) WorkerDone(ctx context.Context, in *WorkerDoneRequest, opts ...grpc.CallOption) (*WorkerDoneResponse, error) {
	out := new(WorkerDoneResponse)
	err := grpc.Invoke(ctx, "/admin.Tracker/WorkerDone", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Tracker service

type TrackerServer interface {
	// WorkflowLaunched marks the workflow as launched for a specified run.
	WorkflowLaunched(context.Context, *WorkflowLaunchedRequest) (*WorkflowLaunchedResponse, error)
	// WorkerLaunched marks the specified worker as launched.
	WorkerLaunched(context.Context, *WorkerLaunchedRequest) (*WorkerLaunchedResponse, error)
	// WorkerDone marks the specified worker as done.
	WorkerDone(context.Context, *WorkerDoneRequest) (*WorkerDoneResponse, error)
}

func RegisterTrackerServer(s prpc.Registrar, srv TrackerServer) {
	s.RegisterService(&_Tracker_serviceDesc, srv)
}

func _Tracker_WorkflowLaunched_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WorkflowLaunchedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrackerServer).WorkflowLaunched(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/admin.Tracker/WorkflowLaunched",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrackerServer).WorkflowLaunched(ctx, req.(*WorkflowLaunchedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Tracker_WorkerLaunched_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WorkerLaunchedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrackerServer).WorkerLaunched(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/admin.Tracker/WorkerLaunched",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrackerServer).WorkerLaunched(ctx, req.(*WorkerLaunchedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Tracker_WorkerDone_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WorkerDoneRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrackerServer).WorkerDone(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/admin.Tracker/WorkerDone",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrackerServer).WorkerDone(ctx, req.(*WorkerDoneRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Tracker_serviceDesc = grpc.ServiceDesc{
	ServiceName: "admin.Tracker",
	HandlerType: (*TrackerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "WorkflowLaunched",
			Handler:    _Tracker_WorkflowLaunched_Handler,
		},
		{
			MethodName: "WorkerLaunched",
			Handler:    _Tracker_WorkerLaunched_Handler,
		},
		{
			MethodName: "WorkerDone",
			Handler:    _Tracker_WorkerDone_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "infra/tricium/api/admin/v1/tracker.proto",
}

func init() { proto.RegisterFile("infra/tricium/api/admin/v1/tracker.proto", fileDescriptor4) }

var fileDescriptor4 = []byte{
	// 397 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x92, 0x4d, 0xee, 0xd3, 0x30,
	0x10, 0xc5, 0x15, 0xfe, 0xa4, 0xc0, 0x2c, 0xa2, 0xd6, 0xb4, 0x25, 0x44, 0x40, 0xab, 0x88, 0x45,
	0x56, 0x49, 0x5b, 0x4e, 0x80, 0xd4, 0x05, 0x95, 0x40, 0x48, 0x69, 0x25, 0x96, 0x91, 0xa9, 0x5d,
	0x62, 0xa5, 0xb5, 0x83, 0xed, 0xb4, 0xe2, 0x36, 0x9c, 0x87, 0xd3, 0x70, 0x04, 0x14, 0x3b, 0x29,
	0xfd, 0x94, 0x10, 0xcb, 0xcc, 0xfb, 0xcd, 0xf8, 0x4d, 0xe6, 0x41, 0xc4, 0xf8, 0x46, 0xe2, 0x44,
	0x4b, 0xb6, 0x66, 0xd5, 0x2e, 0xc1, 0x25, 0x4b, 0x30, 0xd9, 0x31, 0x9e, 0xec, 0xa7, 0x89, 0x96,
	0x78, 0x5d, 0x50, 0x19, 0x97, 0x52, 0x68, 0x81, 0x5c, 0x53, 0x0f, 0x46, 0xd7, 0x0d, 0xfb, 0x69,
	0x42, 0xb0, 0xc6, 0x96, 0x0b, 0xc2, 0x9b, 0x40, 0xf3, 0x69, 0x99, 0x70, 0x02, 0x2f, 0xbe, 0x08,
	0x59, 0x6c, 0xb6, 0xe2, 0xf0, 0x11, 0x57, 0x7c, 0x9d, 0x53, 0x92, 0xd2, 0xef, 0x15, 0x55, 0x1a,
	0x0d, 0xa0, 0x23, 0x2b, 0x9e, 0x31, 0xe2, 0x3b, 0x63, 0x27, 0x7a, 0x48, 0x5d, 0x59, 0xf1, 0x05,
	0x09, 0x03, 0xf0, 0xaf, 0x3b, 0x54, 0x29, 0xb8, 0xa2, 0xe1, 0x4f, 0x07, 0x06, 0xb5, 0x48, 0xe5,
	0xbf, 0x0d, 0x43, 0x43, 0xe8, 0x1c, 0x0c, 0xef, 0x3f, 0x1a, 0x3b, 0xd1, 0xb3, 0xb4, 0xf9, 0x42,
	0x31, 0x3c, 0x67, 0x4a, 0x6c, 0xb1, 0xa6, 0x24, 0x63, 0xbc, 0xac, 0x74, 0x96, 0x63, 0x95, 0xfb,
	0x0f, 0x06, 0xea, 0xb5, 0xd2, 0xa2, 0x56, 0x3e, 0x60, 0x95, 0xa3, 0x08, 0xba, 0xea, 0x80, 0xe5,
	0x8e, 0xf1, 0x6f, 0x99, 0xc6, 0xaa, 0xa8, 0x1f, 0x7a, 0x6c, 0x60, 0xaf, 0xad, 0xaf, 0xb0, 0x2a,
	0x16, 0x24, 0xf4, 0x61, 0x78, 0xe9, 0xb0, 0x31, 0xff, 0xcb, 0x81, 0x9e, 0x95, 0xe6, 0x82, 0xd3,
	0xff, 0x34, 0x3e, 0x81, 0xfe, 0xd1, 0xb8, 0xa8, 0xf4, 0x85, 0x73, 0xd4, 0x6a, 0x9f, 0x8d, 0x64,
	0xac, 0xc7, 0xf0, 0xb4, 0x94, 0x62, 0xcf, 0x08, 0x55, 0xc6, 0xb2, 0x37, 0x43, 0x71, 0x7b, 0xa3,
	0x79, 0x7d, 0xcc, 0xd5, 0x8f, 0x92, 0xa6, 0x47, 0x06, 0xbd, 0x05, 0x57, 0x69, 0xac, 0xa9, 0xef,
	0x1a, 0xd8, 0x3b, 0xc2, 0xcb, 0xba, 0x9a, 0x5a, 0x31, 0xec, 0x03, 0x3a, 0xdd, 0xc5, 0xae, 0x38,
	0xfb, 0xed, 0xc0, 0x93, 0x95, 0xcd, 0x12, 0x5a, 0x42, 0xf7, 0xf2, 0x8e, 0xe8, 0x4d, 0x6c, 0xa2,
	0x15, 0xdf, 0x89, 0x44, 0x30, 0xba, 0xab, 0xdb, 0x07, 0xd0, 0x27, 0xf0, 0xce, 0xff, 0x2e, 0x7a,
	0x75, 0xd2, 0x72, 0x15, 0x8b, 0xe0, 0xf5, 0x1d, 0xb5, 0x19, 0xf7, 0x1e, 0xe0, 0xef, 0x16, 0xc8,
	0x3f, 0x83, 0x4f, 0x8e, 0x14, 0xbc, 0xbc, 0xa1, 0xd8, 0x11, 0x5f, 0x3b, 0x26, 0xe7, 0xef, 0xfe,
	0x04, 0x00, 0x00, 0xff, 0xff, 0x7d, 0x6b, 0x77, 0xd0, 0x5f, 0x03, 0x00, 0x00,
}
