package local

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"

	stotypes "github.com/projecteru2/resource-storage/storage/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/virt/guestfs/gfsx"
	"github.com/projecteru2/yavirt/internal/virt/types"
	virtutils "github.com/projecteru2/yavirt/internal/virt/utils"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/sh"
	"github.com/projecteru2/yavirt/pkg/utils"
)

var (
	//go:embed templates/disk.xml
	diskXML string
	tmpl    *template.Template
)

// Volume .
// etcd keys:
//
//	/vols/<vol id>
type Volume struct {
	base.Volume            `mapstructure:",squash"`
	stotypes.VolumeBinding `mapstructure:",squash"`

	Format         string       `json:"format" mapstructure:"format"`
	SnapIDs        []string     `json:"snaps" mapstructure:"snaps"`
	BaseSnapshotID string       `json:"base_snapshot_id" mapstructure:"base_snapshot_id"`
	Snaps          Snapshots    `json:"-" mapstructure:"-"`
	flock          *utils.Flock `json:"-" mapstructure:"-"`
}

// LoadVolume loads data from etcd
func LoadVolume(id string) (*Volume, error) {
	var vol = NewVolume()
	vol.ID = id

	if err := meta.Load(vol); err != nil {
		return nil, err
	}

	return vol, vol.LoadSnapshots()
}

// NewVolumeFromStr
// format: `src:dst[:flags][:size][:read_IOPS:write_IOPS:read_bytes:write_bytes]`
// example: `/source:/dir0:rw:1024:1000:1000:10M:10M`
func NewVolumeFromStr(s string) (*Volume, error) {
	vb, err := stotypes.NewVolumeBinding(s)
	if err != nil {
		return nil, err
	}
	return &Volume{
		Format:        VolQcow2Format,
		Volume:        *base.New(),
		VolumeBinding: *vb,
	}, nil
}

// NewSysVolume .
func NewSysVolume(cap int64, imageName string) *Volume {
	vol := NewVolume()
	vol.SysImage = imageName
	vol.Flags = "rws"
	vol.SizeInBytes = cap
	return vol
}

// NewDataVolume .
func NewDataVolume(mnt string, cap int64) (*Volume, error) {
	mnt = strings.TrimSpace(mnt)

	src, dest := utils.PartRight(mnt, ":")
	src = strings.TrimSpace(src)
	dest = filepath.Join("/", strings.TrimSpace(dest))

	if len(src) > 0 {
		src = filepath.Join("/", src)
	}

	var vol = NewVolume()
	vol.Source = src
	vol.Destination = dest
	vol.Flags = "rw"
	vol.SizeInBytes = cap

	return vol, vol.Check()
}

func NewVolume() *Volume {
	return &Volume{
		Volume: *base.New(),
		Format: VolQcow2Format,
	}
}

