// Code generated by MockGen. DO NOT EDIT.
// Source: swarming.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	clients "infra/appengine/crosskylabadmin/app/clients"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	swarming "go.chromium.org/luci/common/api/swarming/swarming/v1"
	strpair "go.chromium.org/luci/common/data/strpair"
)

// MockSwarmingClient is a mock of SwarmingClient interface.
type MockSwarmingClient struct {
	ctrl     *gomock.Controller
	recorder *MockSwarmingClientMockRecorder
}

// MockSwarmingClientMockRecorder is the mock recorder for MockSwarmingClient.
type MockSwarmingClientMockRecorder struct {
	mock *MockSwarmingClient
}

// NewMockSwarmingClient creates a new mock instance.
func NewMockSwarmingClient(ctrl *gomock.Controller) *MockSwarmingClient {
	mock := &MockSwarmingClient{ctrl: ctrl}
	mock.recorder = &MockSwarmingClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSwarmingClient) EXPECT() *MockSwarmingClientMockRecorder {
	return m.recorder
}

// CreateTask mocks base method.
func (m *MockSwarmingClient) CreateTask(c context.Context, name string, args *clients.SwarmingCreateTaskArgs) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTask", c, name, args)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTask indicates an expected call of CreateTask.
func (mr *MockSwarmingClientMockRecorder) CreateTask(c, name, args interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTask", reflect.TypeOf((*MockSwarmingClient)(nil).CreateTask), c, name, args)
}

// GetTaskResult mocks base method.
func (m *MockSwarmingClient) GetTaskResult(ctx context.Context, tid string) (*swarming.SwarmingRpcsTaskResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTaskResult", ctx, tid)
	ret0, _ := ret[0].(*swarming.SwarmingRpcsTaskResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTaskResult indicates an expected call of GetTaskResult.
func (mr *MockSwarmingClientMockRecorder) GetTaskResult(ctx, tid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTaskResult", reflect.TypeOf((*MockSwarmingClient)(nil).GetTaskResult), ctx, tid)
}

// ListAliveBotsInPool mocks base method.
func (m *MockSwarmingClient) ListAliveBotsInPool(arg0 context.Context, arg1 string, arg2 strpair.Map) ([]*swarming.SwarmingRpcsBotInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAliveBotsInPool", arg0, arg1, arg2)
	ret0, _ := ret[0].([]*swarming.SwarmingRpcsBotInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAliveBotsInPool indicates an expected call of ListAliveBotsInPool.
func (mr *MockSwarmingClientMockRecorder) ListAliveBotsInPool(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAliveBotsInPool", reflect.TypeOf((*MockSwarmingClient)(nil).ListAliveBotsInPool), arg0, arg1, arg2)
}

// ListAliveIdleBotsInPool mocks base method.
func (m *MockSwarmingClient) ListAliveIdleBotsInPool(c context.Context, pool string, dims strpair.Map) ([]*swarming.SwarmingRpcsBotInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAliveIdleBotsInPool", c, pool, dims)
	ret0, _ := ret[0].([]*swarming.SwarmingRpcsBotInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAliveIdleBotsInPool indicates an expected call of ListAliveIdleBotsInPool.
func (mr *MockSwarmingClientMockRecorder) ListAliveIdleBotsInPool(c, pool, dims interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAliveIdleBotsInPool", reflect.TypeOf((*MockSwarmingClient)(nil).ListAliveIdleBotsInPool), c, pool, dims)
}

// ListBotTasks mocks base method.
func (m *MockSwarmingClient) ListBotTasks(id string) clients.BotTasksCursor {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBotTasks", id)
	ret0, _ := ret[0].(clients.BotTasksCursor)
	return ret0
}

// ListBotTasks indicates an expected call of ListBotTasks.
func (mr *MockSwarmingClientMockRecorder) ListBotTasks(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBotTasks", reflect.TypeOf((*MockSwarmingClient)(nil).ListBotTasks), id)
}

// ListRecentTasks mocks base method.
func (m *MockSwarmingClient) ListRecentTasks(c context.Context, tags []string, state string, limit int) ([]*swarming.SwarmingRpcsTaskResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRecentTasks", c, tags, state, limit)
	ret0, _ := ret[0].([]*swarming.SwarmingRpcsTaskResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRecentTasks indicates an expected call of ListRecentTasks.
func (mr *MockSwarmingClientMockRecorder) ListRecentTasks(c, tags, state, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRecentTasks", reflect.TypeOf((*MockSwarmingClient)(nil).ListRecentTasks), c, tags, state, limit)
}

// ListSortedRecentTasksForBot mocks base method.
func (m *MockSwarmingClient) ListSortedRecentTasksForBot(c context.Context, botID string, limit int) ([]*swarming.SwarmingRpcsTaskResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListSortedRecentTasksForBot", c, botID, limit)
	ret0, _ := ret[0].([]*swarming.SwarmingRpcsTaskResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListSortedRecentTasksForBot indicates an expected call of ListSortedRecentTasksForBot.
func (mr *MockSwarmingClientMockRecorder) ListSortedRecentTasksForBot(c, botID, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSortedRecentTasksForBot", reflect.TypeOf((*MockSwarmingClient)(nil).ListSortedRecentTasksForBot), c, botID, limit)
}

// MockBotTasksCursor is a mock of BotTasksCursor interface.
type MockBotTasksCursor struct {
	ctrl     *gomock.Controller
	recorder *MockBotTasksCursorMockRecorder
}

// MockBotTasksCursorMockRecorder is the mock recorder for MockBotTasksCursor.
type MockBotTasksCursorMockRecorder struct {
	mock *MockBotTasksCursor
}

// NewMockBotTasksCursor creates a new mock instance.
func NewMockBotTasksCursor(ctrl *gomock.Controller) *MockBotTasksCursor {
	mock := &MockBotTasksCursor{ctrl: ctrl}
	mock.recorder = &MockBotTasksCursorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBotTasksCursor) EXPECT() *MockBotTasksCursorMockRecorder {
	return m.recorder
}

// Next mocks base method.
func (m *MockBotTasksCursor) Next(arg0 context.Context, arg1 int64) ([]*swarming.SwarmingRpcsTaskResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next", arg0, arg1)
	ret0, _ := ret[0].([]*swarming.SwarmingRpcsTaskResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Next indicates an expected call of Next.
func (mr *MockBotTasksCursorMockRecorder) Next(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockBotTasksCursor)(nil).Next), arg0, arg1)
}
