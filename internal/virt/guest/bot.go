package guest

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/domain"
	"github.com/projecteru2/yavirt/internal/virt/nic"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Bot .
type Bot interface { //nolint
	Close() error
	Create() error
	Boot(ctx context.Context) error
	Shutdown(ctx context.Context, force bool) error
	Suspend() error
	Resume() error
	Undefine() error
	Migrate() error
	OpenConsole(context.Context, types.OpenConsoleFlags) (*libvirt.Console, error)
	ExecuteCommand(context.Context, []string) (output []byte, exitCode, pid int, err error)
	GetState() (libvirt.DomainState, error)
	GetUUID() (string, error)
	IsFolder(context.Context, string) (bool, error)
	RemoveAll(context.Context, string) error
	Resize(cpu int, mem int64) error
	Capture(user, name string) (*image.UserImage, error)
	AmplifyVolume(vol volume.Volume, cap int64, devPath string) error
	AttachVolume(volmod volume.Volume, devName string) (rollback func(), err error)
	BindExtraNetwork() error
	OpenFile(ctx context.Context, path, mode string) (agent.File, error)
	MakeDirectory(ctx context.Context, path string, parent bool) error
	Trylock() error
	Unlock()
	CreateSnapshot(volume.Volume) error
	CommitSnapshot(volume.Volume, string) error
	CommitSnapshotByDay(volume.Volume, int) error
	RestoreSnapshot(volume.Volume, string) error
	CheckVolume(volume.Volume) error
	RepairVolume(volume.Volume) error
}

type bot struct {
	guest *Guest
	virt  libvirt.Libvirt
	dom   domain.Domain
	ga    *agent.Agent
	flock *utils.Flock
}

func newVirtGuest(guest *Guest) (Bot, error) {
	virt, err := connectSystemLibvirt()
	if err != nil {
		return nil, errors.Trace(err)
	}

	vg := &bot{
		guest: guest,
		virt:  virt,
	}
	vg.dom = domain.New(vg.guest.Guest, vg.virt)
	vg.flock = vg.newFlock()

	vg.ga = agent.New(guest.ID, virt)

	return vg, nil
}

func connectSystemLibvirt() (libvirt.Libvirt, error) {
	return libvirt.Connect("qemu:///system")
}

func (v *bot) Close() (err error) {
	if _, err = v.virt.Close(); err != nil {
		log.WarnStack(err)
	}

	if err = v.ga.Close(); err != nil {
		log.WarnStack(err)
	}

	return
}

func (v *bot) Migrate() error {
	// TODO
	return nil
}

func (v *bot) Boot(ctx context.Context) error {
	log.Infof("Boot(%s): stage1 -> Domain boot...", v.guest.ID)
	if err := v.dom.Boot(ctx); err != nil {
		return err
	}

	log.Infof("Boot(%s): stage2 -> Waiting GA...", v.guest.ID)
	if err := v.waitGA(ctx); err != nil {
		return err
	}
	log.Infof("Boot(%s): stage3 -> Setting NICs...", v.guest.ID)
	if err := v.setupNics(); err != nil {
		return err
	}
	log.Infof("Boot(%s): stage4 -> Setting Vols...", v.guest.ID)
	if err := v.setupVols(); err != nil {
		return err
	}
	log.Infof("Boot(%s): stage5 -> Executing Batches...", v.guest.ID)
	if err := v.execBatches(); err != nil {
		return err
	}
	log.Infof("Boot(%s): stage6 -> Binding extra networks...", v.guest.ID)
	return v.BindExtraNetwork()
}

func (v *bot) waitGA(ctx context.Context) error {
	// var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GABootTimeout.Duration())
	// defer cancel()

	for i := 1; ; i++ {
		if err := v.ga.Ping(ctx); err != nil {
			select {
			case <-ctx.Done():
				return errors.Trace(err)

			default:
				log.WarnStack(err)

				i %= 10
				time.Sleep(time.Second * time.Duration(i))

				if xe := v.reloadGA(); xe != nil {
					return errors.Wrap(err, xe)
				}

				continue
			}
		}

		return nil
	}
}

