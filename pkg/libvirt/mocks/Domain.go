// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	libvirt_go "github.com/libvirt/libvirt-go"
	mock "github.com/stretchr/testify/mock"
)

// Domain is an autogenerated mock type for the Domain type
type Domain struct {
	mock.Mock
}

// AmplifyVolume provides a mock function with given fields: filepath, cap
func (_m *Domain) AmplifyVolume(filepath string, cap uint64) error {
	ret := _m.Called(filepath, cap)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, uint64) error); ok {
		r0 = rf(filepath, cap)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AttachVolume provides a mock function with given fields: xml
func (_m *Domain) AttachVolume(xml string) (libvirt_go.DomainState, error) {
	ret := _m.Called(xml)

	var r0 libvirt_go.DomainState
	if rf, ok := ret.Get(0).(func(string) libvirt_go.DomainState); ok {
		r0 = rf(xml)
	} else {
		r0 = ret.Get(0).(libvirt_go.DomainState)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(xml)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Create provides a mock function with given fields:
func (_m *Domain) Create() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Destroy provides a mock function with given fields:
func (_m *Domain) Destroy() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DestroyFlags provides a mock function with given fields: flags
func (_m *Domain) DestroyFlags(flags libvirt_go.DomainDestroyFlags) error {
	ret := _m.Called(flags)

	var r0 error
	if rf, ok := ret.Get(0).(func(libvirt_go.DomainDestroyFlags) error); ok {
		r0 = rf(flags)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Free provides a mock function with given fields:
func (_m *Domain) Free() {
	_m.Called()
}

// GetInfo provides a mock function with given fields:
func (_m *Domain) GetInfo() (*libvirt_go.DomainInfo, error) {
	ret := _m.Called()

	var r0 *libvirt_go.DomainInfo
	if rf, ok := ret.Get(0).(func() *libvirt_go.DomainInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*libvirt_go.DomainInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetName provides a mock function with given fields:
func (_m *Domain) GetName() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetState provides a mock function with given fields:
func (_m *Domain) GetState() (libvirt_go.DomainState, error) {
	ret := _m.Called()

	var r0 libvirt_go.DomainState
	if rf, ok := ret.Get(0).(func() libvirt_go.DomainState); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(libvirt_go.DomainState)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUUIDString provides a mock function with given fields:
func (_m *Domain) GetUUIDString() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetXMLDesc provides a mock function with given fields: flags
func (_m *Domain) GetXMLDesc(flags libvirt_go.DomainXMLFlags) (string, error) {
	ret := _m.Called(flags)

	var r0 string
	if rf, ok := ret.Get(0).(func(libvirt_go.DomainXMLFlags) string); ok {
		r0 = rf(flags)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(libvirt_go.DomainXMLFlags) error); ok {
		r1 = rf(flags)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Resume provides a mock function with given fields:
func (_m *Domain) Resume() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetMemoryFlags provides a mock function with given fields: memory, flags
func (_m *Domain) SetMemoryFlags(memory uint64, flags libvirt_go.DomainMemoryModFlags) error {
	ret := _m.Called(memory, flags)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint64, libvirt_go.DomainMemoryModFlags) error); ok {
		r0 = rf(memory, flags)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetVcpusFlags provides a mock function with given fields: vcpu, flags
func (_m *Domain) SetVcpusFlags(vcpu uint, flags libvirt_go.DomainVcpuFlags) error {
	ret := _m.Called(vcpu, flags)

	var r0 error
	if rf, ok := ret.Get(0).(func(uint, libvirt_go.DomainVcpuFlags) error); ok {
		r0 = rf(vcpu, flags)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ShutdownFlags provides a mock function with given fields: flags
func (_m *Domain) ShutdownFlags(flags libvirt_go.DomainShutdownFlags) error {
	ret := _m.Called(flags)

	var r0 error
	if rf, ok := ret.Get(0).(func(libvirt_go.DomainShutdownFlags) error); ok {
		r0 = rf(flags)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Suspend provides a mock function with given fields:
func (_m *Domain) Suspend() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UndefineFlags provides a mock function with given fields: flags
func (_m *Domain) UndefineFlags(flags libvirt_go.DomainUndefineFlagsValues) error {
	ret := _m.Called(flags)

	var r0 error
	if rf, ok := ret.Get(0).(func(libvirt_go.DomainUndefineFlagsValues) error); ok {
		r0 = rf(flags)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}