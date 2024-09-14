// Code generated by MockGen. DO NOT EDIT.
// Source: gophermart/internal/api (interfaces: Poller)

// Package api is a generated GoMock package.
package api

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockPoller is a mock of Poller interface.
type MockPoller struct {
	ctrl     *gomock.Controller
	recorder *MockPollerMockRecorder
}

// MockPollerMockRecorder is the mock recorder for MockPoller.
type MockPollerMockRecorder struct {
	mock *MockPoller
}

// NewMockPoller creates a new mock instance.
func NewMockPoller(ctrl *gomock.Controller) *MockPoller {
	mock := &MockPoller{ctrl: ctrl}
	mock.recorder = &MockPollerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPoller) EXPECT() *MockPollerMockRecorder {
	return m.recorder
}

// Push mocks base method.
func (m *MockPoller) Push(arg0 int) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Push", arg0)
}

// Push indicates an expected call of Push.
func (mr *MockPollerMockRecorder) Push(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Push", reflect.TypeOf((*MockPoller)(nil).Push), arg0)
}