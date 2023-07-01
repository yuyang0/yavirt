package volume

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/virt/guestfs/gfsx"
	gfsmocks "github.com/projecteru2/yavirt/internal/virt/guestfs/mocks"
	"github.com/projecteru2/yavirt/internal/virt/nic"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/internal/volume/local"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/sh"
	"github.com/projecteru2/yavirt/pkg/utils"
)

const (
	fs = "ext4"
	// Disable backing up of the device/partition
	backupDump = 0
	// Enable fsck checking the device/partition for errors at boot time.
	fsckPass = 2
)

// // Virt .
// type Virt interface { //nolint
// 	Mount(ga agent.Interface, devPath string) error
// 	IsSys() bool
// 	Amplify(cap int64, dom domain.Domain, ga agent.Interface, devPath string) (delta int64, err error)
// 	Filepath() string
// 	Model() *models.Volume
// 	Alloc(models.Image) error
// 	Undefine() error
// 	ConvertImage(user, name string) (uimg *models.UserImage, err error)
// 	Attach(dom domain.Domain, ga agent.Interface, devName string) (rollback func(), err error)
// 	Check() error
// 	Repair() error
// 	CreateSnapshot() error
// 	CommitSnapshot(snapID string) error
// 	CommitSnapshotByDay(day int) error
// 	RestoreSnapshot(snapID string) error
// }

// Undefine .
func Undefine(vol Volume) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	if err := api.DeleteAll(); err != nil {
		return errors.Trace(err)
	}
	if err := vol.Cleanup(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// ConvertImage .
func ConvertImage(vol Volume, user, name string) (uimg *image.UserImage, err error) {
	if err := vol.Lock(); err != nil {
		return nil, errors.Trace(err)
	}
	defer vol.Unlock()

	if !vol.IsSys() {
		return nil, errors.Annotatef(errors.ErrNotSysVolume, vol.GetID())
	}

	if uimg, err = vol.CaptureImage(user, name); err != nil {
		return nil, errors.Trace(err)
	}
	orig := uimg.Filepath()
	defer func() {
		if err != nil {
			if re := sh.Remove(orig); re != nil {
				err = errors.Wrap(err, re)
			}
		}
	}()

	var gfs guestfs.Guestfs
	if gfs, err = gfsx.New(orig); err != nil {
		return nil, errors.Trace(err)
	}
	defer gfs.Close()

	if uimg.Distro, err = gfs.Distro(); err != nil {
		return nil, errors.Trace(err)
	}

	if err = resetUserImage(gfs, uimg.Distro); err != nil {
		return nil, errors.Trace(err)
	}

	if err := uimg.NextVersion(); err != nil {
		return nil, errors.Trace(err)
	}

	if err = sh.Move(orig, uimg.Filepath()); err != nil {
		return nil, errors.Trace(err)
	}

	if err = cleanObsoleteUserImages(uimg); err != nil {
		return nil, errors.Trace(err)
	}

	return uimg, nil
}

func cleanObsoleteUserImages(_ *image.UserImage) error {
	// TODO
	return nil
}

// Attach .
// Note: caller should call dom.Free() to release resource
func Attach(vol Volume, dom libvirt.Domain, ga agent.Interface, devName string) (rollback func(), err error) {
	if rollback, err = create(vol); err != nil {
		return
	}

	defer func() {
		if err != nil {
			rollback()
		}
		rollback = nil
	}()

	var st libvirt.DomainState
	buf, err := vol.GenerateXMLWithDevName(devName)
	if err != nil {
		return
	}
	if st, err = dom.AttachVolume(string(buf)); err == nil && st == libvirt.DomainRunning {
		err = Mount(vol, ga, base.GetDevicePathByName(devName))
	}

	return
}

func create(vol Volume) (func(), error) {
	if err := Alloc(vol, nil); err != nil {
		return nil, errors.Trace(err)
	}

	if err := vol.Save(); err != nil {
		if ue := Undefine(vol); ue != nil {
			err = errors.Wrap(err, ue)
		}
		return nil, errors.Trace(err)
	}

	rb := func() {
		if err := Undefine(vol); err != nil {
			log.ErrorStack(err)
			return
		}
		if err := vol.Delete(true); err != nil {
			log.ErrorStack(err)
		}
	}

	return rb, nil
}

// Alloc .
func Alloc(vol Volume, img image.Image) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	return vol.Prepare(img)
}

// Amplify .
func Amplify(vol Volume, delta int64, dom libvirt.Domain, ga agent.Interface, devPath string) (normDelta int64, err error) {
	if err := vol.Lock(); err != nil {
		return 0, errors.Trace(err)
	}
	defer vol.Unlock()

	return vol.Amplify(delta, dom, ga, devPath)
}

// Check .
func Check(vol Volume) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	return vol.Check()
}

// Repair .
func Repair(vol Volume) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	return vol.Repair()
}

// CreateSnapshot .
func CreateSnapshot(vol Volume) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	return api.Create()
}

// CommitSnapshot .
func CommitSnapshot(vol Volume, snapID string) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	return api.Commit(snapID)
}

// CommitSnapshotByDay Commit snapshots created `day` days ago.
func CommitSnapshotByDay(vol Volume, day int) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	return api.CommitByDay(day)
}

// RestoreSnapshot .
func RestoreSnapshot(vol Volume, snapID string) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	return api.Restore(snapID)
}

