package boar

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/pkg/errors"
)

var imageMutex sync.Mutex

func (svc *Boar) PushImage(_ context.Context, _, _ string) (err error) {
	// todo
	return
}

func (svc *Boar) RemoveImage(_ context.Context, imageName, user string, force, prune bool) (removed []string, err error) { //nolint
	defer logErr(err)

	img, err := image.Load(imageName, user)
	if err != nil {
		return nil, errors.Trace(err)
	}

	imageMutex.Lock()
	defer imageMutex.Unlock()

	if exists, err := image.Exists(img); err != nil {
		return nil, errors.Trace(err)
	} else if exists {
		if err = os.Remove(img.Filepath()); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return []string{img.GetID()}, nil
}

func (svc *Boar) ListImage(_ context.Context, filter string) (ans []types.SysImage, err error) {
	defer logErr(err)

	imgs, err := image.ListSysImages()
	if err != nil {
		return nil, err
	}

	images := []image.Image{}
	if len(filter) < 1 {
		images = imgs
	} else {
		var regExp *regexp.Regexp
		filter = strings.ReplaceAll(filter, "*", ".*")
		if regExp, err = regexp.Compile(fmt.Sprintf("%s%s%s", "^", filter, "$")); err != nil {
			return nil, err
		}

		for _, img := range imgs {
			if regExp.MatchString(img.GetName()) {
				images = append(images, img)
			}
		}
	}

	for _, img := range images {
		ans = append(ans, types.SysImage{
			Name:   img.GetName(),
			User:   img.GetUser(),
			Distro: img.GetDistro(),
			ID:     img.GetID(),
			Type:   img.GetType(),
		})
	}

	return ans, err
}

func (svc *Boar) PullImage(context.Context, string, bool) (msg string, err error) {
	// todo
	return
}

func (svc *Boar) DigestImage(_ context.Context, imageName string, local bool) (digest []string, err error) {
	defer logErr(err)

	if !local {
		// TODO: wait for image-hub implementation and calico update
		return []string{""}, nil
	}

	// If not exists return error
	// If exists return digests

	img, err := image.LoadSysImage(imageName)
	if err != nil {
		return nil, err
	}

	hash, err := img.UpdateHash()
	if err != nil {
		return nil, err
	}

	return []string{hash}, nil
}
