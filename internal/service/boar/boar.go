package boar

import (
	"context"
	"fmt"
	"io"

	"github.com/projecteru2/libyavirt/types"

	"github.com/robfig/cron/v3"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/util"
	"github.com/projecteru2/yavirt/internal/ver"
	"github.com/projecteru2/yavirt/internal/virt/guest/manager"
	virtypes "github.com/projecteru2/yavirt/internal/virt/types"
	calihandler "github.com/projecteru2/yavirt/internal/vnet/handler/calico"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
	"github.com/projecteru2/yavirt/pkg/utils/hardware"
)

// Boar .
type Boar struct {
	Host        *models.Host
	BootGuestCh chan<- string
	caliHandler *calihandler.Handler
	guest       manager.Manageable

	pid2ExitCode   *utils.ExitCodeMap
	RecoverGuestCh chan<- string
}

func New(ctx context.Context) (br *Boar, err error) {
	br = &Boar{
		guest:        manager.New(),
		pid2ExitCode: utils.NewSyncMap(),
	}

	if br.Host, err = models.LoadHost(); err != nil {
		return br, errors.Trace(err)
	}

	if err := br.setupCalico(); err != nil {
		return br, errors.Trace(err)
	}

	/*
		if err := svc.ScheduleSnapshotCreate(); err != nil {
			return errors.Trace(err)
		}
	*/

	return
}

// TODO: Decide time
func (svc *Boar) ScheduleSnapshotCreate() error {
	c := cron.New()

	// Everyday 3am
	if _, err := c.AddFunc("0 3 * * *", svc.batchCreateSnapshot); err != nil {
		return errors.Trace(err)
	}

	// Every Sunday 1am
	if _, err := c.AddFunc("0 1 * * SUN", svc.batchCommitSnapshot); err != nil {
		return errors.Trace(err)
	}

	// Start job asynchronously
	c.Start()

	return nil
}

func (svc *Boar) batchCreateSnapshot() {
	guests, err := models.GetAllGuests()
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return
	}

	for _, g := range guests {
		for _, volID := range g.VolIDs {
			req := types.CreateSnapshotReq{
				ID:    g.ID,
				VolID: volID,
			}

			if err := svc.CreateSnapshot(
				util.SetCalicoHandler(context.Background(), svc.caliHandler), req,
			); err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
			}
		}
	}
}

func (svc *Boar) batchCommitSnapshot() {
	guests, err := models.GetAllGuests()
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return
	}

	for _, g := range guests {
		for _, volID := range g.VolIDs {
			if err := svc.CommitSnapshotByDay(
				util.SetCalicoHandler(context.Background(), svc.caliHandler),
				g.ID,
				volID,
				configs.Conf.SnapshotRestorableDay,
			); err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
			}
		}
	}
}

// VirtContext .
func (svc *Boar) VirtContext(ctx context.Context) context.Context {
	return util.SetCalicoHandler(ctx, svc.caliHandler)
}

// Ping .
func (svc *Boar) Ping() map[string]string {
	return map[string]string{"version": ver.Version()}
}

// Info .
func (svc *Boar) Info() (*types.HostInfo, error) {
	res, err := hardware.FetchResources()
	if err != nil {
		return nil, err
	}
	return &types.HostInfo{
		ID:        fmt.Sprintf("%d", svc.Host.ID),
		CPU:       svc.Host.CPU,
		Mem:       svc.Host.Memory,
		Storage:   svc.Host.Storage,
		Resources: res,
	}, nil
}

// GetGuest .
func (svc *Boar) GetGuest(ctx context.Context, id string) (*types.Guest, error) {
	vg, err := svc.guest.Load(ctx, id)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return nil, err
	}
	return convGuestResp(vg.Guest), nil
}

// GetGuestIDList .
func (svc *Boar) GetGuestIDList(ctx context.Context) ([]string, error) {
	ids, err := svc.guest.ListLocalIDs(ctx, true)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return nil, err
	}
	return convGuestIDsResp(ids), err
}

// GetGuestUUID .
func (svc *Boar) GetGuestUUID(ctx context.Context, id string) (string, error) {
	uuid, err := svc.guest.LoadUUID(ctx, id)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return "", err
	}
	return uuid, nil
}

