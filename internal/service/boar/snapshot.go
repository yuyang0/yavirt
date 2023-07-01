package boar

import (
	"context"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/pkg/log"
)

// ListSnapshot .
func (svc *Boar) ListSnapshot(ctx context.Context, req types.ListSnapshotReq) (snaps types.Snapshots, err error) {
	volSnap, err := svc.guest.ListSnapshot(ctx, req.ID, req.VolID)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}

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
	if err = svc.guest.CreateSnapshot(ctx, req.ID, req.VolID); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// CommitSnapshot .
func (svc *Boar) CommitSnapshot(ctx context.Context, req types.CommitSnapshotReq) (err error) {
	if err = svc.guest.CommitSnapshot(ctx, req.ID, req.VolID, req.SnapID); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// CommitSnapshotByDay .
func (svc *Boar) CommitSnapshotByDay(ctx context.Context, id, volID string, day int) (err error) {
	if err = svc.guest.CommitSnapshotByDay(ctx, id, volID, day); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// RestoreSnapshot .
func (svc *Boar) RestoreSnapshot(ctx context.Context, req types.RestoreSnapshotReq) (err error) {
	if err = svc.guest.RestoreSnapshot(ctx, req.ID, req.VolID, req.SnapID); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}
