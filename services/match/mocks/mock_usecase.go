// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/piresc/nebengjek/services/match (interfaces: MatchUC)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/piresc/nebengjek/internal/pkg/models"
)

// MockMatchUC is a mock of MatchUC interface.
type MockMatchUC struct {
	ctrl     *gomock.Controller
	recorder *MockMatchUCMockRecorder
}

// MockMatchUCMockRecorder is the mock recorder for MockMatchUC.
type MockMatchUCMockRecorder struct {
	mock *MockMatchUC
}

// NewMockMatchUC creates a new mock instance.
func NewMockMatchUC(ctrl *gomock.Controller) *MockMatchUC {
	mock := &MockMatchUC{ctrl: ctrl}
	mock.recorder = &MockMatchUCMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMatchUC) EXPECT() *MockMatchUCMockRecorder {
	return m.recorder
}

// ConfirmMatchStatus mocks base method.
func (m *MockMatchUC) ConfirmMatchStatus(arg0, arg1 string, arg2 bool, arg3 models.MatchStatus) (models.MatchProposal, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConfirmMatchStatus", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(models.MatchProposal)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConfirmMatchStatus indicates an expected call of ConfirmMatchStatus.
func (mr *MockMatchUCMockRecorder) ConfirmMatchStatus(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConfirmMatchStatus", reflect.TypeOf((*MockMatchUC)(nil).ConfirmMatchStatus), arg0, arg1, arg2, arg3)
}

// GetMatch mocks base method.
func (m *MockMatchUC) GetMatch(arg0 context.Context, arg1 string) (*models.Match, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMatch", arg0, arg1)
	ret0, _ := ret[0].(*models.Match)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMatch indicates an expected call of GetMatch.
func (mr *MockMatchUCMockRecorder) GetMatch(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMatch", reflect.TypeOf((*MockMatchUC)(nil).GetMatch), arg0, arg1)
}

// GetPendingMatch mocks base method.
func (m *MockMatchUC) GetPendingMatch(arg0 context.Context, arg1 string) (*models.Match, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPendingMatch", arg0, arg1)
	ret0, _ := ret[0].(*models.Match)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPendingMatch indicates an expected call of GetPendingMatch.
func (mr *MockMatchUCMockRecorder) GetPendingMatch(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPendingMatch", reflect.TypeOf((*MockMatchUC)(nil).GetPendingMatch), arg0, arg1)
}

// HandleBeaconEvent mocks base method.
func (m *MockMatchUC) HandleBeaconEvent(arg0 models.BeaconEvent) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleBeaconEvent", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleBeaconEvent indicates an expected call of HandleBeaconEvent.
func (mr *MockMatchUCMockRecorder) HandleBeaconEvent(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleBeaconEvent", reflect.TypeOf((*MockMatchUC)(nil).HandleBeaconEvent), arg0)
}
