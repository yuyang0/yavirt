package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	cli "github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	grpcserver "github.com/projecteru2/yavirt/internal/server/grpc"
	httpserver "github.com/projecteru2/yavirt/internal/server/http"
	"github.com/projecteru2/yavirt/internal/service/boar"
	"github.com/projecteru2/yavirt/internal/ver"
	"github.com/projecteru2/yavirt/internal/virt"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/store"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(ver.Version())
	}

	app := &cli.App{
		Name:    "yavirtd",
		Usage:   "yavirt daemon",
		Version: "v",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Value:   "/etc/eru/yavirtd.toml",
				Usage:   "config file path for yavirt, in toml",
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
		},
		Action: Run,
	}

	if err := app.Run(os.Args); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		os.Exit(1)
	}

	os.Exit(0)
}

func initConfig(c *cli.Context) error {
	cfg := &configs.Conf
	if err := cfg.Load([]string{c.String("config")}); err != nil {
		return err
	}
	return cfg.Prepare(c)
}

// Run .
func Run(c *cli.Context) error {
	if err := initConfig(c); err != nil {
		return err
	}
	deferSentry, err := log.Setup(configs.Conf.LogLevel, configs.Conf.LogFile, configs.Conf.LogSentry)
	if err != nil {
		return err
	}
	defer deferSentry()
	// log config
	dump, err := configs.Conf.Dump()
	if err != nil {
		return errors.Trace(err)
	}
	log.Infof("%s", dump)

	host, err := models.LoadHost()
	if err != nil {
		return err
	}

	if err := store.Setup(configs.Conf, nil); err != nil {
		return err
	}
	defer store.Close()

	br, err := boar.New(c.Context)
	if err != nil {
		return err
	}
	idgen.Setup(host.ID, time.Now())

	models.Setup()

	if err := virt.Cleanup(); err != nil {
		return errors.Trace(err)
	}

	// setup epoller
	if err := guest.SetupEpoller(); err != nil {
		return errors.Trace(err)
	}
	defer guest.GetCurrentEpoller().Close()

	quit := make(chan struct{})
	grpcSrv := grpcserver.New(br, quit)
	httpSrv := httpserver.New(br, quit)

	go prof(configs.Conf.ProfHTTPPort)

	// wait for unix signals and try to GracefulStop
	ctx, cancel := signal.NotifyContext(c.Context, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	errExitCh := make(chan struct{})
	run([]serverable{grpcSrv, httpSrv}, errExitCh)
	log.Infof("[main] all servers are running")

	select {
	case <-ctx.Done():
		log.Infof("[main] interrupt by signal")
	case <-errExitCh:
		log.Warnf("[main] server exit abnormally.")
	}
	close(quit)

	grpcSrv.Stop(false)
	httpSrv.Stop(true)

	return nil
}

func run(servers []serverable, errExitCh chan struct{}) {
	var once sync.Once
	for _, srv := range servers {
		go func(server serverable) {
			defer once.Do(func() {
				close(errExitCh)
			})
			if err := server.Serve(); err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
			}
		}(srv)
	}
}

func prof(port int) {
	var enable = strings.ToLower(os.Getenv("YAVIRTD_PPROF"))
	switch enable {
	case "":
		fallthrough
	case "0":
		fallthrough
	case "false":
		fallthrough
	case "off":
		http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil) //nolint
	default:
		return
	}
}

type serverable interface {
	Serve() error
}
