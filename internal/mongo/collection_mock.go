// Code generated by MockGen. DO NOT EDIT.
// Source: internal/mongo/collection.go

package mongo

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	options "go.mongodb.org/mongo-driver/mongo/options"
	reflect "reflect"
)

// MockDriverDatabase is a mock of DriverDatabase interface
type MockDriverDatabase struct {
	ctrl     *gomock.Controller
	recorder *MockDriverDatabaseMockRecorder
}

// MockDriverDatabaseMockRecorder is the mock recorder for MockDriverDatabase
type MockDriverDatabaseMockRecorder struct {
	mock *MockDriverDatabase
}

// NewMockDriverDatabase creates a new mock instance
func NewMockDriverDatabase(ctrl *gomock.Controller) *MockDriverDatabase {
	mock := &MockDriverDatabase{ctrl: ctrl}
	mock.recorder = &MockDriverDatabaseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (_m *MockDriverDatabase) EXPECT() *MockDriverDatabaseMockRecorder {
	return _m.recorder
}

// Name mocks base method
func (_m *MockDriverDatabase) Name() string {
	ret := _m.ctrl.Call(_m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (_mr *MockDriverDatabaseMockRecorder) Name() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Name", reflect.TypeOf((*MockDriverDatabase)(nil).Name))
}

// MockCollectionAdapter is a mock of CollectionAdapter interface
type MockCollectionAdapter struct {
	ctrl     *gomock.Controller
	recorder *MockCollectionAdapterMockRecorder
}

// MockCollectionAdapterMockRecorder is the mock recorder for MockCollectionAdapter
type MockCollectionAdapterMockRecorder struct {
	mock *MockCollectionAdapter
}

// NewMockCollectionAdapter creates a new mock instance
func NewMockCollectionAdapter(ctrl *gomock.Controller) *MockCollectionAdapter {
	mock := &MockCollectionAdapter{ctrl: ctrl}
	mock.recorder = &MockCollectionAdapterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (_m *MockCollectionAdapter) EXPECT() *MockCollectionAdapterMockRecorder {
	return _m.recorder
}

// Aggregate mocks base method
func (_m *MockCollectionAdapter) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (DriverCursor, error) {
	_s := []interface{}{ctx, pipeline}
	for _, _x := range opts {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "Aggregate", _s...)
	ret0, _ := ret[0].(DriverCursor)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Aggregate indicates an expected call of Aggregate
func (_mr *MockCollectionAdapterMockRecorder) Aggregate(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Aggregate", reflect.TypeOf((*MockCollectionAdapter)(nil).Aggregate), _s...)
}

// Database mocks base method
func (_m *MockCollectionAdapter) Database() DriverDatabase {
	ret := _m.ctrl.Call(_m, "Database")
	ret0, _ := ret[0].(DriverDatabase)
	return ret0
}

// Database indicates an expected call of Database
func (_mr *MockCollectionAdapterMockRecorder) Database() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Database", reflect.TypeOf((*MockCollectionAdapter)(nil).Database))
}

// Watch mocks base method
func (_m *MockCollectionAdapter) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (DriverCursor, error) {
	_s := []interface{}{ctx, pipeline}
	for _, _x := range opts {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "Watch", _s...)
	ret0, _ := ret[0].(DriverCursor)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch
func (_mr *MockCollectionAdapterMockRecorder) Watch(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Watch", reflect.TypeOf((*MockCollectionAdapter)(nil).Watch), _s...)
}

// Name mocks base method
func (_m *MockCollectionAdapter) Name() string {
	ret := _m.ctrl.Call(_m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (_mr *MockCollectionAdapterMockRecorder) Name() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Name", reflect.TypeOf((*MockCollectionAdapter)(nil).Name))
}