func (v *Volume) Lock() error {
	fn := fmt.Sprintf("%s.flock", v.ID)
	fpth := filepath.Join(configs.Conf.VirtFlockDir, fn)
	v.flock = utils.NewFlock(fpth)
	if err := v.flock.Trylock(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (v *Volume) Unlock() {
	v.flock.Close()
}

func (v *Volume) GetSize() int64 {
	return v.SizeInBytes
}

// Load .
func (v *Volume) LoadSnapshots() (err error) {
	if v.Snaps, err = LoadSnapshots(v.SnapIDs); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (v *Volume) NewSnapshotAPI() base.SnapshotAPI {
	return newSnapshotAPI(v)
}

// Amplify .
func (v *Volume) Amplify(delta int64, dom libvirt.Domain, ga agent.Interface, devPath string) (int64, error) {
	normDelta := utils.NormalizeMultiple1024(delta)
	newCap := v.SizeInBytes + normDelta
	if newCap > configs.Conf.MaxVolumeCap {
		return 0, errors.Annotatef(errors.ErrInvalidValue, "exceeds the max cap: %d", configs.Conf.MaxVolumeCap)
	}

	least := utils.Max(
		configs.Conf.ResizeVolumeMinSize,
		int64(float64(v.SizeInBytes)*configs.Conf.ResizeVolumeMinRatio),
	)
	if least > normDelta {
		return 0, errors.Annotatef(errors.ErrInvalidValue, "invalid cap: at least %d, but %d",
			v.SizeInBytes+least, v.SizeInBytes+normDelta)
	}

	st, err := dom.GetState()
	if err != nil {
		return 0, errors.Trace(err)
	}
	switch st {
	case libvirt.DomainShutoff:
		err = virtutils.AmplifyImage(context.Background(), v.Filepath(), delta)
	case libvirt.DomainRunning:
		err = v.amplifyOnline(delta, dom, ga, devPath)
	default:
		err = types.NewDomainStatesErr(st, libvirt.DomainShutoff, libvirt.DomainRunning)
	}
	if err != nil {
		return 0, err
	}

	v.SizeInBytes = newCap
	return normDelta, v.Save()
}

func (v *Volume) amplifyOnline(delta int64, dom libvirt.Domain, ga agent.Interface, devPath string) error {
	if err := dom.AmplifyVolume(v.Filepath(), uint64(v.SizeInBytes+delta)); err != nil {
		return errors.Trace(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout.Duration())
	defer cancel()
	return v.amplify(ctx, ga, devPath)
}

func (v *Volume) amplify(ctx context.Context, ga agent.Interface, devPath string) error {
	// NOTICE:
	//   Actually, volume raw devices aren't necessary for re-parting.

	stoppedServices, err := base.StopSystemdServices(ctx, ga, devPath)
	if err != nil {
		return errors.Trace(err)
	}

	cmds := [][]string{
		{"umount", v.GetMountDir()},
		{"partprobe"},
		{"e2fsck", "-fy", devPath},
		{"resize2fs", devPath},
		{"mount", "-a"},
	}

	if err := base.ExecCommands(ctx, ga, cmds); err != nil {
		return errors.Trace(err)
	}

	if err := base.RestartSystemdServices(ctx, ga, stoppedServices); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// AppendSnaps .
func (v *Volume) AppendSnaps(snaps ...*Snapshot) error {
	if v.Snaps.Len()+len(snaps) > configs.Conf.MaxSnapshotsCount {
		return errors.Annotatef(errors.ErrTooManyVolumes, "at most %d", configs.Conf.MaxSnapshotsCount)
	}

	res := Snapshots(snaps)

	v.Snaps.Append(snaps...)

	v.SnapIDs = append(v.SnapIDs, res.Ids()...)

	return nil
}

// RemoveSnaps Remove snapshots meta by preserving the order.
func (v *Volume) RemoveSnap(snapID string) {
	keep := 0

	for i := 0; i < v.Snaps.Len(); i++ {
		if v.Snaps[i].ID == snapID {
			continue
		}

		v.Snaps[keep] = v.Snaps[i]
		v.SnapIDs[keep] = v.SnapIDs[i]
		keep++
	}

	v.Snaps = v.Snaps[:keep]
	v.SnapIDs = v.SnapIDs[:keep]
}

// Save updates metadata to persistence store.
func (v *Volume) Save() error {
	return meta.Save(meta.Resources{v})
}

func (v *Volume) GetMountDir() string {
	if len(v.Destination) > 0 {
		return v.Destination
	}
	return "/"
}

func (v *Volume) String() string {
	var mnt = "/"
	if len(v.Destination) > 0 {
		mnt = v.Destination
	}
	return fmt.Sprintf("%s, %s, %s:%s, size: %d", v.Filepath(), v.Status, v.GuestID, mnt, v.SizeInBytes)
}

// Filepath .
func (v *Volume) Filepath() string {
	if len(v.Source) > 0 {
		return filepath.Join(v.Source, v.Name())
	}
	return v.JoinVirtPath(v.Name())
}

// Name .
func (v *Volume) Name() string {
	ty := "dat"
	if v.IsSys() {
		ty = "sys"
	}
	return fmt.Sprintf("%s-%s.vol", ty, v.ID)
}

// Check .
func (v *Volume) Check() error {
	switch {
	case v.SizeInBytes < configs.Conf.MinVolumeCap || v.SizeInBytes > configs.Conf.MaxVolumeCap:
		return errors.Annotatef(errors.ErrInvalidValue, "capacity: %d", v.SizeInBytes)
	case v.Source == "/":
		return errors.Annotatef(errors.ErrInvalidValue, "host dir: %s", v.Source)
	case v.Destination == "/":
		return errors.Annotatef(errors.ErrInvalidValue, "mount dir: %s", v.Destination)
	default:
		if _, err := os.Stat(v.Filepath()); err == nil {
			return virtutils.Check(context.Background(), v.Filepath())
		}
		return nil
	}
}

// Repair .
func (v *Volume) Repair() error {
	return virtutils.Repair(context.Background(), v.Filepath())
}

// IsSys .
func (v *Volume) IsSys() bool {
	return strings.Contains(v.Flags, "s")
}

func (v *Volume) Prepare(img image.Image) error {
	if v.IsSys() { //nolint
		if err := sh.Copy(img.Filepath(), v.Filepath()); err != nil {
			return errors.Trace(err)
		}

		// Removes the image file form localhost if it's a user image.
		if img.GetType() == image.ImageUser {
			if err := sh.Remove(img.Filepath()); err != nil {
				// Prints the error message but it doesn't break the func.
				log.WarnStack(err)
			}
		}
		return nil
	} else { //nolint
		var path = v.Filepath()
		return virtutils.CreateImage(context.Background(), VolQcow2Format, path, v.SizeInBytes)
	}
}

// Cleanup deletes the qcow2 file
func (v *Volume) Cleanup() error {
	return sh.Remove(v.Filepath())
}

func (v *Volume) CaptureImage(user, name string) (*image.UserImage, error) {
	uimg := image.NewUserImage(user, name, v.SizeInBytes)
	uimg.Distro = types.Unknown

	if err := sh.Copy(v.Filepath(), uimg.Filepath()); err != nil {
		return nil, errors.Trace(err)
	}
	return uimg, nil
}

func (v *Volume) GenerateXML(i int) ([]byte, error) {
	devName := base.GetDeviceName(i)
	return v.GenerateXMLWithDevName(devName)
}

func (v *Volume) GenerateXMLWithDevName(devName string) ([]byte, error) {
	args := map[string]any{
		"path":       v.Filepath(),
		"dev":        devName,
		"read_iops":  fmt.Sprintf("%d", v.ReadIOPS),
		"write_iops": fmt.Sprintf("%d", v.WriteIOPS),
		"read_bps":   fmt.Sprintf("%d", v.ReadBPS),
		"write_bps":  fmt.Sprintf("%d", v.WriteBPS),
	}

	if tmpl == nil {
		t, err := template.New("local-vol-tpl").Parse(diskXML)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tmpl = t
	}

	var wr bytes.Buffer
	if err := tmpl.Execute(&wr, args); err != nil {
		return nil, errors.Trace(err)
	}
	return wr.Bytes(), nil
}

func (v *Volume) GetGfx() (guestfs.Guestfs, error) {
	return gfsx.New(v.Filepath())
}
