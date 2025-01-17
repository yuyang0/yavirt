// Code generated by mockery v2.26.1. DO NOT EDIT.

package mocks

import (
	context "context"
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// File is an autogenerated mock type for the File type
type File struct {
	mock.Mock
}

// Close provides a mock function with given fields: ctx
func (_m *File) Close(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CopyTo provides a mock function with given fields: ctx, dst
func (_m *File) CopyTo(ctx context.Context, dst io.Writer) (int, error) {
	ret := _m.Called(ctx, dst)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, io.Writer) (int, error)); ok {
		return rf(ctx, dst)
	}
	if rf, ok := ret.Get(0).(func(context.Context, io.Writer) int); ok {
		r0 = rf(ctx, dst)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, io.Writer) error); ok {
		r1 = rf(ctx, dst)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Flush provides a mock function with given fields: ctx
func (_m *File) Flush(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Open provides a mock function with given fields: ctx
func (_m *File) Open(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Read provides a mock function with given fields: ctx, p
func (_m *File) Read(ctx context.Context, p []byte) (int, error) {
	ret := _m.Called(ctx, p)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte) (int, error)); ok {
		return rf(ctx, p)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte) int); ok {
		r0 = rf(ctx, p)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte) error); ok {
		r1 = rf(ctx, p)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReadAt provides a mock function with given fields: ctx, dest, pos
func (_m *File) ReadAt(ctx context.Context, dest []byte, pos int) (int, error) {
	ret := _m.Called(ctx, dest, pos)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, int) (int, error)); ok {
		return rf(ctx, dest, pos)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, int) int); ok {
		r0 = rf(ctx, dest, pos)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, int) error); ok {
		r1 = rf(ctx, dest, pos)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Seek provides a mock function with given fields: ctx, offset, whence
func (_m *File) Seek(ctx context.Context, offset int, whence int) (int, error) {
	ret := _m.Called(ctx, offset, whence)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int, int) (int, error)); ok {
		return rf(ctx, offset, whence)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int, int) int); ok {
		r0 = rf(ctx, offset, whence)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, int, int) error); ok {
		r1 = rf(ctx, offset, whence)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Tail provides a mock function with given fields: ctx, n
func (_m *File) Tail(ctx context.Context, n int) ([]byte, error) {
	ret := _m.Called(ctx, n)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int) ([]byte, error)); ok {
		return rf(ctx, n)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int) []byte); ok {
		r0 = rf(ctx, n)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(ctx, n)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Write provides a mock function with given fields: ctx, p
func (_m *File) Write(ctx context.Context, p []byte) (int, error) {
	ret := _m.Called(ctx, p)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte) (int, error)); ok {
		return rf(ctx, p)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte) int); ok {
		r0 = rf(ctx, p)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte) error); ok {
		r1 = rf(ctx, p)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WriteLine provides a mock function with given fields: ctx, p
func (_m *File) WriteLine(ctx context.Context, p []byte) (int, error) {
	ret := _m.Called(ctx, p)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte) (int, error)); ok {
		return rf(ctx, p)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte) int); ok {
		r0 = rf(ctx, p)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte) error); ok {
		r1 = rf(ctx, p)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewFile interface {
	mock.TestingT
	Cleanup(func())
}

// NewFile creates a new instance of File. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewFile(t mockConstructorTestingTNewFile) *File {
	mock := &File{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
