package boar

import (
	"context"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/pkg/log"
)

func (svc *Boar) PushImage(_ context.Context, _, _ string) (err error) {
	// todo
	return
}

func (svc *Boar) RemoveImage(ctx context.Context, imageName, user string, force, prune bool) (removed []string, err error) {
	if removed, err = svc.guest.RemoveImage(ctx, imageName, user, force, prune); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

func (svc *Boar) ListImage(ctx context.Context, filter string) ([]types.SysImage, error) {
	imgs, err := svc.guest.ListImage(ctx, filter)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}

	images := []types.SysImage{}
	for _, img := range imgs {
		images = append(images, types.SysImage{
			Name:   img.GetName(),
			User:   img.GetUser(),
			Distro: img.GetDistro(),
			ID:     img.GetID(),
			Type:   img.GetType(),
		})
	}

	return images, err
}

func (svc *Boar) PullImage(context.Context, string, bool) (msg string, err error) {
	// todo
	return
}

func (svc *Boar) DigestImage(ctx context.Context, imageName string, local bool) (digest []string, err error) {
	if digest, err = svc.guest.DigestImage(ctx, imageName, local); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}
