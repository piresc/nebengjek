// This file contains manual additions to the automatically generated mock
package mocks

import (
	"context"
	"time"

	"reflect"

	"github.com/golang/mock/gomock"
)

// Note: GetPendingMatchByID was previously defined here but has been moved to the auto-generated mock_repository.go file

// StoreIDMapping mocks base method.
func (m *MockMatchRepo) StoreIDMapping(ctx context.Context, key string, value string, expiration time.Duration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreIDMapping", ctx, key, value, expiration)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreIDMapping indicates an expected call of StoreIDMapping.
func (mr *MockMatchRepoMockRecorder) StoreIDMapping(ctx, key, value, expiration interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreIDMapping", reflect.TypeOf((*MockMatchRepo)(nil).StoreIDMapping), ctx, key, value, expiration)
}

// GetIDMapping mocks base method.
func (m *MockMatchRepo) GetIDMapping(ctx context.Context, originalID string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIDMapping", ctx, originalID)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIDMapping indicates an expected call of GetIDMapping.
func (mr *MockMatchRepoMockRecorder) GetIDMapping(ctx, originalID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIDMapping", reflect.TypeOf((*MockMatchRepo)(nil).GetIDMapping), ctx, originalID)
}
