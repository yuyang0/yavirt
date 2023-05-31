package guest

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
)

func destroyFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
		},
	}
}

func stopFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
		},
	}
}

func start(c *cli.Context, runtime run.Runtime) error {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf("Starting guest %s", id)

	if err := runtime.Guest.Start(runtime.VirtContext(), id); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s started\n", id)

	return nil
}

func suspend(c *cli.Context, runtime run.Runtime) error {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf("Suspending guest %s", id)
	if err := runtime.Guest.Suspend(runtime.VirtContext(), id); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s suspended\n", id)

	return nil
}

func resume(c *cli.Context, runtime run.Runtime) error {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf("Resuming guest %s", id)
	if err := runtime.Guest.Resume(runtime.VirtContext(), id); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s resumed\n", id)

	return nil
}

func stop(c *cli.Context, runtime run.Runtime) error {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf("Stopping guest %s", id)
	if err := runtime.Guest.Stop(runtime.VirtContext(), id, c.Bool("force")); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s stopped\n", id)

	return nil
}

func destroy(c *cli.Context, runtime run.Runtime) (err error) {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf("Destroying guest %s", id)

	done, err := runtime.Guest.Destroy(runtime.VirtContext(), id, c.Bool("force"))
	if err != nil {
		return errors.Trace(err)
	}

	select {
	case err = <-done:
	case <-time.After(time.Minute):
		err = errors.ErrTimeout
	}
	if err != nil {
		return errors.Trace(err)
	}
	fmt.Printf("%s destroyed\n", id)

	return nil
}
