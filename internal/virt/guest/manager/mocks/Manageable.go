// Code generated by mockery v2.26.1. DO NOT EDIT.

package mocks

import (
	io "io"

	guest "github.com/projecteru2/yavirt/internal/virt/guest"

	manager "github.com/projecteru2/yavirt/internal/virt/guest/manager"

	mock "github.com/stretchr/testify/mock"

	models "github.com/projecteru2/yavirt/internal/models"

	types "github.com/projecteru2/yavirt/internal/virt/types"

	virt "github.com/projecteru2/yavirt/internal/virt"
)

// Manageable is an autogenerated mock type for the Manageable type
type Manageable struct {
	mock.Mock
}

// AttachConsole provides a mock function with given fields: ctx, id, stream, flags
func (_m *Manageable) AttachConsole(ctx virt.Context, id string, stream io.ReadWriteCloser, flags types.OpenConsoleFlags) error {
	ret := _m.Called(ctx, id, stream, flags)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, io.ReadWriteCloser, types.OpenConsoleFlags) error); ok {
		r0 = rf(ctx, id, stream, flags)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Capture provides a mock function with given fields: ctx, guestID, user, name, overridden
func (_m *Manageable) Capture(ctx virt.Context, guestID string, user string, name string, overridden bool) (*models.UserImage, error) {
	ret := _m.Called(ctx, guestID, user, name, overridden)

	var r0 *models.UserImage
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, string, bool) (*models.UserImage, error)); ok {
		return rf(ctx, guestID, user, name, overridden)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, string, bool) *models.UserImage); ok {
		r0 = rf(ctx, guestID, user, name, overridden)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.UserImage)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string, string, string, bool) error); ok {
		r1 = rf(ctx, guestID, user, name, overridden)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Cat provides a mock function with given fields: ctx, id, path, dest
func (_m *Manageable) Cat(ctx virt.Context, id string, path string, dest io.WriteCloser) error {
	ret := _m.Called(ctx, id, path, dest)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, io.WriteCloser) error); ok {
		r0 = rf(ctx, id, path, dest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CommitSnapshot provides a mock function with given fields: ctx, id, volID, snapID
func (_m *Manageable) CommitSnapshot(ctx virt.Context, id string, volID string, snapID string) error {
	ret := _m.Called(ctx, id, volID, snapID)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, string) error); ok {
		r0 = rf(ctx, id, volID, snapID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CommitSnapshotByDay provides a mock function with given fields: ctx, id, volID, day
func (_m *Manageable) CommitSnapshotByDay(ctx virt.Context, id string, volID string, day int) error {
	ret := _m.Called(ctx, id, volID, day)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, int) error); ok {
		r0 = rf(ctx, id, volID, day)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConnectExtraNetwork provides a mock function with given fields: ctx, id, network, ipv4
func (_m *Manageable) ConnectExtraNetwork(ctx virt.Context, id string, network string, ipv4 string) (string, error) {
	ret := _m.Called(ctx, id, network, ipv4)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, string) (string, error)); ok {
		return rf(ctx, id, network, ipv4)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, string) string); ok {
		r0 = rf(ctx, id, network, ipv4)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string, string, string) error); ok {
		r1 = rf(ctx, id, network, ipv4)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CopyToGuest provides a mock function with given fields: ctx, id, dest, content, override
func (_m *Manageable) CopyToGuest(ctx virt.Context, id string, dest string, content chan []byte, override bool) error {
	ret := _m.Called(ctx, id, dest, content, override)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, chan []byte, bool) error); ok {
		r0 = rf(ctx, id, dest, content, override)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Create provides a mock function with given fields: ctx, opts, host, vols
func (_m *Manageable) Create(ctx virt.Context, opts types.GuestCreateOption, host *models.Host, vols []*models.Volume) (*guest.Guest, error) {
	ret := _m.Called(ctx, opts, host, vols)

	var r0 *guest.Guest
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, types.GuestCreateOption, *models.Host, []*models.Volume) (*guest.Guest, error)); ok {
		return rf(ctx, opts, host, vols)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, types.GuestCreateOption, *models.Host, []*models.Volume) *guest.Guest); ok {
		r0 = rf(ctx, opts, host, vols)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*guest.Guest)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, types.GuestCreateOption, *models.Host, []*models.Volume) error); ok {
		r1 = rf(ctx, opts, host, vols)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateSnapshot provides a mock function with given fields: ctx, id, volID
func (_m *Manageable) CreateSnapshot(ctx virt.Context, id string, volID string) error {
	ret := _m.Called(ctx, id, volID)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string) error); ok {
		r0 = rf(ctx, id, volID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Destroy provides a mock function with given fields: ctx, id, force
func (_m *Manageable) Destroy(ctx virt.Context, id string, force bool) (<-chan error, error) {
	ret := _m.Called(ctx, id, force)

	var r0 <-chan error
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, bool) (<-chan error, error)); ok {
		return rf(ctx, id, force)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string, bool) <-chan error); ok {
		r0 = rf(ctx, id, force)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan error)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string, bool) error); ok {
		r1 = rf(ctx, id, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DigestImage provides a mock function with given fields: ctx, name, local
func (_m *Manageable) DigestImage(ctx virt.Context, name string, local bool) ([]string, error) {
	ret := _m.Called(ctx, name, local)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, bool) ([]string, error)); ok {
		return rf(ctx, name, local)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string, bool) []string); ok {
		r0 = rf(ctx, name, local)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string, bool) error); ok {
		r1 = rf(ctx, name, local)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DisconnectExtraNetwork provides a mock function with given fields: ctx, id, network
func (_m *Manageable) DisconnectExtraNetwork(ctx virt.Context, id string, network string) error {
	ret := _m.Called(ctx, id, network)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string) error); ok {
		r0 = rf(ctx, id, network)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ExecuteCommand provides a mock function with given fields: ctx, id, commands
func (_m *Manageable) ExecuteCommand(ctx virt.Context, id string, commands []string) ([]byte, int, int, error) {
	ret := _m.Called(ctx, id, commands)

	var r0 []byte
	var r1 int
	var r2 int
	var r3 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, []string) ([]byte, int, int, error)); ok {
		return rf(ctx, id, commands)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string, []string) []byte); ok {
		r0 = rf(ctx, id, commands)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string, []string) int); ok {
		r1 = rf(ctx, id, commands)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(virt.Context, string, []string) int); ok {
		r2 = rf(ctx, id, commands)
	} else {
		r2 = ret.Get(2).(int)
	}

	if rf, ok := ret.Get(3).(func(virt.Context, string, []string) error); ok {
		r3 = rf(ctx, id, commands)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// ListImage provides a mock function with given fields: ctx, filter
func (_m *Manageable) ListImage(ctx virt.Context, filter string) ([]models.Image, error) {
	ret := _m.Called(ctx, filter)

	var r0 []models.Image
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string) ([]models.Image, error)); ok {
		return rf(ctx, filter)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string) []models.Image); ok {
		r0 = rf(ctx, filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Image)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string) error); ok {
		r1 = rf(ctx, filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListLocalIDs provides a mock function with given fields: ctx, onlyERU
func (_m *Manageable) ListLocalIDs(ctx virt.Context, onlyERU bool) ([]string, error) {
	ret := _m.Called(ctx, onlyERU)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, bool) ([]string, error)); ok {
		return rf(ctx, onlyERU)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, bool) []string); ok {
		r0 = rf(ctx, onlyERU)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, bool) error); ok {
		r1 = rf(ctx, onlyERU)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListSnapshot provides a mock function with given fields: ctx, guestID, volID
func (_m *Manageable) ListSnapshot(ctx virt.Context, guestID string, volID string) (map[*models.Volume]models.Snapshots, error) {
	ret := _m.Called(ctx, guestID, volID)

	var r0 map[*models.Volume]models.Snapshots
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string) (map[*models.Volume]models.Snapshots, error)); ok {
		return rf(ctx, guestID, volID)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string, string) map[*models.Volume]models.Snapshots); ok {
		r0 = rf(ctx, guestID, volID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[*models.Volume]models.Snapshots)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string, string) error); ok {
		r1 = rf(ctx, guestID, volID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Load provides a mock function with given fields: ctx, id
func (_m *Manageable) Load(ctx virt.Context, id string) (*guest.Guest, error) {
	ret := _m.Called(ctx, id)

	var r0 *guest.Guest
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string) (*guest.Guest, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string) *guest.Guest); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*guest.Guest)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadUUID provides a mock function with given fields: ctx, id
func (_m *Manageable) LoadUUID(ctx virt.Context, id string) (string, error) {
	ret := _m.Called(ctx, id)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string) (string, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string) string); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Log provides a mock function with given fields: ctx, id, logPath, n, dest
