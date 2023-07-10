package hardware

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jaypipes/ghw"

	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
)

var execCommand = exec.Command

type Class string

const (
	// used to refer to the whole machine (laptop, server, desktop computer)
	Systems Class = "system"
	// internal bus converter (PCI-to-PCI brige, AGP bridge, PCMCIA controler, host bridge)
	Bridge Class = "bridge"
	// memory bank that can contain data, executable code, etc.
	// RAM, BIOS, firmware, extension ROM
	Memory Class = "memory"
	// execution processor	 (CPUs, RAID controller on a SCSI bus)
	Processor Class = "processor"
	// memory address range extension ROM, video memory
	Address Class = "address"
	// storage controller	(SCSI controller, IDE controller)
	Storage Class = "storage"
	// random-access storage device discs, optical storage (CD-ROM, DVD±RW...)
	Disk Class = "disk"
	// sequential-access storage device (DAT, DDS)
	Tape Class = "tape"
	// device-connecting bus (USB, SCSI, Firewire)
	Bus Class = "bus"
	// network interface (Ethernet, FDDI, WiFi, Bluetooth)
	Network Class = "network"
	// display adapter (EGA/VGA, UGA...)
	Display Class = "display"
	// user input device (keyboards, mice, joysticks...)
	Input Class = "input"
	// printing device (printer, all-in-one)
	Printer Class = "printer"
	// audio/video device (sound card, TV-output card, video acquisition card)
	Multimedia Class = "multimedia"
	// line communication device (serial ports, modem)
	Communication Class = "communication"
	// energy source (power supply, internal battery)
	Power Class = "power"
	// disk volume	(filesystem, swap, etc.)
	Volume Class = "volume"
	// generic device (used when no pre-defined class is suitable)
	Generic Class = "generic"
	// Print everything
	All Class = "all"
)

type MemoryBank struct {
	Descr   string `json:"description"`
	Size    string `json:"size"`
	Vendor  string `json:"vendor"`
	Product string `json:"product"`
}

type MemoryInfo struct {
	Descr string       `json:"description"`
	Banks []MemoryBank `json:"banks"`
}

type CPUCache struct {
	Descr string `json:"description"`
	Size  string `json:"size"`
}

type CPU struct {
	Descr   string     `json:"description"`
	Version string     `json:"version"`
	Size    string     `json:"size"`
	Width   string     `json:"width"`
	Cache   []CPUCache `json:"cache"`
}

type DiskVolume struct {
	Descr       string `json:"description"`
	LogicalName string `json:"logical_name"`
	Size        string `json:"size"`
}

type DiskInfo struct {
	Descr   string       `json:"description"`
	Product string       `json:"product"`
	Serial  string       `json:"serial"`
	Size    string       `json:"size"`
	Volumes []DiskVolume `json:"volumes"`
}

type Firmware struct {
	Descr    string `json:"description"`
	Vendor   string `json:"vendor"`
	Date     string `json:"date"`
	Size     string `json:"size"`
	Capacity string `json:"capacity"`
}

type Core struct {
	Descr    string       `json:"description"`
	Firmware []Firmware   `json:"firmware"`
	CPU      []CPU        `json:"cpu"`
	Memory   []MemoryInfo `json:"memory"`
	Disks    []DiskInfo   `json:"disk"`
}

type Hardware struct {
	Descr   string `json:"description"`
	Product string `json:"product"`
	Serial  string `json:"serial"`
	Vendor  string `json:"vendor"`
	Core    Core   `json:"core"`
}

func FetchGPUInfo() (*gputypes.NodeResource, error) {
	hInfo.Lock()
	defer hInfo.Unlock()

	if hInfo.gpus != nil {
		return hInfo.gpus, nil
	}

	pci, err := ghw.PCI()
	if err != nil {
		return nil, err
	}

	cmdOut, err := execCommand("lshw", "-quiet", "-json", "-C", "display").Output()
	if err != nil {
		return nil, err
	}
	params := []map[string]any{}
	if err = json.Unmarshal(cmdOut, &params); err != nil {
		return nil, err
	}
	ans := gputypes.NewNodeResource(nil)
	for _, param := range params {
		var addr string
		if businfoRaw, ok := param["businfo"]; ok && businfoRaw != nil {
			addr = strings.Split(businfoRaw.(string), "@")[1]
		} else if handleRaw, ok := param["handle"]; ok && handleRaw != nil {
			handle := handleRaw.(string) //nolint
			idx := strings.Index(handle, ":")
			addr = handle[idx+1:]
		}
		if addr == "" {
			return nil, errors.New("Can't fetch PCI address")
		}
		deviceInfo := pci.GetDevice(addr)
		var numa string
		if deviceInfo != nil && deviceInfo.Node != nil {
			numa = fmt.Sprintf("%d", deviceInfo.Node.ID)
		}
		info := gputypes.GPUInfo{
			Address: addr,
			Product: param["product"].(string),
			Vendor:  param["vendor"].(string),
			NumaID:  numa,
		}
		ans.GPUMap[addr] = info
	}
	hInfo.gpus = ans
	return hInfo.gpus, nil
}
