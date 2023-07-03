package boar

import (
	"context"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	"github.com/projecteru2/yavirt/pkg/errors"
)

// ListSnapshot .
func (svc *Boar) ListSnapshot(ctx context.Context, req types.ListSnapshotReq) (snaps types.Snapshots, err error) {
	defer logErr(err)

	g, err := svc.loadGuest(ctx, req.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	volSnap, err := g.ListSnapshot(req.VolID)

	for vol, s := range volSnap {
		for _, snap := range s {
			snaps = append(snaps, &types.Snapshot{
				VolID:       vol.GetID(),
				VolMountDir: vol.GetMountDir(),
				SnapID:      snap.GetID(),
				CreatedTime: snap.GetCreatedTime(),
			})
		}
	}

	return
}

// CreateSnapshot .
func (svc *Boar) CreateSnapshot(ctx context.Context, req types.CreateSnapshotReq) (err error) {
	defer logErr(err)
	volID := req.VolID

	return svc.ctrl(ctx, req.ID, createSnapshotOp, func(g *guest.Guest) error {
		suspended := false
		stopped := false
		if g.Status == meta.StatusRunning {
			if err := g.Suspend(); err != nil {
				return err
			}
			suspended = true
		}

		if err := g.CreateSnapshot(volID); err != nil {
			return err
		}

		if err := g.CheckVolume(volID); err != nil {

			if suspended {
				if err := g.Stop(ctx, true); err != nil {
					return err
				}
				suspended = false
				stopped = true
			}

			if err := g.RepairVolume(volID); err != nil {
				return err
			}
		}

		if suspended {
			return g.Resume()
		} else if stopped {
			return g.Start(ctx)
		}
		return nil
	}, nil)
}

// CommitSnapshot .
func (svc *Boar) CommitSnapshot(ctx context.Context, req types.CommitSnapshotReq) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, req.ID, commitSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == meta.StatusRunning {
			if err := g.Stop(ctx, true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.CommitSnapshot(req.VolID, req.SnapID); err != nil {
			return err
		}

		if stopped {
			return g.Start(ctx)
		}
		return nil
	}, nil)
}

// CommitSnapshotByDay .
func (svc *Boar) CommitSnapshotByDay(ctx context.Context, id, volID string, day int) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, id, commitSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == meta.StatusRunning {
			if err := g.Stop(ctx, true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.CommitSnapshotByDay(volID, day); err != nil {
			return err
		}

		if stopped {
			return g.Start(ctx)
		}
		return nil
	}, nil)
}

// RestoreSnapshot .
func (svc *Boar) RestoreSnapshot(ctx context.Context, req types.RestoreSnapshotReq) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, req.ID, restoreSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == meta.StatusRunning {
			if err := g.Stop(ctx, true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.RestoreSnapshot(req.VolID, req.SnapID); err != nil {
			return err
		}

		if stopped {
			return g.Start(ctx)
		}
		return nil
	}, nil)
}
