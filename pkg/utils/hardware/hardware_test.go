package hardware

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
)

var fakeExecResult = `[
  {
    "id" : "display",
    "class" : "display",
    "claimed" : true,
    "handle" : "PCI:5678:00:00.0",
    "description" : "3D controller",
    "product" : "Microsoft Corporation",
    "vendor" : "Microsoft Corporation",
    "physid" : "9",
    "businfo" : "pci@1234:00:00.0",
    "version" : "00",
    "width" : 32,
    "clock" : 33000000,
    "configuration" : {
      "driver" : "dxgkrnl",
      "latency" : "0"
    },
    "capabilities" : {
      "bus_master" : "bus mastering",
      "cap_list" : "PCI capabilities listing"
    }
  }
]`

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{
		"GO_WANT_HELPER_PROCESS=1",
		fmt.Sprintf("GOCOVERDIR=%s", os.TempDir()),
	}
	return cmd
}

func TestGPU(t *testing.T) {
	execCommand = fakeExecCommand
	defer func() { execCommand = exec.Command }()

	hw, err := FetchGPUInfo()
	assert.Nil(t, err)
	assert.Equal(t, hw.Len(), 1)
	gInfo, ok := hw.GPUMap["1234:00:00.0"]
	assert.True(t, ok)
	assert.Equal(t, "Microsoft Corporation", gInfo.Product)
	hwJSON, _ := json.Marshal(hw)
	t.Logf(string(hwJSON))
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// some code here to check arguments perhaps?
	fmt.Fprintf(os.Stdout, fakeExecResult)
	os.Exit(0)
}

func TestCPUMem(t *testing.T) {
	cpumem, err := FetchCPUMem()
	assert.Nil(t, err)
	b, _ := json.Marshal(cpumem)

	t.Logf(string(b))
}
