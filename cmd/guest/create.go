package guest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/utils"

	stotypes "github.com/projecteru2/resource-storage/storage/types"
	rbdtypes "github.com/yuyang0/resource-rbd/rbd/types"
)

func createFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:  "count",
			Value: 1,
		},
		&cli.IntFlag{
			Name:  "cpu",
			Value: 1,
		},
		&cli.Int64Flag{
			Name:  "memory",
			Value: utils.GB,
		},
		&cli.StringFlag{
			Name:  "storage",
			Usage: "mount info. like, --storage /data0:53687091200",
		},
		&cli.StringFlag{
			Name: "network",
		},
		&cli.StringFlag{
			Name: "dmi",
		},
		&cli.StringFlag{
			Name: "image-user",
		},
	}
}

func create(c *cli.Context, runtime run.Runtime) error {
	res, err := generateResources(c)
	if err != nil {
		return errors.Trace(err)
	}

	opts := types.GuestCreateOption{
		CPU:       c.Int("cpu"),
		Mem:       c.Int64("memory"),
		ImageName: c.Args().First(),
		ImageUser: c.String("image-user"),
		DmiUUID:   c.String("dmi"),
		Resources: res,
	}

	cnt := c.Int("count")
	network := c.String("network")

	if len(network) < 1 {
		network = runtime.Host.NetworkMode
	}

	switch {
	case len(opts.ImageName) < 1:
		return fmt.Errorf("image name is required")
	case opts.CPU < 1:
		return fmt.Errorf("--cpu is required")
	case opts.Mem < 1:
		return fmt.Errorf("--memory is required")
	case cnt < 1:
		return fmt.Errorf("--count must be greater than 0")
	case network != vnet.NetworkCalico && network != vnet.NetworkVlan:
		return fmt.Errorf("--network is invalid: %s", network)
	}

	runtime.Host.NetworkMode = network

	for i := 0; i < cnt; i++ {
		g, err := runtime.Svc.CreateGuest(runtime.VirtContext(), opts)
		if err != nil {
			return err
		}

		fmt.Printf("guest %s created\n\n", g.ID)
	}

	return nil
}

func generateResources(c *cli.Context) (ans map[string][]byte, err error) {
	ans = map[string][]byte{}
	// for storage resources
	{
		mounts := c.String("storage")
		if len(mounts) < 1 {
			return
		}
		eParmas := stotypes.EngineParams{
			Volumes: strings.Split(mounts, ","),
		}
		bs, err := json.Marshal(eParmas)
		if err != nil {
			return nil, err
		}
		ans["storage"] = bs
	}

	// for rbd resources
	{
		mounts := c.String("rbd")
		if len(mounts) < 1 {
			return
		}
		eParmas := rbdtypes.EngineParams{
			Volumes: strings.Split(mounts, ","),
		}
		bs, err := json.Marshal(eParmas)
		if err != nil {
			return nil, err
		}
		ans["rbd"] = bs
	}
	return
}
