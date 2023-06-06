package hardware

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/jaypipes/ghw"
	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	stotypes "github.com/projecteru2/resource-storage/storage/types"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
)

type hardwareInfo struct {
	sync.Mutex
	cpumem *cpumemtypes.NodeResource
	sto    *stotypes.NodeResource
	gpus   *gputypes.NodeResource
}

var (
	hInfo hardwareInfo
)

func FetchCPUMem() (*cpumemtypes.NodeResource, error) {
	hInfo.Lock()
	defer hInfo.Unlock()

	if hInfo.cpumem == nil {
		ans := &cpumemtypes.NodeResource{}
		numa := cpumemtypes.NUMA{}
		numaMem := cpumemtypes.NUMAMemory{}

		topology, err := ghw.Topology()
		if err != nil {
			return nil, err
		}

		for _, node := range topology.Nodes {
			numaMem[fmt.Sprintf("%d", node.ID)] = node.Memory.TotalUsableBytes
			for _, core := range node.Cores {
				for _, id := range core.LogicalProcessors {
					numa[fmt.Sprintf("%d", id)] = fmt.Sprintf("%d", node.ID)
				}
			}
		}
		cpu, err := ghw.CPU()
		if err != nil {
			return nil, err
		}
		ans.CPU = float64(cpu.TotalThreads)
		ans.NUMA = numa
		mem, err := ghw.Memory()
		if err != nil {
			return nil, err
		}
		ans.Memory = mem.TotalUsableBytes
		ans.NUMAMemory = numaMem
		hInfo.cpumem = ans
	}
	return hInfo.cpumem, nil
}

func FetchStorage() (*stotypes.NodeResource, error) {
	hInfo.Lock()
	defer hInfo.Unlock()

	// update global variable
	if hInfo.sto == nil {
		ans := &stotypes.NodeResource{}
		total := int64(0)
		vols := stotypes.Volumes{}
		// use df to fetch volume information

		cmdOut, err := exec.Command("df", "-h").Output()
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(cmdOut), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) != 6 {
				continue
			}
			var size uint64
			size, err = humanize.ParseBytes(parts[1])
			if err != nil {
				return nil, err
			}
			mountPoint := parts[len(parts)-1]
			if path.Base(mountPoint) == "eru" {
				vols[mountPoint] = int64(size)
				total += int64(size)
			}
		}
		ans.Volumes = vols
		ans.Storage = total
		hInfo.sto = ans
	}
	return hInfo.sto, nil
}

// Detect all hardware resources, we don't acquire lock directly in this function
func FetchResources() (map[string][]byte, error) {
	cpumem, err := FetchCPUMem()
	if err != nil {
		return nil, err
	}
	cpumemBytes, err := json.Marshal(cpumem)
	if err != nil {
		return nil, err
	}
	storage, err := FetchStorage()
	if err != nil {
		return nil, err
	}
	stoBytes, err := json.Marshal(storage)
	if err != nil {
		return nil, err
	}
	gpus, err := FetchGPUInfo()
	if err != nil {
		return nil, err
	}
	gpusBytes, err := json.Marshal(gpus)
	if err != nil {
		return nil, err
	}
	ans := map[string][]byte{
		"cpumem":  cpumemBytes,
		"storage": stoBytes,
		"gpu":     gpusBytes,
	}
	return ans, nil
}