// CreateGuest .
func (svc *Boar) CreateGuest(ctx context.Context, opts virtypes.GuestCreateOption) (*types.Guest, error) {
	if opts.CPU == 0 {
		opts.CPU = utils.Min(svc.Host.CPU, configs.Conf.MaxCPU)
	}
	if opts.Mem == 0 {
		opts.Mem = utils.Min(svc.Host.Memory, configs.Conf.MaxMemory)
	}

	g, err := svc.guest.Create(ctx, opts, svc.Host)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return nil, err
	}

	go func() {
		svc.BootGuestCh <- g.ID
	}()

	return convGuestResp(g.Guest), nil
}

// CaptureGuest .
func (svc *Boar) CaptureGuest(ctx context.Context, req types.CaptureGuestReq) (uimg *image.UserImage, err error) {
	if uimg, err = svc.guest.Capture(ctx, req.VirtID(), req.User, req.Name, req.Overridden); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ResizeGuest .
func (svc *Boar) ResizeGuest(ctx context.Context, opts *virtypes.GuestResizeOption) (err error) {
	// vols := map[string]volume.Volume{}
	// for _, v := range opts.Volumes {
	// 	vol, err := models.NewDataVolume(v.Mount, v.Capacity, v.IO)
	// 	if err != nil {
	// 		return errors.Trace(err)
	// 	}
	// 	vols[vol.MountDir] = vol
	// }
	if err = svc.guest.Resize(ctx, opts); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ControlGuest .
func (svc *Boar) ControlGuest(ctx context.Context, id, operation string, force bool) (err error) {
	switch operation {
	case types.OpStart:
		err = svc.guest.Start(ctx, id)
	case types.OpStop:
		err = svc.guest.Stop(ctx, id, force)
	case types.OpDestroy:
		_, err = svc.guest.Destroy(ctx, id, force)
	case types.OpSuspend:
		err = svc.guest.Suspend(ctx, id)
	case types.OpResume:
		err = svc.guest.Resume(ctx, id)
	}

	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return errors.Trace(err)
	}

	return nil
}

// AttachGuest .
func (svc *Boar) AttachGuest(ctx context.Context, id string, stream io.ReadWriteCloser, flags virtypes.OpenConsoleFlags) (err error) {
	if err = svc.guest.AttachConsole(ctx, id, stream, flags); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ResizeConsoleWindow .
func (svc *Boar) ResizeConsoleWindow(ctx context.Context, id string, height, width uint) (err error) {
	if err = svc.guest.ResizeConsoleWindow(ctx, id, height, width); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ExecuteGuest .
func (svc *Boar) ExecuteGuest(ctx context.Context, id string, commands []string) (*types.ExecuteGuestMessage, error) {
	stdout, exitCode, pid, err := svc.guest.ExecuteCommand(ctx, id, commands)
	if err != nil {
		log.WarnStack(err)
		metrics.IncrError()
	}
	svc.pid2ExitCode.Put(id, pid, exitCode)
	return &types.ExecuteGuestMessage{
		Pid:      pid,
		Data:     stdout,
		ExitCode: exitCode,
	}, err
}

// ExecExitCode .
func (svc *Boar) ExecExitCode(id string, pid int) (int, error) {
	exitCode, err := svc.pid2ExitCode.Get(id, pid)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return 0, err
	}
	return exitCode, nil
}

// Cat .
func (svc *Boar) Cat(ctx context.Context, id, path string, dest io.WriteCloser) (err error) {
	if err = svc.guest.Cat(ctx, id, path, dest); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// CopyToGuest .
func (svc *Boar) CopyToGuest(ctx context.Context, id, dest string, content chan []byte, override bool) (err error) {
	if err = svc.guest.CopyToGuest(ctx, id, dest, content, override); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// Log .
func (svc *Boar) Log(ctx context.Context, id, logPath string, n int, dest io.WriteCloser) (err error) {
	if err = svc.guest.Log(ctx, id, logPath, n, dest); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// Wait .
func (svc *Boar) Wait(ctx context.Context, id string, block bool) (msg string, code int, err error) {
	err = svc.guest.Stop(ctx, id, !block)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return "stop error", -1, err
	}
	if msg, code, err = svc.guest.Wait(ctx, id, block); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

func (svc *Boar) WatchGuestEvents(context.Context) (*manager.Watcher, error) {
	return svc.guest.NewWatcher()
}
