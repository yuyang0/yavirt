package guest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Guest .
type Guest struct {
	*models.Guest

	ctx context.Context

	newBot func(*Guest) (Bot, error)
}

// New initializes a new Guest.
func New(ctx context.Context, g *models.Guest) *Guest {
	return &Guest{
		Guest:  g,
		ctx:    ctx,
		newBot: newVirtGuest,
	}
}

// ListLocalIDs lists all local guest domain names.
func ListLocalIDs(context.Context) ([]string, error) {
	virt, err := connectSystemLibvirt()
	if err != nil {
		return nil, errors.Trace(err)
	}

	defer func() {
		if _, ce := virt.Close(); ce != nil {
			log.ErrorStack(ce)
		}
	}()

	return virt.ListDomainsNames()
}

// Load .
func (g *Guest) Load() error {
	host, err := models.LoadHost()
	if err != nil {
		return errors.Trace(err)
	}

	hand, err := g.NetworkHandler(host)
	if err != nil {
		return errors.Trace(err)
	}

	if err := g.Guest.Load(host, hand); err != nil {
		return errors.Trace(err)
	}

	return g.loadExtraNetworks()
}

// SyncState .
func (g *Guest) SyncState(ctx context.Context) error {
	switch g.Status {
	case meta.StatusDestroying:
		return g.ProcessDestroy()

	case meta.StatusStopping:
		return g.stop(ctx, true)

	case meta.StatusRunning:
		fallthrough
	case meta.StatusStarting:
		return g.start(ctx)

	case meta.StatusCreating:
		return g.create()

	default:
		// nothing to do
		return nil
	}
}

// Start .
func (g *Guest) Start(ctx context.Context) error {
	if err := g.ForwardStarting(); err != nil {
		return err
	}
	defer log.Debugf("exit g.Start")
	return g.start(ctx)
}

func (g *Guest) start(ctx context.Context) error {
	return g.botOperate(func(bot Bot) error {
		switch st, err := bot.GetState(); {
		case err != nil:
			return errors.Trace(err)
		case st == libvirt.DomainRunning:
			return nil
		}
		log.Debugf("Entering Boot")
		if err := bot.Boot(ctx); err != nil {
			return err
		}
		log.Debugf("Entering joinEthernet")
		if err := g.joinEthernet(); err != nil {
			return err
		}
		log.Debugf("Entering forwardRunning")
		return g.ForwardRunning()
	})
}

// Resize .
func (g *Guest) Resize(cpu int, mem int64, vols []volume.Volume) error {
	// Only checking, without touch metadata
	// due to we wanna keep further booting successfully all the time.
	if !g.CheckForwardStatus(meta.StatusResizing) {
		return errors.Annotatef(errors.ErrForwardStatus, "only stopped/running guest can be resized, but it's %s", g.Status)
	}

	volMap := map[string]volume.Volume{}
	for _, vol := range vols {
		volMap[vol.GetMountDir()] = vol
	}
	// Actually, mntCaps from ERU will include completed volumes,
	// even those volumes aren't affected.
	if len(vols) > 0 {
		// Just amplifies those original volumes.
		log.Infof("Resize(%s): Try to amplify exist volumes", g.ID)
		if err := g.amplifyOrigVols(volMap); err != nil {
			return errors.Trace(err)
		}
		// Attaches new extra volumes.
		log.Infof("Resize(%s): Try to attach new volumes", g.ID)
		if err := g.attachVols(volMap); err != nil {
			return errors.Trace(err)
		}
	}

	log.Infof("Resize(%s): Resize cpu and memory if necessary", g.ID)
	if cpu == g.CPU && mem == g.Memory {
		return nil
	}

	return g.resizeSpec(cpu, mem)
}

func (g *Guest) amplifyOrigVols(volMap map[string]volume.Volume) error {
	var err error
	g.rangeVolumes(func(sn int, vol volume.Volume) bool {
		newCapMod, affected := volMap[vol.GetMountDir()]
		if !affected {
			return true
		}

		var delta int64
		switch delta = newCapMod.GetSize() - vol.GetSize(); {
		case delta < 0:
			err = errors.Annotatef(errors.ErrCannotShrinkVolume, "mount dir: %s", newCapMod.GetMountDir())
			return false
		case delta == 0: // nothing changed
			return true
		}

		err = g.botOperate(func(bot Bot) error {
			return bot.AmplifyVolume(vol, delta, base.GetDevicePathBySerialNumber(sn))
		})
		return err == nil
	})

	return err
}

