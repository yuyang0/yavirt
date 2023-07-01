package models

import (
	"fmt"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/utils/hardware"
)

// Host .
// etcd keys:
//
//	/hosts:counter
//	/hosts/<host name>
type Host struct {
	*meta.Generic

	ID          uint32 `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Subnet      int64  `json:"subnet"`
	CPU         int    `json:"cpu"`
	Memory      int64  `json:"mem"`
	Storage     int64  `json:"storage"`
	NetworkMode string `json:"network,omitempty"`
}

// LoadHost .
func LoadHost() (*Host, error) {
	cfg := &configs.Conf
	cpu, mem, sto := cfg.Host.CPU, int64(cfg.Host.Memory), int64(cfg.Host.Storage)
	// update cpu, memory, storage using hardware information
	if cpu == 0 || mem == 0 {
		cpumem, err := hardware.FetchCPUMem()
		if err != nil {
			return nil, err
		}
		if cpu == 0 {
			cpu = int(cpumem.CPU)
		}
		if mem == 0 {
			mem = cpumem.Memory
		}
	}
	if sto == 0 {
		storage, err := hardware.FetchStorage()
		if err != nil {
			return nil, err
		}
		sto = storage.Storage
	}

	host := &Host{
		Generic:     meta.NewGeneric(),
		Name:        cfg.Host.Name,
		Type:        HostVirtType,
		Subnet:      int64(cfg.Host.Subnet),
		CPU:         cpu,
		Memory:      mem,
		Storage:     sto,
		NetworkMode: cfg.Host.NetworkMode,
	}
	dec, err := netx.IPv4ToInt(configs.Conf.Host.Addr)
	if err != nil {
		return nil, err
	}
	host.ID = uint32(dec)

	return host, nil
}

// NewHost .
func NewHost() *Host {
	return &Host{Generic: meta.NewGeneric()}
}

// MetaKey .
func (h *Host) MetaKey() string {
	return meta.HostKey(h.Name)
}

func (h *Host) String() string {
	return fmt.Sprintf("%d %s subnet: %d, cpu: %d, memory: %d, storage: %d",
		h.ID, h.Name, h.Subnet, h.CPU, h.Memory, h.Storage)
}

type hostGuest struct {
	*meta.Ver
	HostName string `json:"-"`
	GuestID  string `json:"-"`
}

func newHostGuest(hostName, guestID string) hostGuest {
	return hostGuest{
		Ver:      meta.NewVer(),
		HostName: hostName,
		GuestID:  guestID,
	}
}

func (hg hostGuest) MetaKey() string {
	return meta.HostGuestKey(hg.HostName, hg.GuestID)
}
