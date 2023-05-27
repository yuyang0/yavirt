package hardware

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/jaypipes/ghw"
	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	stotypes "github.com/projecteru2/resource-storage/storage/types"
)

var (
	cpumem *cpumemtypes.NodeResource
	sto    *stotypes.NodeResource
)

func FetchCPUMem() (ans *cpumemtypes.NodeResource, err error) {
	if cpumem != nil {
		return cpumem, nil
	}
	// update global variable
	defer func() {
		if err == nil && cpumem == nil {
			cpumem = ans
		}
	}()
	ans = &cpumemtypes.NodeResource{}
	numa := cpumemtypes.NUMA{}
	numaMem := cpumemtypes.NUMAMemory{}

	topology, err := ghw.Topology()
	if err != nil {
		return
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
		return
	}
	ans.CPU = float64(cpu.TotalThreads)
	ans.NUMA = numa
	mem, err := ghw.Memory()
	if err != nil {
		return
	}
	ans.Memory = mem.TotalUsableBytes
	ans.NUMAMemory = numaMem

	return
}

func FetchStorage() (ans *stotypes.NodeResource, err error) {
	if sto != nil {
		return sto, nil
	}
	// update global variable
	defer func() {
		if err == nil && sto == nil {
			sto = ans
		}
	}()
	ans = &stotypes.NodeResource{}
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
			return
		}
		mountPoint := parts[len(parts)-1]
		if strings.Contains(mountPoint, "eru") {
			vols[mountPoint] = int64(size)
			total += int64(size)
		}
	}
	ans.Volumes = vols
	ans.Storage = total
	return
}

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
