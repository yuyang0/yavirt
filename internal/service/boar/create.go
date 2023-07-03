package boar

import (
	"context"
	"time"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"

	"github.com/projecteru2/yavirt/internal/virt/guest"
	virtypes "github.com/projecteru2/yavirt/internal/virt/types"
)

// CreateGuest .
func (svc *Boar) CreateGuest(ctx context.Context, opts virtypes.GuestCreateOption) (*types.Guest, error) {
	if opts.CPU == 0 {
		opts.CPU = utils.Min(svc.Host.CPU, configs.Conf.MaxCPU)
	}
	if opts.Mem == 0 {
		opts.Mem = utils.Min(svc.Host.Memory, configs.Conf.MaxMemory)
	}

	g, err := svc.Create(ctx, opts, svc.Host)
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

// Create creates a new guest.
func (svc *Boar) Create(ctx context.Context, opts virtypes.GuestCreateOption, host *models.Host) (*guest.Guest, error) {
	vols, err := extractVols(opts.Resources)
	if err != nil {
		return nil, err
	}

	// Creates metadata.
	g, err := models.CreateGuest(opts, host, vols)
	if err != nil {
		return nil, errors.Trace(err)
	}
	log.Debugf("Guest Created: %+v", g)
	// Destroys resource and delete metadata while rolling back.
	var vg *guest.Guest
	destroy := func() {
		// Create a new context, because even when the origin ctx is done, we still need run destroy here
		dCtx, dCancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
		defer dCancelFn()

		defer func() {
			// delete guest in etcd
			log.Infof("Failed to create guest: %v", g)
			err := g.Delete(true)
			log.Errorf("Failed to delete guest in ETCD: %v", err)
		}()

		if vg == nil {
			return
		}

		done, err := vg.Destroy(dCtx, true)
		if err != nil {
			log.ErrorStack(err)
		}

		select {
		case err := <-done:
			if err != nil {
				log.ErrorStack(err)
			}
		case <-time.After(time.Minute):
			log.ErrorStackf(errors.ErrTimeout, "destroy timeout")
		}
	}

	// Creates the resource.
	create := func(ctx context.Context) (any, error) {
		var err error
		vg, err = svc.create(ctx, g)
		return vg, err
	}

	res, err := svc.do(ctx, g.ID, createOp, create, destroy)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return res.(*guest.Guest), nil
}

func (svc *Boar) create(ctx context.Context, g *models.Guest) (vg *guest.Guest, err error) {
	vg = guest.New(ctx, g)
	if err := vg.CacheImage(&imageMutex); err != nil {
		return nil, errors.Trace(err)
	}

	if vg.MAC, err = utils.QemuMAC(); err != nil {
		return nil, errors.Trace(err)
	}

	var rollback func() error
	if rollback, err = vg.CreateEthernet(); err != nil {
		return nil, errors.Trace(err)
	}

	if err = vg.Create(); err != nil {
		if re := rollback(); re != nil {
			err = errors.Wrap(err, re)
		}
		return nil, errors.Trace(err)
	}

	return vg, nil
}