func (g *Guest) attachVols(volMap map[string]volume.Volume) error {
	for _, vol := range volMap {
		if g.Vols.Exists(vol.GetMountDir()) {
			continue
		}
		vol.SetGuestID(g.ID)
		vol.SetStatus(g.Status, true) //nolint:errcheck
		vol.GenerateID()
		if err := g.attachVol(vol); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (g *Guest) attachVol(volmod volume.Volume) (err error) {
	devName := g.nextVolumeName()

	if err = g.AppendVols(volmod); err != nil {
		return errors.Trace(err)
	}

	var rollback func()
	defer func() {
		if err != nil {
			if rollback != nil {
				rollback()
			}
			g.RemoveVol(volmod.GetID())
		}
	}()

	if err = g.botOperate(func(bot Bot) (ae error) {
		rollback, ae = bot.AttachVolume(volmod, devName)
		return ae
	}); err != nil {
		return
	}

	return g.Save()
}

func (g *Guest) resizeSpec(cpu int, mem int64) error {
	if err := g.botOperate(func(bot Bot) error {
		return bot.Resize(cpu, mem)
	}); err != nil {
		return errors.Trace(err)
	}

	return g.Guest.Resize(cpu, mem)
}

// ListSnapshot If volID == "", list snapshots of all vols. Else will find vol with matching volID.
func (g *Guest) ListSnapshot(volID string) (map[volume.Volume]base.Snapshots, error) {
	volSnap := make(map[volume.Volume]base.Snapshots)

	matched := false
	for _, v := range g.Vols {
		if v.GetID() == volID || volID == "" {
			api := v.NewSnapshotAPI()
			volSnap[v] = api.List()
			matched = true
		}
	}

	if !matched {
		return nil, errors.Annotatef(errors.ErrInvalidValue, "volID %s not exists", volID)
	}

	return volSnap, nil
}

// CheckVolume .
func (g *Guest) CheckVolume(volID string) error {
	if g.Status != meta.StatusStopped && g.Status != meta.StatusPaused {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only paused/stopped guest can be perform volume check, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CheckVolume(vol)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// RepairVolume .
func (g *Guest) RepairVolume(volID string) error {
	if g.Status != meta.StatusStopped {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only stopped guest can be perform volume check, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.RepairVolume(vol)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// CreateSnapshot .
func (g *Guest) CreateSnapshot(volID string) error {
	if g.Status != meta.StatusStopped && g.Status != meta.StatusPaused {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only paused/stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CreateSnapshot(vol)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// CommitSnapshot .
func (g *Guest) CommitSnapshot(volID string, snapID string) error {
	if g.Status != meta.StatusStopped {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CommitSnapshot(vol, snapID)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// CommitSnapshot .
func (g *Guest) CommitSnapshotByDay(volID string, day int) error {
	if g.Status != meta.StatusStopped {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CommitSnapshotByDay(vol, day)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// RestoreSnapshot .
func (g *Guest) RestoreSnapshot(volID string, snapID string) error {
	if g.Status != meta.StatusStopped {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.RestoreSnapshot(vol, snapID)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Capture .
func (g *Guest) Capture(user, name string, overridden bool) (uimg *image.UserImage, err error) {
	var orig *image.UserImage
	if overridden {
		if orig, err = image.LoadUserImage(user, name); err != nil {
			return
		}
	}

	if err = g.ForwardCapturing(); err != nil {
		return
	}

	if err = g.botOperate(func(bot Bot) error {
		var ce error
		if uimg, ce = bot.Capture(user, name); ce != nil {
			return errors.Trace(ce)
		}
		return g.ForwardCaptured()
	}); err != nil {
		return
	}

	if err = g.ForwardStopped(false); err != nil {
		return
	}

	if overridden {
		orig.Distro = uimg.Distro
		orig.Size = uimg.Size
		err = orig.Save()
	} else {
		err = uimg.Create()
	}

	return uimg, err
}

// Migrate .
func (g *Guest) Migrate() error {
	return utils.Invoke([]func() error{
		g.ForwardMigrating,
		g.migrate,
	})
}

func (g *Guest) migrate() error {
	return g.botOperate(func(bot Bot) error {
		return bot.Migrate()
	})
}

// Create .
func (g *Guest) Create() error {
	return utils.Invoke([]func() error{
		g.ForwardCreating,
		g.create,
	})
}

func (g *Guest) create() error {
	return g.botOperate(func(bot Bot) error {
		return utils.Invoke([]func() error{
			bot.Create,
		})
	})
}

// Stop .
func (g *Guest) Stop(ctx context.Context, force bool) error {
	if err := g.ForwardStopping(); !force && err != nil {
		return errors.Trace(err)
	}
	return g.stop(ctx, force)
}

func (g *Guest) stop(ctx context.Context, force bool) error {
	return g.botOperate(func(bot Bot) error {
		if err := bot.Shutdown(ctx, force); err != nil {
			return errors.Trace(err)
		}
		return g.ForwardStopped(force)
	})
}

// Suspend .
func (g *Guest) Suspend() error {
	return utils.Invoke([]func() error{
		g.ForwardPausing,
		g.suspend,
	})
}

func (g *Guest) suspend() error {
	return g.botOperate(func(bot Bot) error {
		return utils.Invoke([]func() error{
			bot.Suspend,
			g.ForwardPaused,
		})
	})
}

// Resume .
func (g *Guest) Resume() error {
	return utils.Invoke([]func() error{
		g.ForwardResuming,
		g.resume,
	})
}

func (g *Guest) resume() error {
	return g.botOperate(func(bot Bot) error {
		return utils.Invoke([]func() error{
			bot.Resume,
			g.ForwardRunning,
		})
	})
}

// Destroy .
func (g *Guest) Destroy(ctx context.Context, force bool) (<-chan error, error) {
	if force {
		if err := g.stop(ctx, true); err != nil && !errors.IsDomainNotExistsErr(err) {
			return nil, errors.Trace(err)
		}
	}

	if err := g.ForwardDestroying(force); err != nil {
		return nil, errors.Trace(err)
	}
	log.Infof("[guest.Destroy] set state of guest %s to destroying", g.ID)

	done := make(chan error, 1)

	// will return immediately as the destroy request has been accepted.
	// the detail work will be processed asynchronously
	go func() {
		err := g.ProcessDestroy()
		if err != nil {
			log.ErrorStackf(err, "destroy guest %s failed", g.ID)
			//TODO: move to recovery list
		}
		done <- err
	}()

	return done, nil
}

func (g *Guest) ProcessDestroy() error {
	log.Infof("[guest.destroy] begin to destroy guest %s ", g.ID)
	return g.botOperate(func(bot Bot) error {
		if err := g.DeleteNetwork(); err != nil {
			return errors.Trace(err)
		}

		if err := bot.Undefine(); err != nil {
			return errors.Trace(err)
		}

		return g.Delete(false)
	})
}

const waitRetries = 30 // 30 second
// Wait .
func (g *Guest) Wait(toStatus string, block bool) error {
	if !g.CheckForwardStatus(toStatus) {
		return errors.ErrForwardStatus
	}
	return g.botOperate(func(bot Bot) error {
		cnt := 0
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			if g.Status == toStatus {
				return nil
			}
			if !block {
				if cnt++; cnt > waitRetries {
					return errors.New("wait time out")
				}
			}
		}
		return nil
	})
}

// GetUUID .
func (g *Guest) GetUUID() (uuid string, err error) {
	err = g.botOperate(func(bot Bot) error {
		uuid, err = bot.GetUUID()
		return err
	})
	return
}

func (g *Guest) botOperate(fn func(bot Bot) error, skipLock ...bool) error {
	var bot, err = g.newBot(g)
	if err != nil {
		return errors.Trace(err)
	}

	defer bot.Close()

	if !(len(skipLock) > 0 && skipLock[0]) {
		if err := bot.Trylock(); err != nil {
			return errors.Trace(err)
		}
		defer bot.Unlock()
	}
	return fn(bot)
}

// CacheImage downloads the image from hub.
func (g *Guest) CacheImage(_ sync.Locker) error {
	// not implemented
	return nil
}

func (g *Guest) sysVolume() (vol volume.Volume, err error) {
	g.rangeVolumes(func(_ int, v volume.Volume) bool {
		if v.IsSys() {
			vol = v
			return false
		}
		return true
	})

	if vol == nil {
		err = errors.Annotatef(errors.ErrSysVolumeNotExists, g.ID)
	}

	return
}

func (g *Guest) rangeVolumes(fn func(int, volume.Volume) bool) {
	for i, vol := range g.Vols {
		if !fn(i, vol) {
			return
		}
	}
}

// Distro .
func (g *Guest) Distro() string {
	return g.Img.GetDistro()
}

// AttachConsole .
func (g *Guest) AttachConsole(ctx context.Context, serverStream io.ReadWriteCloser, flags types.OpenConsoleFlags) error {
	return g.botOperate(func(bot Bot) error {
		// epoller should bind with deamon's lifecycle but just init/destroy here for simplicity
		console, err := bot.OpenConsole(ctx, flags)
		if err != nil {
			return errors.Trace(err)
		}

		done1 := make(chan struct{})
		done2 := make(chan struct{})

		// pty -> user
		go func() {
			defer func() {
				close(done1)
				log.Infof("[guest.AttachConsole] copy console stream goroutine exited")
			}()
			console.To(ctx, serverStream) //nolint
		}()

		// user -> pty
		go func() {
			defer func() {
				close(done2)
				log.Infof("[guest.AttachConsole] copy server stream goroutine exited")
			}()

			initCmds := append([]byte(strings.Join(flags.Commands, " ")), '\r')
			reader := io.MultiReader(bytes.NewBuffer(initCmds), serverStream)
			console.From(ctx, reader) //nolint
		}()

		// either copy goroutine exit
		select {
		case <-done1:
		case <-done2:
		case <-ctx.Done():
			log.Debugf("[guest.AttachConsole] context done")
		}
		if err := console.Close(); err != nil {
			log.Errorf("[guest.AttachConsole] failed to close epoll console")
		}

		<-done1
		<-done2
		log.Infof("[guest.AttachConsole] exit.")
		// g.ExecuteCommand(context.Background(), []string{"yaexec", "kill"}) //nolint
		// log.Infof("[guest.AttachConsole] yaexec completes: %v", commands)
		return nil
	})
}

// ResizeConsoleWindow .
func (g *Guest) ResizeConsoleWindow(ctx context.Context, height, width uint) (err error) {
	return g.botOperate(func(bot Bot) error {
		// types.ConsoleStateManager.WaitUntilConsoleOpen(ctx, g.ID)
		resizeCmd := fmt.Sprintf("yaexec resize -r %d -c %d", height, width)
		output, code, _, err := g.ExecuteCommand(ctx, strings.Split(resizeCmd, " "))
		if code != 0 || err != nil {
			log.Errorf("[guest.ResizeConsoleWindow] resize failed: %v, %v", output, err)
		}
		return err
	}, true)
}

// Cat .
func (g *Guest) Cat(ctx context.Context, path string, dest io.Writer) error {
	return g.botOperate(func(bot Bot) error {
		src, err := bot.OpenFile(ctx, path, "r")
		if err != nil {
			return errors.Trace(err)
		}

		defer src.Close(ctx)

		_, err = src.CopyTo(ctx, dest)

		return err
	})
}

// Log .
func (g *Guest) Log(ctx context.Context, n int, logPath string, dest io.WriteCloser) error {
	return g.botOperate(func(bot Bot) error {
		if n == 0 {
			return nil
		}
		switch g.Status {
		case meta.StatusRunning:
			return g.logRunning(ctx, bot, n, logPath, dest)
		case meta.StatusStopped:
			gfx, err := g.getGfx(logPath)
			if err != nil {
				return err
			}
			defer gfx.Close()
			return g.logStopped(n, logPath, dest, gfx)
		default:
			return errors.Annotatef(errors.ErrNotValidLogStatus, "guest is %s", g.Status)
		}
	})
}

// CopyToGuest .
func (g *Guest) CopyToGuest(ctx context.Context, dest string, content chan []byte, overrideFolder bool) error {
	return g.botOperate(func(bot Bot) error {
		switch g.Status {
		case meta.StatusRunning:
			return g.copyToGuestRunning(ctx, dest, content, bot, overrideFolder)
		case meta.StatusStopped:
			fallthrough
		case meta.StatusCreating:
			gfx, err := g.getGfx(dest)
			if err != nil {
				return errors.Trace(err)
			}
			defer gfx.Close()
			return g.copyToGuestNotRunning(dest, content, overrideFolder, gfx)
		default:
			return errors.Annotatef(errors.ErrNotValidCopyStatus, "guest is %s", g.Status)
		}
	})
}

// ExecuteCommand .
func (g *Guest) ExecuteCommand(ctx context.Context, commands []string) (output []byte, exitCode, pid int, err error) {
	err = g.botOperate(func(bot Bot) error {
		switch st, err := bot.GetState(); {
		case err != nil:
			return errors.Trace(err)
		case st != libvirt.DomainRunning:
			return errors.Annotatef(errors.ErrExecOnNonRunningGuest, g.ID)
		}

		output, exitCode, pid, err = bot.ExecuteCommand(ctx, commands)
		return err
	}, true)
	return
}

// nextVolumeName .
func (g *Guest) nextVolumeName() string {
	return base.GetDeviceName(g.Vols.Len())
}
