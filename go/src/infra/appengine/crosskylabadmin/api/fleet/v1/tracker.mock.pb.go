// Code generated by MockGen. DO NOT EDIT.
// Source: tracker.pb.go

// Package fleet is a generated GoMock package.
package fleet

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	grpc "google.golang.org/grpc"
)

// MockTrackerClient is a mock of TrackerClient interface.
type MockTrackerClient struct {
	ctrl     *gomock.Controller
	recorder *MockTrackerClientMockRecorder
}

// MockTrackerClientMockRecorder is the mock recorder for MockTrackerClient.
type MockTrackerClientMockRecorder struct {
	mock *MockTrackerClient
}

// NewMockTrackerClient creates a new mock instance.
func NewMockTrackerClient(ctrl *gomock.Controller) *MockTrackerClient {
	mock := &MockTrackerClient{ctrl: ctrl}
	mock.recorder = &MockTrackerClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTrackerClient) EXPECT() *MockTrackerClientMockRecorder {
	return m.recorder
}

// PushBotsForAdminAuditTasks mocks base method.
func (m *MockTrackerClient) PushBotsForAdminAuditTasks(ctx context.Context, in *PushBotsForAdminAuditTasksRequest, opts ...grpc.CallOption) (*PushBotsForAdminAuditTasksResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PushBotsForAdminAuditTasks", varargs...)
	ret0, _ := ret[0].(*PushBotsForAdminAuditTasksResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PushBotsForAdminAuditTasks indicates an expected call of PushBotsForAdminAuditTasks.
func (mr *MockTrackerClientMockRecorder) PushBotsForAdminAuditTasks(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushBotsForAdminAuditTasks", reflect.TypeOf((*MockTrackerClient)(nil).PushBotsForAdminAuditTasks), varargs...)
}

// PushBotsForAdminTasks mocks base method.
func (m *MockTrackerClient) PushBotsForAdminTasks(ctx context.Context, in *PushBotsForAdminTasksRequest, opts ...grpc.CallOption) (*PushBotsForAdminTasksResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PushBotsForAdminTasks", varargs...)
	ret0, _ := ret[0].(*PushBotsForAdminTasksResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PushBotsForAdminTasks indicates an expected call of PushBotsForAdminTasks.
func (mr *MockTrackerClientMockRecorder) PushBotsForAdminTasks(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushBotsForAdminTasks", reflect.TypeOf((*MockTrackerClient)(nil).PushBotsForAdminTasks), varargs...)
}

// PushRepairJobsForLabstations mocks base method.
func (m *MockTrackerClient) PushRepairJobsForLabstations(ctx context.Context, in *PushRepairJobsForLabstationsRequest, opts ...grpc.CallOption) (*PushRepairJobsForLabstationsResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "PushRepairJobsForLabstations", varargs...)
	ret0, _ := ret[0].(*PushRepairJobsForLabstationsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PushRepairJobsForLabstations indicates an expected call of PushRepairJobsForLabstations.
func (mr *MockTrackerClientMockRecorder) PushRepairJobsForLabstations(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushRepairJobsForLabstations", reflect.TypeOf((*MockTrackerClient)(nil).PushRepairJobsForLabstations), varargs...)
}

// ReportBots mocks base method.
func (m *MockTrackerClient) ReportBots(ctx context.Context, in *ReportBotsRequest, opts ...grpc.CallOption) (*ReportBotsResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ReportBots", varargs...)
	ret0, _ := ret[0].(*ReportBotsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReportBots indicates an expected call of ReportBots.
func (mr *MockTrackerClientMockRecorder) ReportBots(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportBots", reflect.TypeOf((*MockTrackerClient)(nil).ReportBots), varargs...)
}

// MockTrackerServer is a mock of TrackerServer interface.
type MockTrackerServer struct {
	ctrl     *gomock.Controller
	recorder *MockTrackerServerMockRecorder
}

// MockTrackerServerMockRecorder is the mock recorder for MockTrackerServer.
type MockTrackerServerMockRecorder struct {
	mock *MockTrackerServer
}

// NewMockTrackerServer creates a new mock instance.
func NewMockTrackerServer(ctrl *gomock.Controller) *MockTrackerServer {
	mock := &MockTrackerServer{ctrl: ctrl}
	mock.recorder = &MockTrackerServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTrackerServer) EXPECT() *MockTrackerServerMockRecorder {
	return m.recorder
}

// PushBotsForAdminAuditTasks mocks base method.
func (m *MockTrackerServer) PushBotsForAdminAuditTasks(arg0 context.Context, arg1 *PushBotsForAdminAuditTasksRequest) (*PushBotsForAdminAuditTasksResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PushBotsForAdminAuditTasks", arg0, arg1)
	ret0, _ := ret[0].(*PushBotsForAdminAuditTasksResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PushBotsForAdminAuditTasks indicates an expected call of PushBotsForAdminAuditTasks.
func (mr *MockTrackerServerMockRecorder) PushBotsForAdminAuditTasks(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushBotsForAdminAuditTasks", reflect.TypeOf((*MockTrackerServer)(nil).PushBotsForAdminAuditTasks), arg0, arg1)
}

// PushBotsForAdminTasks mocks base method.
func (m *MockTrackerServer) PushBotsForAdminTasks(arg0 context.Context, arg1 *PushBotsForAdminTasksRequest) (*PushBotsForAdminTasksResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PushBotsForAdminTasks", arg0, arg1)
	ret0, _ := ret[0].(*PushBotsForAdminTasksResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PushBotsForAdminTasks indicates an expected call of PushBotsForAdminTasks.
func (mr *MockTrackerServerMockRecorder) PushBotsForAdminTasks(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushBotsForAdminTasks", reflect.TypeOf((*MockTrackerServer)(nil).PushBotsForAdminTasks), arg0, arg1)
}

// PushRepairJobsForLabstations mocks base method.
func (m *MockTrackerServer) PushRepairJobsForLabstations(arg0 context.Context, arg1 *PushRepairJobsForLabstationsRequest) (*PushRepairJobsForLabstationsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PushRepairJobsForLabstations", arg0, arg1)
	ret0, _ := ret[0].(*PushRepairJobsForLabstationsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PushRepairJobsForLabstations indicates an expected call of PushRepairJobsForLabstations.
func (mr *MockTrackerServerMockRecorder) PushRepairJobsForLabstations(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushRepairJobsForLabstations", reflect.TypeOf((*MockTrackerServer)(nil).PushRepairJobsForLabstations), arg0, arg1)
}

// ReportBots mocks base method.
func (m *MockTrackerServer) ReportBots(arg0 context.Context, arg1 *ReportBotsRequest) (*ReportBotsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReportBots", arg0, arg1)
	ret0, _ := ret[0].(*ReportBotsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReportBots indicates an expected call of ReportBots.
func (mr *MockTrackerServerMockRecorder) ReportBots(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReportBots", reflect.TypeOf((*MockTrackerServer)(nil).ReportBots), arg0, arg1)
}