func (v *bot) Shutdown(ctx context.Context, force bool) error {
	return v.dom.Shutdown(ctx, force)
}

func (v *bot) Suspend() error {
	return v.dom.Suspend()
}

func (v *bot) Resume() error {
	return v.dom.Resume()
}

func (v *bot) Undefine() error {
	var undeVols = func() (err error) {
		v.guest.rangeVolumes(func(_ int, vol volume.Volume) bool {
			err = volume.Undefine(vol)
			return err == nil
		})
		return
	}

	return utils.Invoke([]func() error{
		v.dom.Undefine,
		undeVols,
	})
}

func (v *bot) Create() (err error) {
	defer func() {
		if err != nil {
			v.deallocVols()
		}
	}()

	if err = v.allocVols(); err != nil {
		return err
	}
	if err = v.allocGuest(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (v *bot) allocVols() (err error) {
	v.guest.rangeVolumes(func(_ int, vol volume.Volume) bool {
		err = volume.Alloc(vol, v.guest.Img)
		return err == nil
	})
	return
}

func (v *bot) deallocVols() {
	v.guest.rangeVolumes(func(_ int, vol volume.Volume) bool {
		// try best behavior and ignore error
		volume.Undefine(vol) //nolint
		return true
	})
}

func (v *bot) allocGuest() error {
	if err := v.dom.Define(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (v *bot) CheckVolume(volmod volume.Volume) error {
	return volume.Check(volmod)
}

func (v *bot) RepairVolume(volmod volume.Volume) error {
	return volume.Repair(volmod)
}

func (v *bot) CreateSnapshot(volmod volume.Volume) error {
	return volume.CreateSnapshot(volmod)
}

func (v *bot) CommitSnapshot(volmod volume.Volume, snapID string) error {
	return volume.CommitSnapshot(volmod, snapID)
}

func (v *bot) CommitSnapshotByDay(volmod volume.Volume, day int) error {
	return volume.CommitSnapshotByDay(volmod, day)
}

func (v *bot) RestoreSnapshot(volmod volume.Volume, snapID string) error {
	return volume.RestoreSnapshot(volmod, snapID)
}

// AttachVolume .
func (v *bot) AttachVolume(vol volume.Volume, devName string) (rollback func(), err error) {
	dom, err := v.dom.Lookup()
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer dom.Free()

	rollback, err = volume.Create(vol)
	if err != nil {
		return nil, errors.Trace(err)
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
	st, err = dom.AttachVolume(string(buf))
	if err == nil && st == libvirt.DomainRunning && configs.Conf.Storage.InitGuestVolume {
		log.Debugf("Mount(%s): start to mount volume(%s)", v.guest.ID, vol.GetMountDir())
		err = volume.Mount(vol, v.ga, base.GetDevicePathByName(devName))
	}
	return
}

// AmplifyVolume .
func (v *bot) AmplifyVolume(vol volume.Volume, cap int64, devPath string) (err error) {
	dom, err := v.dom.Lookup()
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()
	_, err = volume.Amplify(vol, cap, dom, v.ga, devPath)

	return err
}

func (v *bot) newFlock() *utils.Flock {
	var fn = fmt.Sprintf("%s.flock", v.guest.ID)
	var fpth = filepath.Join(configs.Conf.VirtFlockDir, fn)
	return utils.NewFlock(fpth)
}

func (v *bot) execBatches() error {
	for _, bat := range configs.Conf.Batches {
		if err := v.ga.ExecBatch(bat); err != nil {
			if bat.ForceOK {
				log.ErrorStackf(err, "forced batch error")
				metrics.IncrError()
				break
			}

			log.ErrorStackf(err, "non-forced batch err")
		}
	}

	// always non-error
	return nil
}

func (v *bot) setupVols() (err error) {
	if !configs.Conf.Storage.InitGuestVolume {
		return nil
	}
	v.guest.rangeVolumes(func(sn int, vol volume.Volume) bool {
		if vol.IsSys() {
			return true
		}
		err = volume.Mount(vol, v.ga, base.GetDevicePathBySerialNumber(sn))
		return err == nil
	})
	return
}

func (v *bot) setupNics() error {
	var leng = time.Duration(len(v.guest.IPs))
	var ctx, cancel = context.WithTimeout(context.Background(), time.Minute*leng) //nolint
	defer cancel()

	if err := nic.NewNicList(v.guest.IPs, v.ga).Setup(ctx); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// BindExtraNetwork .
func (v *bot) BindExtraNetwork() error {
	dev := "eth0"
	distro := v.guest.Distro()

	if distro != types.Ubuntu {
		return nil
	}

	leng := time.Duration(v.guest.ExtraNetworks.Len())
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*leng) //nolint
	defer cancel()

	for _, netw := range v.guest.ExtraNetworks {
		// fn := fmt.Sprintf("%s.extra%d", dev, i)
		if err := nic.NewNic(netw.IP, v.ga).AddIP(ctx, dev); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (v *bot) reloadGA() error {
	if err := v.ga.Close(); err != nil {
		return errors.Trace(err)
	}

	v.ga = agent.New(v.guest.ID, v.virt)

	return nil
}

func (v *bot) OpenConsole(_ context.Context, flags types.OpenConsoleFlags) (*libvirt.Console, error) {
	err := v.dom.CheckRunning()
	if err != nil {
		return nil, err
	}
	// yavirtctl may specify devname directly
	ttyname := flags.Devname
	if ttyname == "" {
		ttyname, err = v.dom.GetConsoleTtyname()
		if err != nil {
			return nil, err
		}
	}
	c, err := v.dom.OpenConsole(ttyname, flags)
	return c, err
}

func (v *bot) ExecuteCommand(ctx context.Context, commands []string) (output []byte, exitCode, pid int, err error) {
	var prog string
	var args []string

	switch leng := len(commands); {
	case leng < 1:
		return nil, -1, -1, errors.Annotatef(errors.ErrInvalidValue, "invalid command")
	case leng > 1:
		args = commands[1:]
		fallthrough
	default:
		prog = commands[0]
	}

	select {
	case <-ctx.Done():
		err = context.Canceled
		return

	case st := <-v.ga.ExecOutput(ctx, prog, args...):
		so, se, err := st.Stdio()
		return append(so, se...), st.Code, st.Pid, err
	}
}

func (v *bot) GetUUID() (string, error) {
	return v.dom.GetUUID()
}

func (v *bot) Capture(user, name string) (*image.UserImage, error) {
	if err := v.dom.CheckShutoff(); err != nil {
		return nil, errors.Trace(err)
	}

	vol, err := v.guest.sysVolume()
	if err != nil {
		return nil, errors.Trace(err)
	}

	return volume.ConvertImage(vol, user, name)
}

func (v *bot) Trylock() error {
	return v.flock.Trylock()
}

func (v *bot) Unlock() {
	v.flock.Close()
}

func (v *bot) GetState() (libvirt.DomainState, error) {
	return v.dom.GetState()
}

func (v *bot) Resize(cpu int, mem int64) error {
	return v.dom.SetSpec(cpu, mem)
}

// OpenFile .
func (v *bot) OpenFile(ctx context.Context, path string, mode string) (agent.File, error) {
	return agent.OpenFile(ctx, v.ga, path, mode)
}

// MakeDirectory .
func (v *bot) MakeDirectory(ctx context.Context, path string, parent bool) error {
	return v.ga.MakeDirectory(ctx, path, parent)
}

// IsFolder .
func (v *bot) IsFolder(ctx context.Context, path string) (bool, error) {
	return v.ga.IsFolder(ctx, path)
}

// RemoveAll .
func (v *bot) RemoveAll(ctx context.Context, path string) error {
	return v.ga.RemoveAll(ctx, path)
}
