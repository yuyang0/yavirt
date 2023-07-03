package guest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt/guest/mocks"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/local"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	storemocks "github.com/projecteru2/yavirt/pkg/store/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/projecteru2/yavirt/pkg/utils"
	utilmocks "github.com/projecteru2/yavirt/pkg/utils/mocks"
)

const (
	MAX_RETRIES = 3
	INTERVAL    = 200 * time.Millisecond
)

func init() {
	idgen.Setup(0, time.Now())
}

func TestCreate_WithExtVolumes(t *testing.T) {
	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	var genvol = func(id int64, cap int64) volume.Volume {
		vol, err := local.NewDataVolume("/data", cap)
		assert.NilErr(t, err)
		return vol
	}
	var extVols = []volume.Volume{genvol(1, utils.GB*10), genvol(2, utils.GB*20)}
	guest.AppendVols(extVols...)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusPending))

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)

	bot.On("Create").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	assert.NilErr(t, guest.Create())
	assert.Equal(t, meta.StatusCreating, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusCreating))
}

func TestLifecycle(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	assert.Equal(t, meta.StatusPending, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusPending))

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	bot.On("Create").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.Create())
	assert.Equal(t, meta.StatusCreating, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusCreating))

	bot.On("Boot", ctx).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.Start(ctx))
	assert.Equal(t, meta.StatusRunning, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusRunning))

	bot.On("Shutdown", ctx, mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.Stop(ctx, true))
	assert.Equal(t, meta.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusStopped))

	assert.NilErr(t, guest.Resize(guest.CPU, guest.Memory, []volume.Volume{}))
	assert.Equal(t, meta.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusStopped))

	bot.On("Capture", mock.Anything, mock.Anything).Return(image.NewUserImage("anrs", "aa", 1024), nil).Once()
	bot.On("Close").Return(nil).Once()
	sto.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(int64(0), nil)
	_, err := guest.Capture("anrs", "aa", true)
	assert.NilErr(t, err)
	assert.Equal(t, meta.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusStopped))

	bot.On("Boot", ctx).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.Start(ctx))
	assert.Equal(t, meta.StatusRunning, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusRunning))

	bot.On("Shutdown", ctx, mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.Stop(ctx, true))
	assert.Equal(t, meta.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusStopped))

	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil).Once()

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	sto.On("NewMutex", mock.Anything).Return(mutex, nil).Once()
	sto.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	done, err := guest.Destroy(ctx, false)
	assert.NilErr(t, err)
	assert.NilErr(t, <-done)
}

func checkVolsStatus(t *testing.T, expSt string) func(int, volume.Volume) bool {
	return func(_ int, v volume.Volume) bool {
		assert.Equal(t, expSt, v.GetStatus())
		return true
	}
}

func TestLifecycle_InvalidStatus(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	guest.Status = meta.StatusDestroyed
	assert.Err(t, guest.Create())
	assert.Err(t, guest.Stop(ctx, false))
	assert.Err(t, guest.Start(ctx))

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)

	sto.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(int64(0), nil)
	_, err := guest.Capture("anrs", "aa", true)
	assert.Err(t, err)

	guest.Status = meta.StatusResizing
	done, err := guest.Destroy(ctx, false)
	assert.Err(t, err)
	assert.Nil(t, done)

	guest.Status = meta.StatusPending
	assert.Err(t, guest.Resize(guest.CPU, guest.Memory, []volume.Volume{}))
}

func TestSyncState(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	guest.Status = meta.StatusCreating
	bot.On("Create").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	assert.NilErr(t, guest.SyncState(ctx))

	guest.Status = meta.StatusDestroying
	guest.rangeVolumes(func(_ int, v volume.Volume) bool {
		v.SetStatus(meta.StatusDestroying, true)
		return true
	})

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	sto.On("NewMutex", mock.Anything).Return(mutex, nil).Once()
	sto.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	assert.NilErr(t, guest.SyncState(ctx))

	guest.Status = meta.StatusStopping
	guest.rangeVolumes(func(_ int, v volume.Volume) bool {
		v.SetStatus(meta.StatusStopping, true)
		return true
	})
	bot.On("Shutdown", ctx, mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.SyncState(ctx))

	guest.Status = meta.StatusStarting
	guest.rangeVolumes(func(_ int, v volume.Volume) bool {
		v.SetStatus(meta.StatusStarting, true)
		return true
	})
	bot.On("Boot", ctx).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.SyncState(ctx))
}

