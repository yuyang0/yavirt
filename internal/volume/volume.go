package volume

import (
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/internal/volume/local"
	"github.com/projecteru2/yavirt/internal/volume/rbd"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
)

type Volume interface { //nolint:interfacebloat
	meta.GenericInterface

	// getters
	Name() string
	GetMountDir() string
	GetSize() int64
	GetHostname() string
	GetGuestID() string
	// setters
	SetHostname(name string)
	SetGuestID(id string)
	GenerateID()

	// Note: caller should call dom.Free() to release resource
	Amplify(delta int64, dom libvirt.Domain, ga agent.Interface, devPath string) (normDelta int64, err error)
	Check() error
	Repair() error
	IsSys() bool
	// prepare the volume, run before create guest.
	Prepare(image.Image) error
	GenerateXML(i int) ([]byte, error)
	GenerateXMLWithDevName(devName string) ([]byte, error)
	Cleanup() error
	// delete data in store
	Delete(force bool) error
	CaptureImage(user, name string) (*image.UserImage, error)
	// Save data to store
	Save() error

	Lock() error
	Unlock()

	GetGfx() (guestfs.Guestfs, error)

	NewSnapshotAPI() base.SnapshotAPI
}

type Volumes []Volume

func LoadVolumes(ids []string) (vols Volumes, err error) {
	vols = make(Volumes, len(ids))

	for i, id := range ids {
		key := meta.VolumeKey(id)
		rawVal, ver, err := meta.LoadRaw(key)
		if err != nil {
			return nil, errors.Trace(err)
		}
		var vol Volume
		if _, ok := rawVal["pool"]; ok {
			vol = rbd.New()
		} else {
			vol = local.NewVolume()
		}

		if err := mapstructure.Decode(rawVal, &vol); err != nil {
			return vols, err
		}
		vol.SetVer(ver)
		vols[i] = vol
	}
	return vols, nil
}

// Check .
func (vols Volumes) Check() error {
	for _, v := range vols {
		if v == nil {
			return errors.Annotatef(errors.ErrInvalidValue, "nil *Volume")
		}
		if err := v.Check(); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Find .
func (vols Volumes) Find(volID string) (Volume, error) {
	for _, v := range vols {
		if v.GetID() == volID {
			return v, nil
		}
	}

	return nil, errors.Annotatef(errors.ErrInvalidValue, "volID %s not exists", volID)
}

// Exists checks the volume if exists, in which mounted the directory.
func (vols Volumes) Exists(mnt string) bool {
	for _, vol := range vols {
		switch {
		case vol.IsSys():
			continue
		case vol.GetMountDir() == mnt:
			return true
		}
	}
	return false
}

// Len .
func (vols Volumes) Len() int {
	return len(vols)
}

func (vols Volumes) Resources() meta.Resources {
	var r = make(meta.Resources, len(vols))
	for i, v := range vols {
		r[i] = v
	}
	return r
}

func (vols Volumes) SetGuestID(id string) {
	for _, vol := range vols {
		vol.SetGuestID(id)
	}
}

func (vols Volumes) SetHostName(name string) {
	for _, vol := range vols {
		vol.SetHostname(name)
	}
}

func (vols Volumes) Ids() []string {
	var v = make([]string, len(vols))
	for i, vol := range vols {
		v[i] = vol.GetID()
	}
	return v
}

func (vols Volumes) GenID() {
	for _, vol := range vols {
		vol.GenerateID()
	}
}

func (vols Volumes) SetStatus(st string, force bool) error {
	for _, vol := range vols {
		if err := vol.SetStatus(st, force); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (vols Volumes) MetaKeys() []string {
	var keys = make([]string, len(vols))
	for i, vol := range vols {
		keys[i] = vol.MetaKey()
	}
	return keys
}

// GetMntVol return the vol of a path if exists .
func (vols Volumes) GetMntVol(path string) (Volume, error) {
	path = filepath.Dir(path)
	if path[0] != '/' {
		return nil, errors.ErrDestinationInvalid
	}

	// append a `/` to the end.
	// this can avoid the wrong result caused by following example:
	// path: /dst10, mount dir: /dst1
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	var sys, maxVol Volume
	maxLen := -1
	for _, vol := range vols {
		if vol.IsSys() {
			sys = vol
			continue
		}

		mntDirLen := len(vol.GetMountDir())
		mntDir := vol.GetMountDir()
		if !strings.HasSuffix(mntDir, "/") {
			mntDir += "/"
		}
		if mntDirLen > maxLen && strings.Index(path, mntDir) == 0 {
			maxLen = mntDirLen
			maxVol = vol
		}
	}

	if maxLen < 1 {
		return sys, nil
	}
	return maxVol, nil
}