func (_m *Manageable) Log(ctx virt.Context, id string, logPath string, n int, dest io.WriteCloser) error {
	ret := _m.Called(ctx, id, logPath, n, dest)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, int, io.WriteCloser) error); ok {
		r0 = rf(ctx, id, logPath, n, dest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewWatcher provides a mock function with given fields:
func (_m *Manageable) NewWatcher() (*manager.Watcher, error) {
	ret := _m.Called()

	var r0 *manager.Watcher
	var r1 error
	if rf, ok := ret.Get(0).(func() (*manager.Watcher, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *manager.Watcher); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*manager.Watcher)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveImage provides a mock function with given fields: ctx, imageName, user, force, prune
func (_m *Manageable) RemoveImage(ctx virt.Context, imageName string, user string, force bool, prune bool) ([]string, error) {
	ret := _m.Called(ctx, imageName, user, force, prune)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, bool, bool) ([]string, error)); ok {
		return rf(ctx, imageName, user, force, prune)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, bool, bool) []string); ok {
		r0 = rf(ctx, imageName, user, force, prune)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string, string, bool, bool) error); ok {
		r1 = rf(ctx, imageName, user, force, prune)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Resize provides a mock function with given fields: ctx, id, cpu, mem, vols
func (_m *Manageable) Resize(ctx virt.Context, id string, cpu int, mem int64, vols map[string]int64) error {
	ret := _m.Called(ctx, id, cpu, mem, vols)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, int, int64, map[string]int64) error); ok {
		r0 = rf(ctx, id, cpu, mem, vols)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ResizeConsoleWindow provides a mock function with given fields: ctx, id, height, width
func (_m *Manageable) ResizeConsoleWindow(ctx virt.Context, id string, height uint, width uint) error {
	ret := _m.Called(ctx, id, height, width)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, uint, uint) error); ok {
		r0 = rf(ctx, id, height, width)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RestoreSnapshot provides a mock function with given fields: ctx, id, volID, snapID
func (_m *Manageable) RestoreSnapshot(ctx virt.Context, id string, volID string, snapID string) error {
	ret := _m.Called(ctx, id, volID, snapID)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, string, string) error); ok {
		r0 = rf(ctx, id, volID, snapID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Resume provides a mock function with given fields: ctx, id
func (_m *Manageable) Resume(ctx virt.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields: ctx, id
func (_m *Manageable) Start(ctx virt.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StartWatch provides a mock function with given fields:
func (_m *Manageable) StartWatch() {
	_m.Called()
}

// Stop provides a mock function with given fields: ctx, id, force
func (_m *Manageable) Stop(ctx virt.Context, id string, force bool) error {
	ret := _m.Called(ctx, id, force)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, bool) error); ok {
		r0 = rf(ctx, id, force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Suspend provides a mock function with given fields: ctx, id
func (_m *Manageable) Suspend(ctx virt.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(virt.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Wait provides a mock function with given fields: ctx, id, block
func (_m *Manageable) Wait(ctx virt.Context, id string, block bool) (string, int, error) {
	ret := _m.Called(ctx, id, block)

	var r0 string
	var r1 int
	var r2 error
	if rf, ok := ret.Get(0).(func(virt.Context, string, bool) (string, int, error)); ok {
		return rf(ctx, id, block)
	}
	if rf, ok := ret.Get(0).(func(virt.Context, string, bool) string); ok {
		r0 = rf(ctx, id, block)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(virt.Context, string, bool) int); ok {
		r1 = rf(ctx, id, block)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(virt.Context, string, bool) error); ok {
		r2 = rf(ctx, id, block)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

type mockConstructorTestingTNewManageable interface {
	mock.TestingT
	Cleanup(func())
}

// NewManageable creates a new instance of Manageable. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewManageable(t mockConstructorTestingTNewManageable) *Manageable {
	mock := &Manageable{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
