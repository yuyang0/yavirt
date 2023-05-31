package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/guest"
	"github.com/projecteru2/yavirt/cmd/image"
	"github.com/projecteru2/yavirt/cmd/maint"
	"github.com/projecteru2/yavirt/cmd/network"
	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/server"
	"github.com/projecteru2/yavirt/internal/ver"
	"github.com/projecteru2/yavirt/pkg/errors"
)

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(ver.Version())
	}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Value:   "/etc/eru/yavirtd.toml",
				Usage:   "config file path for yavirt, in yaml",
				EnvVars: []string{"ERU_YAVIRT_CONFIG_PATH"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "",
				Usage:   "set log level",
				EnvVars: []string{"ERU_YAVIRT_LOG_LEVEL"},
			},
			&cli.StringSliceFlag{
				Name:    "core-addrs",
				Value:   cli.NewStringSlice(),
				Usage:   "core addresses",
				EnvVars: []string{"ERU_YAVIRT_CORE_ADDRS"},
			},
			&cli.StringFlag{
				Name:    "core-username",
				Value:   "",
				Usage:   "core username",
				EnvVars: []string{"ERU_YAVIRT_CORE_USERNAME"},
			},
			&cli.StringFlag{
				Name:    "core-password",
				Value:   "",
				Usage:   "core password",
				EnvVars: []string{"ERU_YAVIRT_CORE_PASSWORD"},
			},
			&cli.StringFlag{
				Name:    "hostname",
				Value:   "",
				Usage:   "change hostname",
				EnvVars: []string{"ERU_HOSTNAME", "HOSTNAME"},
			},
			&cli.IntFlag{
				Name:    "timeout",
				Value:   300,
				Usage:   "command timeout",
				EnvVars: []string{"ERU_YAVIRT_CMD_TIMEOUT"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "info",
				Action: run.Run(info),
			},
			guest.Command(),
			image.Command(),
			network.Command(),
			maint.Command(),
		},

		Version: "v",
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(errors.Stack(err))
	}
}

func info(c *cli.Context, _ run.Runtime) (err error) {
	cfg := &configs.Conf

	if err := cfg.Load([]string{c.String("config")}); err != nil {
		return errors.Trace(err)
	}
	if err := cfg.Prepare(c); err != nil {
		return err
	}
	svc, err := server.SetupYavirtdService()
	if err != nil {
		return err
	}
	info, err := svc.Info()
	if err != nil {
		return err
	}
	ans := map[string]string{
		"addr":     cfg.Host.Addr,
		"hostname": cfg.Host.Name,
	}
	for name, res := range info.Resources {
		ans[name] = string(res)
	}
	b, err := json.MarshalIndent(ans, "", "\t")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", string(b))
	return nil
}
