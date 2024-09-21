// Code generated by MockGen. DO NOT EDIT.
// Source: gophermart/internal/polling (interfaces: Store)

// Package polling is a generated GoMock package.
package polling

import (
	context "context"
	model "gophermart/internal/model"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// UpdateOrderInfo mocks base method.
func (m *MockStore) UpdateOrderInfo(arg0 context.Context, arg1 model.AccrualResp) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateOrderInfo", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateOrderInfo indicates an expected call of UpdateOrderInfo.
func (mr *MockStoreMockRecorder) UpdateOrderInfo(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateOrderInfo", reflect.TypeOf((*MockStore)(nil).UpdateOrderInfo), arg0, arg1)
}