// for ceph, before create snapshot, we need run fsfreeze
func FSFreeze(ctx context.Context, ga agent.Interface, v Volume, unfreeze bool) error {
	var cmd []string
	if unfreeze {
		cmd = []string{"fsfreeze", "--unfreeze", v.GetMountDir()}
	} else {
		cmd = []string{"fsfreeze", "--freeze", v.GetMountDir()}
	}
	var st = <-ga.ExecOutput(ctx, cmd[0], cmd[1:]...)
	if err := st.Error(); err != nil {
		return errors.Annotatef(err, "%v", cmd)
	}
	return nil
}

// Mount .
func Mount(vol Volume, ga agent.Interface, devPath string) error {
	if err := vol.Lock(); err != nil {
		return errors.Trace(err)
	}
	defer vol.Unlock()

	var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout.Duration())
	defer cancel()

	if err := format(ctx, ga, vol, devPath); err != nil {
		return errors.Trace(err)
	}

	if err := mount(ctx, ga, vol, devPath); err != nil {
		return errors.Trace(err)
	}

	if err := saveFstab(ctx, ga, vol, devPath); err != nil {
		return errors.Trace(err)
	}

	switch amplified, err := isAmplifying(ctx, ga, vol, devPath); {
	case err != nil:
		return errors.Trace(err)

	case amplified:
		return amplify(ctx, ga, vol, devPath)

	default:
		return nil
	}
}

func mount(ctx context.Context, ga agent.Interface, v Volume, devPath string) error {
	var mnt = v.GetMountDir()
	var st = <-ga.Exec(ctx, "mkdir", "-p", mnt)
	if err := st.Error(); err != nil {
		return errors.Annotatef(err, "mkdir %s failed", mnt)
	}

	st = <-ga.ExecOutput(ctx, "mount", "-t", fs, devPath, mnt)
	_, _, err := st.CheckStdio(func(_, se []byte) bool {
		return bytes.Contains(se, []byte("already mounted"))
	})
	return errors.Annotatef(err, "mount %s failed", mnt)
}

func saveFstab(ctx context.Context, ga agent.Interface, v Volume, devPath string) error {
	var blkid, err = ga.Blkid(ctx, devPath)
	if err != nil {
		return errors.Trace(err)
	}

	switch exists, err := ga.Grep(ctx, blkid, types.FstabFile); {
	case err != nil:
		return errors.Trace(err)
	case exists:
		return nil
	}

	var line = fmt.Sprintf("\nUUID=%s %s %s defaults %d %d",
		blkid, v.GetMountDir(), fs, backupDump, fsckPass)

	return ga.AppendLine(types.FstabFile, []byte(line))
}

func format(ctx context.Context, ga agent.Interface, v Volume, devPath string) error {
	switch formatted, err := isFormatted(ctx, ga, v); {
	case err != nil:
		return errors.Trace(err)
	case formatted:
		return nil
	}

	if err := fdisk(ctx, ga, devPath); err != nil {
		return errors.Trace(err)
	}

	return ga.Touch(ctx, formattedFlagPath(v))
}

func fdisk(ctx context.Context, ga agent.Interface, devPath string) error {
	var cmds = [][]string{
		{"parted", "-s", devPath, "mklabel", "gpt"},
		{"parted", "-s", devPath, "mkpart", "primary", "1049K", "--", "-1"},
		{"mkfs", "-F", "-t", fs, devPath},
	}
	return base.ExecCommands(ctx, ga, cmds)
}

func isFormatted(ctx context.Context, ga agent.Interface, v Volume) (bool, error) {
	return ga.IsFile(ctx, formattedFlagPath(v))
}

func formattedFlagPath(v Volume) string {
	return fmt.Sprintf("/etc/%s", v.Name())
}

func isAmplifying(ctx context.Context, ga agent.Interface, v Volume, devPath string) (bool, error) {
	mbs, err := getMountedBlocks(ctx, ga, v)
	if err != nil {
		return false, errors.Trace(err)
	}

	cap, err := agent.NewParted(ga, devPath).GetSize(ctx) //nolint
	if err != nil {
		return false, errors.Trace(err)
	}

	mbs = int64(float64(mbs) * (1 + configs.Conf.ResizeVolumeMinRatio))
	cap >>= 10 //nolint // in bytes, aka. 1K-blocks.

	return cap > mbs, nil
}

func getMountedBlocks(ctx context.Context, ga agent.Interface, v Volume) (int64, error) {
	df, err := ga.GetDiskfree(ctx, v.GetMountDir())
	if err != nil {
		return 0, errors.Trace(err)
	}
	return df.Blocks, nil
}

func amplify(ctx context.Context, ga agent.Interface, v Volume, devPath string) error {
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

func resetUserImage(gfs guestfs.Guestfs, distro string) error {
	if err := resetFstab(gfs); err != nil {
		return errors.Trace(err)
	}

	return resetEth0(gfs, distro)
}

func resetEth0(gfs guestfs.Guestfs, distro string) error {
	path, err := nic.GetEthFile(distro, "eth0")
	if err != nil {
		return errors.Trace(err)
	}
	return gfs.Remove(path)
}

func resetFstab(gfs guestfs.Guestfs) error {
	origFstabEntries, err := gfs.GetFstabEntries()
	if err != nil {
		return errors.Trace(err)
	}

	blkids, err := gfs.GetBlkids()
	if err != nil {
		return errors.Trace(err)
	}

	var cont string
	for dev, entry := range origFstabEntries {
		if blkids.Exists(dev) {
			cont += fmt.Sprintf("%s\n", strings.TrimSpace(entry))
		}
	}

	return gfs.Write(types.FstabFile, cont)
}

// NewMockedVolume for unit test.
func NewMockedVolume() (Volume, *gfsmocks.Guestfs) {
	gfs := &gfsmocks.Guestfs{}
	vol := local.NewSysVolume(utils.GB, "unitest-image")
	return vol, gfs
}
