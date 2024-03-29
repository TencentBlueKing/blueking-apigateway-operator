// Code generated by MockGen. DO NOT EDIT.
// Source: types.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	registry "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	gomock "github.com/golang/mock/gomock"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockRegistry is a mock of Registry interface.
type MockRegistry struct {
	ctrl     *gomock.Controller
	recorder *MockRegistryMockRecorder
}

// MockRegistryMockRecorder is the mock recorder for MockRegistry.
type MockRegistryMockRecorder struct {
	mock *MockRegistry
}

// NewMockRegistry creates a new mock instance.
func NewMockRegistry(ctrl *gomock.Controller) *MockRegistry {
	mock := &MockRegistry{ctrl: ctrl}
	mock.recorder = &MockRegistryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegistry) EXPECT() *MockRegistryMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockRegistry) Get(ctx context.Context, key registry.ResourceKey, obj client.Object) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, key, obj)
	ret0, _ := ret[0].(error)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockRegistryMockRecorder) Get(ctx, key, obj interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockRegistry)(nil).Get), ctx, key, obj)
}

// List mocks base method.
func (m *MockRegistry) List(ctx context.Context, key registry.ResourceKey, obj client.ObjectList) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, key, obj)
	ret0, _ := ret[0].(error)
	return ret0
}

// List indicates an expected call of List.
func (mr *MockRegistryMockRecorder) List(ctx, key, obj interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockRegistry)(nil).List), ctx, key, obj)
}

// ListStages mocks base method.
func (m *MockRegistry) ListStages(ctx context.Context) ([]registry.StageInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListStages", ctx)
	ret0, _ := ret[0].([]registry.StageInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListStages indicates an expected call of ListStages.
func (mr *MockRegistryMockRecorder) ListStages(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListStages", reflect.TypeOf((*MockRegistry)(nil).ListStages), ctx)
}

// Watch mocks base method.
func (m *MockRegistry) Watch(ctx context.Context) <-chan *registry.ResourceMetadata {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", ctx)
	ret0, _ := ret[0].(<-chan *registry.ResourceMetadata)
	return ret0
}

// Watch indicates an expected call of Watch.
func (mr *MockRegistryMockRecorder) Watch(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockRegistry)(nil).Watch), ctx)
}
