package models

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	erucluster "github.com/projecteru2/core/cluster"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/image"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/internal/vnet/handler"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/local"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Guest indicates a virtual machine.
type Guest struct {
	*meta.Generic

	ImageName        string            `json:"img"`
	ImageUser        string            `json:"img_user,omitempty"`
	HostName         string            `json:"host"`
	CPU              int               `json:"cpu"`
	Memory           int64             `json:"mem"`
	VolIDs           []string          `json:"vols"`
	GPUAddrs         []string          `json:"gpus"`
	IPNets           meta.IPNets       `json:"ips"`
	ExtraNetworks    Networks          `json:"extra_networks,omitempty"`
	NetworkMode      string            `json:"network,omitempty"`
	EnabledCalicoCNI bool              `json:"enabled_calico_cni,omitempty"`
	NetworkPair      string            `json:"network_pair,omitempty"`
	EndpointID       string            `json:"endpoint,omitempty"`
	MAC              string            `json:"mac"`
	JSONLabels       map[string]string `json:"labels"`

	LambdaOption *LambdaOptions `json:"lambda_option,omitempty"`
	LambdaStdin  bool           `json:"lambda_stdin,omitempty"`
	Host         *Host          `json:"-"`
	Img          image.Image    `json:"-"`
	Vols         volume.Volumes `json:"-"`
	IPs          IPs            `json:"-"`

	DmiUUID string `json:"-"`
}

// Check .
func (g *Guest) Check() error {
	if g.CPU < configs.Conf.MinCPU || g.CPU > configs.Conf.MaxCPU {
		return errors.Annotatef(errors.ErrInvalidValue,
			"invalid CPU num: %d, it should be [%d, %d]",
			g.CPU, configs.Conf.MinCPU, configs.Conf.MaxCPU)
	}

	if g.Memory < configs.Conf.MinMemory || g.Memory > configs.Conf.MaxMemory {
		return errors.Annotatef(errors.ErrInvalidValue,
			"invalie memory: %d, it shoule be [%d, %d]",
			g.Memory, configs.Conf.MinMemory, configs.Conf.MaxMemory)
	}

	if lab, exists := g.JSONLabels[erucluster.LabelMeta]; exists {
		obj := map[string]any{}
		if err := utils.JSONDecode([]byte(lab), &obj); err != nil {
			return errors.Annotatef(errors.ErrInvalidValue, "'%s' should be JSON format", lab)
		}
	}

	return g.Vols.Check()
}

// MetaKey .
func (g *Guest) MetaKey() string {
	return meta.GuestKey(g.ID)
}

// Create .
func (g *Guest) Create() error {
	g.Vols.GenID()
	g.Vols.SetGuestID(g.ID)
	g.VolIDs = g.Vols.Ids()

	g.IPs.setGuestID(g.ID)

	var res = meta.Resources{g}
	res.Concate(g.Vols.Resources())
	res.Concate(meta.Resources{newHostGuest(g.HostName, g.ID)})

	return meta.Create(res)
}

// AppendIPs .
func (g *Guest) AppendIPs(ips ...meta.IP) {
	g.IPs.append(ips...)
	g.IPNets = g.IPs.ipNets()
}

// ClearVols .
func (g *Guest) ClearVols() {
	g.Vols = g.Vols[:0]
	g.VolIDs = g.VolIDs[:0]
}

// RemoveVol .
func (g *Guest) RemoveVol(volID string) {
	n := g.Vols.Len()

	for i := n - 1; i >= 0; i-- {
		if g.Vols[i].GetID() != volID {
			continue
		}

		last := n - 1
		g.Vols[last], g.Vols[i] = g.Vols[i], g.Vols[last]
		g.VolIDs[last], g.VolIDs[i] = g.VolIDs[i], g.VolIDs[last]

		n--
	}

	g.Vols = g.Vols[:n]
	g.VolIDs = g.VolIDs[:n]
}

// AppendVols .
func (g *Guest) AppendVols(vols ...volume.Volume) error {
	if g.Vols.Len()+len(vols) > configs.Conf.MaxVolumesCount {
		return errors.Annotatef(errors.ErrTooManyVolumes, "at most %d", configs.Conf.MaxVolumesCount)
	}

	var res = volume.Volumes(vols)
	res.SetHostName(g.HostName)

	g.Vols = append(g.Vols, vols...)

	g.VolIDs = append(g.VolIDs, res.Ids()...)

	return nil
}