func TestForceDestroy(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	sto, stoCancel := storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)
	sto.On("NewMutex", mock.Anything).Return(mutex, nil).Once()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	sto.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Status = meta.StatusRunning
	bot.On("Shutdown", ctx, true).Return(nil).Once()
	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil)
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	done, err := guest.Destroy(ctx, true)
	assert.NilErr(t, err)
	assert.NilErr(t, <-done)
}

func mockMutex() *utilmocks.Locker {
	var unlock utils.Unlocker = func(context.Context) error {
		return nil
	}

	mutex := &utilmocks.Locker{}
	mutex.On("Lock", mock.Anything).Return(unlock, nil)

	return mutex
}

func TestSyncStateSkipsRunning(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainRunning, nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	guest.Status = meta.StatusRunning
	assert.NilErr(t, guest.SyncState(ctx))
}

func TestAmplifyOrigVols_HostDirMount(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	volmod, err := local.NewDataVolume("/tmp:/data", utils.GB)
	assert.NilErr(t, err)

	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	bot.On("AmplifyVolume", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Vols = volume.Volumes{volmod}
	vol, err := local.NewDataVolume("/tmp:/data", utils.GB*10)
	assert.Nil(t, err)
	mnt := map[string]volume.Volume{vol.GetMountDir(): vol}
	assert.NilErr(t, guest.amplifyOrigVols(mnt))
}

func TestAttachVolumes_CheckVolumeModel(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("AttachVolume", mock.Anything, mock.Anything).Return(nil, nil).Once()

	sto, cancel := storemocks.Mock()
	defer sto.AssertExpectations(t)
	defer cancel()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Status = meta.StatusRunning
	guest.HostName = "lo"
	guest.ID = "guestid"
	vol, err := local.NewDataVolume("/data", utils.GB)
	assert.Nil(t, err)
	vols := []volume.Volume{vol}
	assert.NilErr(t, guest.Resize(guest.CPU, guest.Memory, vols))

	volmod := guest.Vols[1] // guest.Vols[0] is the sys volume.
	assert.True(t, len(volmod.GetID()) > 0)
	assert.Equal(t, guest.Status, volmod.GetStatus())
	assert.Equal(t, "/data", volmod.GetMountDir())
	// assert.Equal(t, "", volmod.HostDir)
	assert.Equal(t, utils.GB, volmod.GetSize())
	assert.Equal(t, guest.HostName, volmod.GetHostname())
	assert.Equal(t, guest.ID, volmod.GetGuestID())
}

func TestAttachVolumes_Rollback(t *testing.T) {
	var rolled bool
	rollback := func() { rolled = true }

	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("AttachVolume", mock.Anything, mock.Anything).Return(rollback, nil).Once()

	sto, cancel := storemocks.Mock()
	defer sto.AssertExpectations(t)
	defer cancel()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("faked-error")).Once()

	guest.Status = meta.StatusRunning
	vol, err := local.NewDataVolume("/data", utils.GB)
	assert.Nil(t, err)
	vols := []volume.Volume{vol}
	assert.Err(t, guest.Resize(guest.CPU, guest.Memory, vols))
	assert.Equal(t, 1, guest.Vols.Len())
	// assert.Equal(t, models.VolSysType, guest.Vols[0].Type)
	assert.True(t, rolled)
}

func TestCannotShrinkOrigVolumes(t *testing.T) {
	testcases := []struct {
		exists   string
		resizing string
	}{
		{"/data", "/data"},
		{"/data", "/tmp2:/data"},
		{"/tmp:/data", "/data"},
		{"/tmp:/data", "/tmp2:/data"},
	}

	for _, tc := range testcases {
		guest, _ := newMockedGuest(t)
		volmod, err := local.NewDataVolume(tc.exists, utils.GB*2)
		assert.NilErr(t, err)
		assert.NilErr(t, guest.AppendVols(volmod))

		guest.Status = meta.StatusRunning
		vol, err := local.NewDataVolume(tc.resizing, utils.GB)
		assert.Nil(t, err)
		vols := []volume.Volume{vol}
		assert.True(t, errors.Contain(
			guest.Resize(guest.CPU, guest.Memory, vols),
			errors.ErrCannotShrinkVolume,
		))
	}
}

func newMockedGuest(t *testing.T) (*Guest, *mocks.Bot) {
	var bot = &mocks.Bot{}

	gmod, err := models.NewGuest(models.NewHost(), image.NewSysImage())
	assert.NilErr(t, err)

	var guest = &Guest{
		Guest:  gmod,
		newBot: func(g *Guest) (Bot, error) { return bot, nil },
	}

	return guest, bot
}
