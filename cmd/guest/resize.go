package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/errors"
)

func resizeFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name: "volumes",
		},
		&cli.IntFlag{
			Name: "cpu",
		},
		&cli.Int64Flag{
			Name: "memory",
		},
	}
}

func resize(c *cli.Context, runtime run.Runtime) (err error) {
	vs := map[string]*models.Volume{}
	for _, raw := range c.StringSlice("volumes") {
		vol, err := models.NewDataVolumeFromStr(raw)
		if err != nil {
			return errors.Trace(err)
		}
		vs[vol.MountDir] = vol
	}

	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	cpu := c.Int("cpu")
	mem := c.Int64("memory")
	if err = runtime.Guest.Resize(runtime.VirtContext(), id, cpu, mem, vs); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s resized\n", id)

	return
}