func (g *Guest) SwitchVol(vol volume.Volume, idx int) error {
	if idx < 0 || idx >= g.Vols.Len() {
		return errors.Annotatef(errors.ErrInvalidValue, "must in range 0 to %d", g.Vols.Len()-1)
	}

	g.Vols[idx] = vol
	g.VolIDs[idx] = vol.GetID()

	return nil
}

// Load .
func (g *Guest) Load(host *Host, networkHandler handler.Handler) (err error) {
	g.Host = host

	if g.Img, err = image.LoadImage(g.ImageName, g.ImageUser); err != nil {
		return errors.Trace(err)
	}

	if g.Vols, err = volume.LoadVolumes(g.VolIDs); err != nil {
		return errors.Trace(err)
	}

	if err = g.LoadIPs(networkHandler); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadIPs .
func (g *Guest) LoadIPs(networkHandler handler.Handler) (err error) {
	for _, ipn := range g.IPNets {
		ipn.Assigned = true
	}

	g.IPs, err = networkHandler.QueryIPs(g.IPNets)

	return
}

// Resize .
func (g *Guest) Resize(cpu int, mem int64) error {
	g.CPU = cpu
	g.Memory = mem
	return g.Save()
}

// Save updates metadata to persistence store.
func (g *Guest) Save() error {
	return meta.Save(meta.Resources{g})
}

// Delete .
func (g *Guest) Delete(force bool) error {
	if err := g.SetStatus(meta.StatusDestroyed, force); err != nil {
		return errors.Trace(err)
	}

	var keys = []string{
		g.MetaKey(),
		newHostGuest(g.HostName, g.ID).MetaKey(),
	}
	keys = append(keys, g.Vols.MetaKeys()...)

	var vers = map[string]int64{g.MetaKey(): g.GetVer()}
	for _, vol := range g.Vols {
		vers[vol.MetaKey()] = vol.GetVer()
	}

	var ctx, cancel = meta.Context(context.Background())
	defer cancel()

	return store.Delete(ctx, keys, vers)
}

// SysVolume .
func (g *Guest) SysVolume() (volume.Volume, error) {
	for _, vol := range g.Vols {
		if vol.IsSys() {
			return vol, nil
		}
	}
	return nil, errors.ErrSysVolumeNotExists
}

// HealthCheck .
func (g *Guest) HealthCheck() (HealthCheck, error) {
	hcb, err := g.healthCheckBridge()
	if err != nil {
		return HealthCheck{}, errors.Trace(err)
	}
	return hcb.healthCheck(g)
}

// PublishPorts .
func (g *Guest) PublishPorts() ([]int, error) {
	hcb, err := g.healthCheckBridge()
	if err != nil {
		return []int{}, errors.Trace(err)
	}
	return hcb.publishPorts()
}

// CIDRs .
func (g *Guest) CIDRs() []string {
	cidrs := make([]string, len(g.IPNets))
	for i, ipn := range g.IPNets {
		cidrs[i] = ipn.CIDR()
	}
	return cidrs
}

// MemoryInMiB .
func (g *Guest) MemoryInMiB() int64 {
	return utils.ConvToMB(g.Memory)
}

// SocketFilepath shows the socket filepath of the guest on the host.
func (g *Guest) SocketFilepath() string {
	var fn = fmt.Sprintf("%s.sock", g.ID)
	return filepath.Join(configs.Conf.VirtSockDir, fn)
}

// NetworkPairName .
func (g *Guest) NetworkPairName() string {
	switch {
	case g.NetworkMode == vnet.NetworkCalico:
		fallthrough
	case len(g.NetworkPair) > 0:
		return g.NetworkPair

	default:
		return configs.Conf.VirtBridge
	}
}

func newGuest() *Guest {
	return &Guest{
		Generic: meta.NewGeneric(),
		Vols:    volume.Volumes{},
		IPs:     IPs{},
	}
}

// LoadGuest .
func LoadGuest(id string) (*Guest, error) {
	return manager.LoadGuest(id)
}

// CreateGuest .
func CreateGuest(opts types.GuestCreateOption, host *Host, vols []volume.Volume) (*Guest, error) {
	return manager.CreateGuest(opts, host, vols)
}

// NewGuest creates a new guest.
func NewGuest(host *Host, img image.Image) (*Guest, error) {
	return manager.NewGuest(host, img)
}

// GetNodeGuests gets all guests which belong to the node.
func GetNodeGuests(nodename string) ([]*Guest, error) {
	return manager.GetNodeGuests(nodename)
}

// GetAllGuests .
func GetAllGuests() ([]*Guest, error) {
	return manager.GetAllGuests()
}

// LoadGuest .
func (m *Manager) LoadGuest(id string) (*Guest, error) {
	g, err := m.NewGuest(nil, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}

	g.ID = id

	if err := meta.Load(g); err != nil {
		return nil, errors.Trace(err)
	}

	return g, nil
}

// CreateGuest .
func (m *Manager) CreateGuest(opts types.GuestCreateOption, host *Host, vols []volume.Volume) (*Guest, error) {
	img, err := image.LoadImage(opts.ImageName, opts.ImageUser)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var guest = newGuest()
	guest.Host = host
	guest.HostName = guest.Host.Name
	guest.NetworkMode = guest.Host.NetworkMode

	guest.Img = img
	guest.ImageName = img.GetName()
	guest.ImageUser = img.GetUser()
	// Create sys volume when user doesn't specify one
	if len(vols) == 0 || (!vols[0].IsSys()) {
		sysVol := local.NewSysVolume(img.GetSize(), img.GetName())
		if err := guest.AppendVols(sysVol); err != nil {
			return nil, errors.Trace(err)
		}
	}

	guest.ID = idgen.Next()
	guest.CPU = opts.CPU
	guest.Memory = opts.Mem
	guest.DmiUUID = opts.DmiUUID
	guest.JSONLabels = opts.Labels

	if guest.NetworkMode == vnet.NetworkCalico {
		guest.EnabledCalicoCNI = configs.Conf.EnabledCalicoCNI
	}

	if opts.Lambda {
		guest.LambdaOption = &LambdaOptions{
			Cmd:       opts.Cmd,
			CmdOutput: nil,
		}
	}
	log.Debugf("Resources: %v", opts.Resources)
	if bs, ok := opts.Resources["gpu"]; ok {
		var eParams types.GPUEngineParams
		if err := json.Unmarshal(bs, &eParams); err != nil {
			return nil, errors.Trace(err)
		}
		guest.GPUAddrs = eParams.Addrs
	}

	if err := guest.AppendVols(vols...); err != nil {
		return nil, errors.Trace(err)
	}

	if err := guest.Check(); err != nil {
		return nil, errors.Trace(err)
	}

	if err := guest.Create(); err != nil {
		return nil, errors.Trace(err)
	}

	return guest, nil
}

// NewGuest creates a new guest.
func (m *Manager) NewGuest(host *Host, img image.Image) (*Guest, error) {
	var guest = newGuest()

	if host != nil {
		guest.Host = host
		guest.HostName = guest.Host.Name
		guest.NetworkMode = guest.Host.NetworkMode
	}

	if img != nil {
		guest.Img = img
		guest.ImageName = img.GetName()
		guest.ImageUser = img.GetUser()

		sysVol := local.NewSysVolume(img.GetSize(), img.GetName())
		if err := guest.AppendVols(sysVol); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return guest, nil
}

// GetNodeGuests gets all guests which belong to the node.
func (m *Manager) GetNodeGuests(nodename string) ([]*Guest, error) {
	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	data, _, err := store.GetPrefix(ctx, meta.HostGuestsPrefix(nodename), 0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	guests := []*Guest{}
	for key := range data {
		parts := strings.Split(key, "/")
		gid := parts[len(parts)-1]
		if len(gid) < 1 {
			continue
		}

		g, err := m.LoadGuest(gid)
		if err != nil {
			return nil, errors.Trace(err)
		}

		guests = append(guests, g)
	}

	return guests, nil
}

// GetAllGuests .
func (m *Manager) GetAllGuests() ([]*Guest, error) {
	var ctx, cancel = meta.Context(context.Background())
	defer cancel()

	var data, vers, err = store.GetPrefix(ctx, meta.GuestsPrefix(), 0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var guests = []*Guest{}

	for key, val := range data {
		var ver, exists = vers[key]
		if !exists {
			return nil, errors.Annotatef(errors.ErrKeyBadVersion, key)
		}

		var g = newGuest()
		if err := utils.JSONDecode(val, g); err != nil {
			return nil, errors.Trace(err)
		}

		g.SetVer(ver)
		guests = append(guests, g)
	}

	return guests, nil
}
