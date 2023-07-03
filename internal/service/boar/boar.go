package boar

import (
	"context"
	"fmt"
	"time"

	"github.com/projecteru2/libyavirt/types"

	"github.com/robfig/cron/v3"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/util"
	"github.com/projecteru2/yavirt/internal/ver"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	virtypes "github.com/projecteru2/yavirt/internal/virt/types"
	calihandler "github.com/projecteru2/yavirt/internal/vnet/handler/calico"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
	"github.com/projecteru2/yavirt/pkg/utils/hardware"
)

// Boar .
type Boar struct {
	Host        *models.Host
	BootGuestCh chan<- string
	caliHandler *calihandler.Handler

	pid2ExitCode   *utils.ExitCodeMap
	RecoverGuestCh chan<- string

	serializer *serializer
	watchers   *util.Watchers
}

func New(ctx context.Context) (br *Boar, err error) {
	br = &Boar{
		pid2ExitCode: utils.NewSyncMap(),
		serializer:   newSerializer(),
		watchers:     util.NewWatchers(),
	}

	go br.watchers.Run()

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
	vg, err := svc.loadGuest(ctx, id)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return nil, err
	}
	return convGuestResp(vg.Guest), nil
}

// GetGuestIDList .
func (svc *Boar) GetGuestIDList(ctx context.Context) ([]string, error) {
	ids, err := svc.ListLocalIDs(ctx, true)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return nil, err
	}
	return convGuestIDsResp(ids), err
}

// GetGuestUUID .
func (svc *Boar) GetGuestUUID(ctx context.Context, id string) (string, error) {
	uuid, err := svc.LoadUUID(ctx, id)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return "", err
	}
	return uuid, nil
}

// CaptureGuest .
func (svc *Boar) CaptureGuest(ctx context.Context, req types.CaptureGuestReq) (uimg *image.UserImage, err error) {
	defer logErr(err)

	g, err := svc.loadGuest(ctx, req.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	uImg, err := g.Capture(req.User, req.Name, req.Overridden)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return uImg, nil
}

// ResizeGuest re-allocates spec or volumes.
func (svc *Boar) ResizeGuest(ctx context.Context, opts *virtypes.GuestResizeOption) (err error) {
	defer logErr(err)

	vols, err := extractVols(opts.Resources)
	if err != nil {
		return err
	}
	return svc.ctrl(ctx, opts.ID, resizeOp, func(g *guest.Guest) error {
		return g.Resize(opts.CPU, opts.Mem, vols)
	}, nil)
}

// Wait .
func (svc *Boar) Wait(ctx context.Context, id string, block bool) (msg string, code int, err error) {
	defer logErr(err)

	err = svc.stopGuest(ctx, id, !block)
	if err != nil {
		return "stop error", -1, err
	}

	err = svc.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		if err = g.Wait(meta.StatusStopped, block); err != nil {
			return err
		}

		if g.LambdaOption != nil {
			msg = string(g.LambdaOption.CmdOutput)
			code = g.LambdaOption.ExitCode
		}

		return nil
	}, nil)
	return msg, code, err
}

// ListLocals lists all local guests.
func (svc *Boar) ListLocalIDs(ctx context.Context, onlyERU bool) ([]string, error) {
	ids, err := guest.ListLocalIDs(ctx)
	if err != nil {
		return nil, err
	}
	if !onlyERU {
		return ids, nil
	}
	var ans []string
	for _, id := range ids {
		if idgen.CheckID(id) {
			ans = append(ans, id)
		}
	}
	return ans, nil
}

// LoadUUID read a guest's UUID.
func (svc *Boar) LoadUUID(ctx context.Context, id string) (string, error) {
	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return "", errors.Trace(err)
	}
	return g.GetUUID()
}

// loadGuest read a guest from metadata.
func (svc *Boar) loadGuest(ctx context.Context, id string) (*guest.Guest, error) {
	g, err := models.LoadGuest(id)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var vg = guest.New(ctx, g)
	if err := vg.Load(); err != nil {
		return nil, errors.Trace(err)
	}

	return vg, nil
}
func (svc *Boar) WatchGuestEvents(context.Context) (*util.Watcher, error) {
	return svc.NewWatcher()
}

func logErr(err error) {
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
}

type ctrlFunc func(*guest.Guest) error

func (m *Boar) ctrl(ctx context.Context, id string, op op, fn ctrlFunc, rollback rollbackFunc) error { //nolint
	_, err := m.doCtrl(ctx, id, op, func(g *guest.Guest) (any, error) {
		return nil, fn(g)
	}, rollback)
	return err
}

type rollbackFunc func()
type doCtrlFunc func(*guest.Guest) (any, error)

func (m *Boar) doCtrl(ctx context.Context, id string, op op, fn doCtrlFunc, rollback rollbackFunc) (any, error) {
	do := func(ctx context.Context) (any, error) {
		g, err := m.loadGuest(ctx, id)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return fn(g)
	}
	return m.do(ctx, id, op, do, rollback)
}

type doFunc func(context.Context) (any, error)

func (m *Boar) do(ctx context.Context, id string, op op, fn doFunc, rollback rollbackFunc) (any, error) {
	t := &task{
		id:  id,
		op:  op,
		do:  fn,
		ctx: ctx,
	}

	dur := configs.Conf.VirtTimeout.Duration()
	timeout := time.After(dur)

	noti := m.serializer.Serialize(id, t)

	var result any
	var err error

	select {
	case <-noti.done:
		result = noti.result()
		err = noti.error()
	case <-ctx.Done():
		err = ctx.Err()
	case <-timeout:
		err = errors.Annotatef(errors.ErrTimeout, "exceed %v", dur)
	}
	if err != nil {
		if rollback != nil {
			rollback()
		}
		return nil, errors.Trace(err)
	}

	m.watchers.Watched(virtypes.Event{
		ID:     id,
		Type:   guestEventType,
		Action: op.String(),
		Time:   time.Now().UTC(),
	})

	return result, nil
}

func (m *Boar) NewWatcher() (*util.Watcher, error) {
	return m.watchers.Get()
}

const guestEventType = "guest"
