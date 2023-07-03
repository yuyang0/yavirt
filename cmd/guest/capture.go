package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/pkg/errors"
)

func captureFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "user",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.BoolFlag{
			Name: "overridden",
		},
	}
}

func capture(c *cli.Context, runtime run.Runtime) error {
	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	user := c.String("user")
	name := c.String("name")
	overridden := c.Bool("overridden")
	req := types.CaptureGuestReq{
		GuestReq: types.GuestReq{
			ID: id,
		},
		User:       user,
		Name:       name,
		Overridden: overridden,
	}
	_, err := runtime.Svc.CaptureGuest(runtime.VirtContext(), req)
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s captured\n", name)

	return nil
}
