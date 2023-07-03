package rbd

import (
	"bytes"
	"fmt"
	"strings"

	_ "embed"

	"text/template"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/virt/guestfs/gfsx"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	libguestfs "github.com/projecteru2/yavirt/third_party/guestfs"
	rbdtypes "github.com/yuyang0/resource-rbd/rbd/types"
)

var (
	//go:embed templates/disk.xml
	diskXML string
	tmpl    *template.Template
)

type Volume struct {
	base.Volume            `mapstructure:",squash"`
	rbdtypes.VolumeBinding `mapstructure:",squash"`
}

func New() *Volume {
	return &Volume{
		Volume: *base.New(),
	}
}
func NewFromStr(ss string) (*Volume, error) {
	vb, err := rbdtypes.NewVolumeBinding(ss)
	if err != nil {
		return nil, err
	}
	return &Volume{
		Volume:        *base.New(),
		VolumeBinding: *vb,
	}, nil
}

func (v *Volume) Name() string {
	return fmt.Sprintf("rbd-%s", v.ID)
}

func (v *Volume) GetSize() int64 {
	return v.SizeInBytes
}

func (v *Volume) GetMountDir() string {
	return v.Destination
}

func (v *Volume) IsSys() bool {
	return strings.Contains(v.Flags, "s")
}

func (v *Volume) Prepare(image.Image) error {
	return nil
}

func (v *Volume) Check() error {
	return nil
}

func (v *Volume) Repair() error {
	return nil
}

func (v *Volume) GenerateXML(i int) ([]byte, error) {
	devName := base.GetDeviceName(i)
	return v.GenerateXMLWithDevName(devName)
}

func (v *Volume) GenerateXMLWithDevName(devName string) ([]byte, error) {
	// prepare monitor addresses
	cephMonitorAddrs := []map[string]string{}
	for _, addr := range configs.Conf.Storage.Ceph.MonitorAddrs {
		parts := strings.Split(addr, ":")
		d := map[string]string{
			"host": parts[0],
			"port": parts[1],
		}
		cephMonitorAddrs = append(cephMonitorAddrs, d)
	}

	args := map[string]any{
		"source":       v.GetSource(),
		"dev":          devName,
		"monitorAddrs": cephMonitorAddrs,
		"username":     configs.Conf.Storage.Ceph.Username,
		"secretUUID":   configs.Conf.Storage.Ceph.SecretUUID,
	}
	if tmpl == nil {
		t, err := template.New("rbd-vol-tpl").Parse(diskXML)
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

func (v *Volume) Cleanup() error {
	return nil
}

func (v *Volume) CaptureImage(user, name string) (uimg *image.UserImage, err error) { //nolint
	return
}

func (v *Volume) Save() error {
	return meta.Save(meta.Resources{v})
}

func (v *Volume) Amplify(delta int64, dom libvirt.Domain, ga agent.Interface, devPath string) (normDelta int64, err error) { //nolint
	return
}

func (v *Volume) Lock() error {
	return nil
}

func (v *Volume) Unlock() {}

func (v *Volume) NewSnapshotAPI() base.SnapshotAPI {
	return newSnapshotAPI(v)
}

func (v *Volume) GetGfx() (guestfs.Guestfs, error) {
	opts := &libguestfs.OptargsAdd_drive{
		Readonly_is_set: false,
		Format_is_set:   true,
		Format:          "raw",
		Protocol_is_set: true,
		Protocol:        "rbd",
		Server_is_set:   true,
		// TODO add servers, user and secret for ceph
		Server:          []string{},
		Username_is_set: true,
		// User: "eru",
		Secret_is_set: true,
		Secret:        "",
	}
	return gfsx.NewFromOpts(v.GetSource(), opts)
}
