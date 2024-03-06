// Code generated by mockery v2.21.1. DO NOT EDIT.

package scaler

import (
	metal "github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	mock "github.com/stretchr/testify/mock"
)

// MockMachineManager is an autogenerated mock type for the MachineManager type
type MockMachineManager struct {
	mock.Mock
}

// AllMachines provides a mock function with given fields:
func (_m *MockMachineManager) AllMachines() (metal.Machines, error) {
	ret := _m.Called()

	var r0 metal.Machines
	var r1 error
	if rf, ok := ret.Get(0).(func() (metal.Machines, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() metal.Machines); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metal.Machines)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PowerOn provides a mock function with given fields: m
func (_m *MockMachineManager) PowerOn(m *metal.Machine) error {
	ret := _m.Called(m)

	var r0 error
	if rf, ok := ret.Get(0).(func(*metal.Machine) error); ok {
		r0 = rf(m)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Shutdown provides a mock function with given fields: m
func (_m *MockMachineManager) Shutdown(m *metal.Machine) error {
	ret := _m.Called(m)

	var r0 error
	if rf, ok := ret.Get(0).(func(*metal.Machine) error); ok {
		r0 = rf(m)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ShutdownMachines provides a mock function with given fields:
func (_m *MockMachineManager) ShutdownMachines() (metal.Machines, error) {
	ret := _m.Called()

	var r0 metal.Machines
	var r1 error
	if rf, ok := ret.Get(0).(func() (metal.Machines, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() metal.Machines); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metal.Machines)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WaitingMachines provides a mock function with given fields:
func (_m *MockMachineManager) WaitingMachines() (metal.Machines, error) {
	ret := _m.Called()

	var r0 metal.Machines
	var r1 error
	if rf, ok := ret.Get(0).(func() (metal.Machines, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() metal.Machines); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metal.Machines)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockMachineManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockMachineManager creates a new instance of MockMachineManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockMachineManager(t mockConstructorTestingTNewMockMachineManager) *MockMachineManager {
	mock := &MockMachineManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
