// Code generated by mockery v2.40.2. DO NOT EDIT.

package testdata

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockStructyThing is an autogenerated mock type for the StructyThing type
type MockStructyThing struct {
	mock.Mock
}

type MockStructyThing_Expecter struct {
	mock *mock.Mock
}

func (_m *MockStructyThing) EXPECT() *MockStructyThing_Expecter {
	return &MockStructyThing_Expecter{mock: &_m.Mock}
}

// DoThing provides a mock function with given fields: _a0, _a1
func (_m *MockStructyThing) DoThing(_a0 context.Context, _a1 *Struct) (*Struct, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for DoThing")
	}

	var r0 *Struct
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *Struct) (*Struct, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *Struct) *Struct); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Struct)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *Struct) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockStructyThing_DoThing_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DoThing'
type MockStructyThing_DoThing_Call struct {
	*mock.Call
}

// DoThing is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *Struct
func (_e *MockStructyThing_Expecter) DoThing(_a0 interface{}, _a1 interface{}) *MockStructyThing_DoThing_Call {
	return &MockStructyThing_DoThing_Call{Call: _e.mock.On("DoThing", _a0, _a1)}
}

func (_c *MockStructyThing_DoThing_Call) Run(run func(_a0 context.Context, _a1 *Struct)) *MockStructyThing_DoThing_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*Struct))
	})
	return _c
}

func (_c *MockStructyThing_DoThing_Call) Return(_a0 *Struct, _a1 error) *MockStructyThing_DoThing_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStructyThing_DoThing_Call) RunAndReturn(run func(context.Context, *Struct) (*Struct, error)) *MockStructyThing_DoThing_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockStructyThing creates a new instance of MockStructyThing. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockStructyThing(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockStructyThing {
	mock := &MockStructyThing{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
