package hardware

import (
	"encoding/json"
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestGPU(t *testing.T) {
	hw, err := FetchGPUInfo()
	assert.Nil(t, err)
	hwJSON, _ := json.Marshal(hw)
	t.Logf(string(hwJSON))
}

func TestCPUMem(t *testing.T) {
	cpumem, err := FetchCPUMem()
	assert.Nil(t, err)
	b, _ := json.Marshal(cpumem)

	t.Logf(string(b))
}
